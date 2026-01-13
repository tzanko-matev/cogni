package tb

import (
	"context"
	"fmt"
	"time"

	"cogni/internal/tbutil"
	"cogni/pkg/ratelimiter"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// Complete reconciles pending transfers based on actual usage.
func (b *Backend) Complete(ctx context.Context, req ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	if req.LeaseID == "" {
		return ratelimiter.CompleteResponse{Ok: false, Error: invalidRequestError}, nil
	}
	now := b.nowFn()

	b.mu.Lock()
	state, ok := b.leases[req.LeaseID]
	if ok {
		delete(b.leases, req.LeaseID)
	}
	defs := map[ratelimiter.LimitKey]ratelimiter.LimitDefinition{}
	if ok {
		for _, req := range state.Requirements {
			if state, found := b.states[req.Key]; found {
				defs[req.Key] = state.Definition
			}
		}
		for _, actual := range req.Actuals {
			if state, found := b.states[actual.Key]; found {
				defs[actual.Key] = state.Definition
			}
		}
	}
	b.mu.Unlock()
	if !ok {
		return ratelimiter.CompleteResponse{Ok: true}, nil
	}

	if err := b.releaseConcurrency(ctx, state, defs); err != nil {
		return ratelimiter.CompleteResponse{Ok: false, Error: "backend_error"}, err
	}
	if err := b.reconcileRolling(ctx, state, defs, req.Actuals, now); err != nil {
		return ratelimiter.CompleteResponse{Ok: false, Error: "backend_error"}, err
	}
	return ratelimiter.CompleteResponse{Ok: true}, nil
}

// releaseConcurrency voids pending transfers for concurrency limits.
func (b *Backend) releaseConcurrency(ctx context.Context, state LeaseState, defs map[ratelimiter.LimitKey]ratelimiter.LimitDefinition) error {
	var transfers []tbtypes.Transfer
	for _, req := range state.Requirements {
		def, ok := defs[req.Key]
		if !ok || def.Kind != ratelimiter.KindConcurrency {
			continue
		}
		amount := state.ReservedAmounts[req.Key]
		if amount == 0 {
			continue
		}
		transfers = append(transfers, voidTransfer(state.LeaseID, req.Key, amount, false))
	}
	if len(transfers) == 0 {
		return nil
	}
	result, err := b.submitTransfers(ctx, transfers)
	if err != nil {
		return err
	}
	if hasNonIgnorableErrors(result.Errors) {
		return fmt.Errorf("concurrency release failed")
	}
	return nil
}

// reconcileRolling applies rolling-limit reconciliation based on actuals.
func (b *Backend) reconcileRolling(ctx context.Context, state LeaseState, defs map[ratelimiter.LimitKey]ratelimiter.LimitDefinition, actuals []ratelimiter.Actual, now time.Time) error {
	for _, actual := range actuals {
		def, ok := defs[actual.Key]
		if !ok || def.Kind != ratelimiter.KindRolling {
			continue
		}
		reserved := state.ReservedAmounts[actual.Key]
		if reserved == 0 {
			continue
		}
		switch {
		case actual.ActualAmount < reserved:
			if err := b.reconcileUnderuse(ctx, state, def, actual.ActualAmount, now); err != nil {
				return err
			}
		case actual.ActualAmount > reserved:
			if err := b.reconcileOveruse(ctx, state, def, actual.ActualAmount-reserved, now); err != nil {
				return err
			}
		}
	}
	return nil
}

// reconcileUnderuse voids and re-reserves when actual usage is lower than reserved.
func (b *Backend) reconcileUnderuse(ctx context.Context, state LeaseState, def ratelimiter.LimitDefinition, actual uint64, now time.Time) error {
	remaining := remainingWindowSeconds(state.ReservedAtUnix, now, def.WindowSeconds)
	void := voidTransfer(state.LeaseID, def.Key, state.ReservedAmounts[def.Key], actual != 0)
	if actual == 0 {
		result, err := b.submitTransfers(ctx, []tbtypes.Transfer{void})
		if err != nil {
			return err
		}
		if hasNonIgnorableErrors(result.Errors) {
			return fmt.Errorf("void transfer failed")
		}
		return nil
	}
	rereserve := pendingTransfer(state.LeaseID, def.Key, actual, remaining)
	result, err := b.submitTransfers(ctx, []tbtypes.Transfer{void, rereserve})
	if err != nil {
		return err
	}
	if hasNonIgnorableErrors(result.Errors) {
		return fmt.Errorf("reconcile underuse failed")
	}
	return nil
}

// reconcileOveruse attempts to reserve extra usage or records debt.
func (b *Backend) reconcileOveruse(ctx context.Context, state LeaseState, def ratelimiter.LimitDefinition, diff uint64, now time.Time) error {
	if diff == 0 {
		return nil
	}
	remaining := remainingWindowSeconds(state.ReservedAtUnix, now, def.WindowSeconds)
	rereserve := pendingTransfer(state.LeaseID, def.Key, diff, remaining)
	result, err := b.submitTransfers(ctx, []tbtypes.Transfer{rereserve})
	if err != nil {
		return err
	}
	if len(result.Errors) == 0 {
		return nil
	}
	if isOverageDenied(result.Errors) {
		if def.Overage != ratelimiter.OverageDebt {
			return nil
		}
		debt := debtTransfer(state.LeaseID, def.Key, diff)
		debtResult, err := b.submitTransfers(ctx, []tbtypes.Transfer{debt})
		if err != nil {
			return err
		}
		if hasNonIgnorableErrors(debtResult.Errors) {
			return fmt.Errorf("debt transfer failed")
		}
		return nil
	}
	if hasNonIgnorableErrors(result.Errors) {
		return fmt.Errorf("overage reserve failed")
	}
	return nil
}

// voidTransfer builds a transfer to void a pending reservation.
func voidTransfer(leaseID string, key ratelimiter.LimitKey, amount uint64, linked bool) tbtypes.Transfer {
	flags := tbtypes.TransferFlags{VoidPendingTransfer: true, Linked: linked}
	return tbtypes.Transfer{
		ID:              tbutil.VoidTransferID(leaseID, key),
		DebitAccountID:  tbutil.LimitAccountID(key),
		CreditAccountID: tbutil.OperatorAccountID(),
		Amount:          tbutil.Uint128FromUint64(amount),
		Ledger:          ledgerLimits,
		Code:            codeLimit,
		Flags:           flags.ToUint16(),
		PendingID:       tbutil.ReserveTransferID(leaseID, key),
	}
}

// pendingTransfer builds a pending transfer for rolling reconciliation.
func pendingTransfer(leaseID string, key ratelimiter.LimitKey, amount uint64, timeout int) tbtypes.Transfer {
	return tbtypes.Transfer{
		ID:              tbutil.RereserveTransferID(leaseID, key),
		DebitAccountID:  tbutil.LimitAccountID(key),
		CreditAccountID: tbutil.OperatorAccountID(),
		Amount:          tbutil.Uint128FromUint64(amount),
		Ledger:          ledgerLimits,
		Code:            codeLimit,
		Flags:           tbtypes.TransferFlags{Pending: true}.ToUint16(),
		Timeout:         uint32(timeout),
	}
}

// debtTransfer builds a posted transfer for debt accounting.
func debtTransfer(leaseID string, key ratelimiter.LimitKey, amount uint64) tbtypes.Transfer {
	return tbtypes.Transfer{
		ID:              tbutil.DebtTransferID(leaseID, key),
		DebitAccountID:  tbutil.DebtAccountID(key),
		CreditAccountID: tbutil.OperatorAccountID(),
		Amount:          tbutil.Uint128FromUint64(amount),
		Ledger:          ledgerLimits,
		Code:            codeLimit,
	}
}
