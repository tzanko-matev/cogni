package tb

import (
	"context"

	"cogni/internal/tbutil"
	"cogni/pkg/ratelimiter"
)

// DebtForKey returns the recorded debt for a limit key.
func (b *Backend) DebtForKey(ctx context.Context, key ratelimiter.LimitKey) (uint64, error) {
	account, err := b.lookupAccount(ctx, tbutil.DebtAccountID(key))
	if err != nil {
		return 0, err
	}
	return tbutil.Uint128ToUint64(account.DebitsPosted), nil
}
