//go:build integration

package tb

import (
	"context"

	"cogni/internal/tbutil"
	"cogni/pkg/ratelimiter"
)

// DebugPendingDebits returns the pending debit total for a limit account.
func (b *Backend) DebugPendingDebits(ctx context.Context, key ratelimiter.LimitKey) (uint64, error) {
	account, err := b.lookupAccount(ctx, tbutil.LimitAccountID(key))
	if err != nil {
		return 0, err
	}
	return tbutil.Uint128ToUint64(account.DebitsPending), nil
}
