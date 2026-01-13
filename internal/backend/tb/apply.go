package tb

import (
	"context"
	"fmt"

	"cogni/internal/tbutil"
	"cogni/pkg/ratelimiter"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
)

// ApplyDefinition provisions accounts and updates capacity.
func (b *Backend) ApplyDefinition(ctx context.Context, def ratelimiter.LimitDefinition) error {
	if err := b.ensureAccounts(ctx, def); err != nil {
		return err
	}

	b.mu.Lock()
	prev, ok := b.states[def.Key]
	b.mu.Unlock()

	if !ok || def.Capacity >= prev.Definition.Capacity {
		if err := b.ensureCapacity(ctx, def); err != nil {
			return err
		}
		b.mu.Lock()
		b.states[def.Key] = ratelimiter.LimitState{
			Definition:        def,
			Status:            ratelimiter.LimitStatusActive,
			PendingDecreaseTo: 0,
		}
		b.mu.Unlock()
		return nil
	}

	b.mu.Lock()
	b.states[def.Key] = ratelimiter.LimitState{
		Definition:        prev.Definition,
		Status:            ratelimiter.LimitStatusDecreasing,
		PendingDecreaseTo: def.Capacity,
	}
	b.mu.Unlock()
	return nil
}

// ApplyState loads a persisted limit state into the backend.
func (b *Backend) ApplyState(state ratelimiter.LimitState) error {
	if err := b.ensureAccounts(context.Background(), state.Definition); err != nil {
		return err
	}
	if err := b.ensureCapacity(context.Background(), state.Definition); err != nil {
		return err
	}
	b.mu.Lock()
	b.states[state.Definition.Key] = state
	b.mu.Unlock()
	return nil
}

// ensureAccounts creates operator, limit, and debt accounts as needed.
func (b *Backend) ensureAccounts(ctx context.Context, def ratelimiter.LimitDefinition) error {
	accounts := []tbtypes.Account{
		{
			ID:     tbutil.OperatorAccountID(),
			Ledger: ledgerLimits,
			Code:   codeLimit,
		},
		{
			ID:     tbutil.LimitAccountID(def.Key),
			Ledger: ledgerLimits,
			Code:   codeLimit,
			Flags:  tbtypes.AccountFlags{DebitsMustNotExceedCredits: true},
		},
	}
	if def.Overage == ratelimiter.OverageDebt {
		accounts = append(accounts, tbtypes.Account{
			ID:     tbutil.DebtAccountID(def.Key),
			Ledger: ledgerLimits,
			Code:   codeLimit,
		})
	}
	results, err := b.createAccounts(ctx, accounts)
	if err != nil {
		return err
	}
	for _, result := range results {
		if result.Result == tbtypes.AccountExists {
			continue
		}
		return fmt.Errorf("create account error: %s", result.Result)
	}
	return nil
}

// ensureCapacity increases the resource account to match the target capacity.
func (b *Backend) ensureCapacity(ctx context.Context, def ratelimiter.LimitDefinition) error {
	account, err := b.lookupAccount(ctx, tbutil.LimitAccountID(def.Key))
	if err != nil {
		return err
	}
	balance := accountBalance(account)
	if def.Capacity <= balance {
		return nil
	}
	delta := def.Capacity - balance
	transfer := tbtypes.Transfer{
		ID:              capacityTransferID(def.Key, def.Capacity),
		DebitAccountID:  tbutil.OperatorAccountID(),
		CreditAccountID: tbutil.LimitAccountID(def.Key),
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
	return nil
}

// createAccounts issues a CreateAccounts call through the client pool.
func (b *Backend) createAccounts(ctx context.Context, accounts []tbtypes.Account) ([]tbtypes.AccountEventResult, error) {
	client, err := b.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer b.pool.Release(client)
	return client.CreateAccounts(accounts)
}

// lookupAccount fetches a single account by ID.
func (b *Backend) lookupAccount(ctx context.Context, id tbtypes.Uint128) (tbtypes.Account, error) {
	client, err := b.pool.Acquire(ctx)
	if err != nil {
		return tbtypes.Account{}, err
	}
	defer b.pool.Release(client)
	accounts, err := client.LookupAccounts([]tbtypes.Uint128{id})
	if err != nil {
		return tbtypes.Account{}, err
	}
	if len(accounts) == 0 {
		return tbtypes.Account{}, fmt.Errorf("account not found")
	}
	return accounts[0], nil
}
