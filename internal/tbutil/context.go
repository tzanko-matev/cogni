package tbutil

import (
	"context"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func callWithContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	type result struct {
		value T
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		value, err := fn()
		ch <- result{value: value, err: err}
	}()
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case res := <-ch:
		return res.value, res.err
	}
}

// CreateAccounts executes a CreateAccounts call with context cancellation.
func CreateAccounts(ctx context.Context, client tb.Client, accounts []tbtypes.Account) ([]tbtypes.AccountEventResult, error) {
	return callWithContext(ctx, func() ([]tbtypes.AccountEventResult, error) {
		return client.CreateAccounts(accounts)
	})
}

// LookupAccounts executes a LookupAccounts call with context cancellation.
func LookupAccounts(ctx context.Context, client tb.Client, ids []tbtypes.Uint128) ([]tbtypes.Account, error) {
	return callWithContext(ctx, func() ([]tbtypes.Account, error) {
		return client.LookupAccounts(ids)
	})
}

// CreateTransfers executes a CreateTransfers call with context cancellation.
func CreateTransfers(ctx context.Context, client tb.Client, transfers []tbtypes.Transfer) ([]tbtypes.TransferEventResult, error) {
	return callWithContext(ctx, func() ([]tbtypes.TransferEventResult, error) {
		return client.CreateTransfers(transfers)
	})
}
