package tb

import (
	"context"
	"fmt"
	"time"

	"cogni/internal/tbutil"
	"cogni/pkg/ratelimiter"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// submitTransfers sends transfers through the submitter and waits for results.
func (b *Backend) submitTransfers(ctx context.Context, transfers []tbtypes.Transfer) (tbutil.WorkResult, error) {
	if len(transfers) == 0 {
		return tbutil.WorkResult{}, nil
	}
	item := tbutil.WorkItem{
		Transfers: transfers,
		Done:      make(chan tbutil.WorkResult, 1),
	}
	select {
	case <-ctx.Done():
		return tbutil.WorkResult{}, ctx.Err()
	case b.submitter.In <- item:
	}
	select {
	case <-ctx.Done():
		return tbutil.WorkResult{}, ctx.Err()
	case result := <-item.Done:
		if result.Err != nil {
			return result, result.Err
		}
		return result, nil
	}
}

// accountBalance returns the posted balance for an account.
func accountBalance(account tbtypes.Account) uint64 {
	credits := tbutil.Uint128ToUint64(account.CreditsPosted)
	debits := tbutil.Uint128ToUint64(account.DebitsPosted)
	if credits < debits {
		return 0
	}
	return credits - debits
}

// accountAvailable returns the available balance after pending debits.
func accountAvailable(account tbtypes.Account) uint64 {
	balance := accountBalance(account)
	pending := tbutil.Uint128ToUint64(account.DebitsPending)
	if balance < pending {
		return 0
	}
	return balance - pending
}

// capacityTransferID builds a deterministic transfer ID for capacity updates.
func capacityTransferID(key ratelimiter.LimitKey, target uint64) tbtypes.Uint128 {
	label := fmt.Sprintf("xfer:capacity:%s:%d", key, target)
	return tbutil.ID128(label)
}

// firstTransferError returns the first unexpected transfer error.
func firstTransferError(errors map[int]tbtypes.CreateTransferResult) error {
	for _, result := range errors {
		if result == tbtypes.TransferExists {
			continue
		}
		return fmt.Errorf("transfer error: %s", result)
	}
	return nil
}

// remainingWindowSeconds computes remaining rolling seconds for reconciliation.
func remainingWindowSeconds(reservedAtUnix int64, now time.Time, windowSeconds int) int {
	if windowSeconds <= 0 {
		return 1
	}
	reservedAt := time.UnixMilli(reservedAtUnix)
	elapsed := int(now.Sub(reservedAt).Seconds())
	remaining := windowSeconds - elapsed
	if remaining < 1 {
		return 1
	}
	return remaining
}
