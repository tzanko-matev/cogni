package tb

import tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

// hasNonIgnorableErrors reports whether errors contain unexpected results.
func hasNonIgnorableErrors(errors map[int]tbtypes.CreateTransferResult) bool {
	if len(errors) == 0 {
		return false
	}
	hasIgnorable := false
	for _, result := range errors {
		if result == tbtypes.TransferLinkedEventFailed {
			continue
		}
		if isIgnorableResult(result) {
			hasIgnorable = true
			continue
		}
		return true
	}
	if !hasIgnorable {
		return true
	}
	return false
}

// isOverageDenied checks if an overage reserve failed due to capacity.
func isOverageDenied(errors map[int]tbtypes.CreateTransferResult) bool {
	for _, result := range errors {
		switch result {
		case tbtypes.TransferExceedsCredits, tbtypes.TransferExceedsDebits:
			return true
		}
	}
	return false
}

// isIgnorableResult returns true for idempotent or already-released results.
func isIgnorableResult(result tbtypes.CreateTransferResult) bool {
	switch result {
	case tbtypes.TransferExists,
		tbtypes.TransferPendingTransferExpired,
		tbtypes.TransferPendingTransferAlreadyVoided,
		tbtypes.TransferPendingTransferAlreadyPosted,
		tbtypes.TransferPendingTransferNotFound:
		return true
	default:
		return false
	}
}
