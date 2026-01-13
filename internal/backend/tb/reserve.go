package tb

import (
	"context"
	"fmt"
	"time"

	"cogni/internal/tbutil"
	"cogni/pkg/ratelimiter"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
)

const invalidRequestError = "invalid_request"

// Reserve attempts to create a linked chain of pending transfers.
func (b *Backend) Reserve(ctx context.Context, req ratelimiter.ReserveRequest, now time.Time) (ratelimiter.ReserveResponse, error) {
	if req.LeaseID == "" || len(req.Requirements) == 0 {
		return ratelimiter.ReserveResponse{Allowed: false, Error: invalidRequestError}, nil
	}
	if now.IsZero() {
		now = b.nowFn()
	}

	b.mu.Lock()
	if state, ok := b.leases[req.LeaseID]; ok {
		b.mu.Unlock()
		if requirementsEqual(state.Requirements, req.Requirements) {
			return ratelimiter.ReserveResponse{Allowed: true, ReservedAtUnixMs: state.ReservedAtUnix}, nil
		}
		return ratelimiter.ReserveResponse{Allowed: false, Error: invalidRequestError}, nil
	}
	defs := make([]ratelimiter.LimitDefinition, len(req.Requirements))
	for i, r := range req.Requirements {
		state, ok := b.states[r.Key]
		if !ok {
			b.mu.Unlock()
			return ratelimiter.ReserveResponse{Allowed: false, Error: "unknown_limit_key:" + string(r.Key)}, nil
		}
		if state.Status == ratelimiter.LimitStatusDecreasing {
			b.mu.Unlock()
			return ratelimiter.ReserveResponse{
				Allowed:      false,
				RetryAfterMs: b.retryPolicy.Decreasing.FixedMs,
				Error:        "limit_decreasing:" + string(r.Key),
			}, nil
		}
		defs[i] = state.Definition
	}
	b.mu.Unlock()

	transfers := make([]tbtypes.Transfer, len(req.Requirements))
	for i, r := range req.Requirements {
		def := defs[i]
		timeout := timeoutSeconds(def)
		flags := tbtypes.TransferFlags{Pending: true}
		if i < len(req.Requirements)-1 {
			flags.Linked = true
		}
		transfers[i] = tbtypes.Transfer{
			ID:              tbutil.ReserveTransferID(req.LeaseID, r.Key),
			DebitAccountID:  tbutil.LimitAccountID(r.Key),
			CreditAccountID: tbutil.OperatorAccountID(),
			Amount:          r.Amount,
			Ledger:          ledgerLimits,
			Code:            codeLimit,
			Flags:           flags,
			Timeout:         uint32(timeout),
		}
	}

	result, err := b.submitTransfers(ctx, transfers)
	if err != nil {
		return ratelimiter.ReserveResponse{}, err
	}
	decision, retryAfter, err := b.evaluateReserveErrors(req, defs, result.Errors)
	if err != nil {
		return ratelimiter.ReserveResponse{}, err
	}
	if !decision {
		return ratelimiter.ReserveResponse{Allowed: false, RetryAfterMs: retryAfter}, nil
	}

	lease := LeaseState{
		LeaseID:         req.LeaseID,
		ReservedAtUnix:  now.UnixMilli(),
		Requirements:    req.Requirements,
		ReservedAmounts: indexByKey(req.Requirements),
	}
	b.mu.Lock()
	b.leases[req.LeaseID] = lease
	b.mu.Unlock()
	for _, r := range req.Requirements {
		b.denyTracker.Decay(r.Key)
	}
	return ratelimiter.ReserveResponse{Allowed: true, ReservedAtUnixMs: lease.ReservedAtUnix}, nil
}

// evaluateReserveErrors converts TB transfer errors into allow/deny decisions.
func (b *Backend) evaluateReserveErrors(req ratelimiter.ReserveRequest, defs []ratelimiter.LimitDefinition, errors map[int]tbtypes.CreateTransferResult) (bool, int, error) {
	if len(errors) == 0 {
		return true, 0, nil
	}
	maxRetry := 0
	hasDenied := false
	hasLinked := false
	hasUnexpected := false
	onlyExists := true
	for index, result := range errors {
		switch result {
		case tbtypes.TransferExceedsCredits, tbtypes.TransferExceedsDebits, tbtypes.TransferIDAlreadyFailed:
			hasDenied = true
			onlyExists = false
			if index >= 0 && index < len(req.Requirements) {
				key := req.Requirements[index].Key
				def := defs[index]
				streak := b.denyTracker.Increment(key)
				retry := RetryAfterMs(def, streak, b.retryPolicy, b.retryRand.Jitter)
				if retry > maxRetry {
					maxRetry = retry
				}
			}
		case tbtypes.TransferExists:
		case tbtypes.TransferLinkedEventFailed:
			hasLinked = true
			onlyExists = false
		default:
			hasUnexpected = true
			onlyExists = false
		}
	}
	if hasDenied {
		return false, maxRetry, nil
	}
	if onlyExists {
		return true, 0, nil
	}
	if hasUnexpected || hasLinked {
		return false, 0, fmt.Errorf("reserve transfer error")
	}
	return false, 0, fmt.Errorf("reserve transfer error")
}

// timeoutSeconds selects the timeout based on limit kind.
func timeoutSeconds(def ratelimiter.LimitDefinition) int {
	switch def.Kind {
	case ratelimiter.KindRolling:
		return def.WindowSeconds
	case ratelimiter.KindConcurrency:
		return def.TimeoutSeconds
	default:
		return 0
	}
}
