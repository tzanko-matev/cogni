package tb

import (
	"context"
	"fmt"
	"time"

	"cogni/internal/tbutil"
	"cogni/pkg/ratelimiter"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
)

const decreaseCheckInterval = 200 * time.Millisecond

// decreaseLoop periodically attempts to apply capacity decreases.
func (b *Backend) decreaseLoop(ctx context.Context) {
	ticker := time.NewTicker(decreaseCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.applyPendingDecreases(ctx)
		}
	}
}

// applyPendingDecreases scans for decreasing limits and applies them.
func (b *Backend) applyPendingDecreases(ctx context.Context) {
	b.mu.Lock()
	states := make([]ratelimiter.LimitState, 0, len(b.states))
	for _, state := range b.states {
		if state.Status == ratelimiter.LimitStatusDecreasing {
			states = append(states, state)
		}
	}
	b.mu.Unlock()

	for _, state := range states {
		_ = b.tryApplyDecrease(ctx, state)
	}
}

// tryApplyDecrease applies a pending decrease when capacity allows.
func (b *Backend) tryApplyDecrease(ctx context.Context, state ratelimiter.LimitState) error {
	if state.Status != ratelimiter.LimitStatusDecreasing {
		return nil
	}
	current := state.Definition.Capacity
	target := state.PendingDecreaseTo
	if target == 0 || target >= current {
		return nil
	}
	account, err := b.lookupAccount(ctx, tbutil.LimitAccountID(state.Definition.Key))
	if err != nil {
		return err
	}
	available := accountAvailable(account)
	delta := current - target
	if available < delta {
		return nil
	}
	transfer := tbtypes.Transfer{
		ID:              tbutil.DecreaseTransferID(state.Definition.Key, target),
		DebitAccountID:  tbutil.LimitAccountID(state.Definition.Key),
		CreditAccountID: tbutil.OperatorAccountID(),
		Ledger:          ledgerLimits,
		Code:            codeLimit,
		Amount:          delta,
	}
	result, err := b.submitTransfers(ctx, []tbtypes.Transfer{transfer})
	if err != nil {
		return err
	}
	if err := firstTransferError(result.Errors); err != nil {
		return err
	}

	updated := ratelimiter.LimitState{
		Definition:        state.Definition,
		Status:            ratelimiter.LimitStatusActive,
		PendingDecreaseTo: 0,
	}
	updated.Definition.Capacity = target
	b.mu.Lock()
	b.states[state.Definition.Key] = updated
	b.mu.Unlock()
	if b.registry != nil {
		b.registry.Put(updated)
		if b.registryPath != "" {
			if err := b.registry.Save(b.registryPath); err != nil {
				return fmt.Errorf("save registry: %w", err)
			}
		}
	}
	return nil
}
