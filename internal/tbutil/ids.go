package tbutil

import (
	"crypto/sha256"
	"strconv"

	"cogni/pkg/ratelimiter"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
)

const (
	operatorAccountLabel  = "acct:operator"
	limitAccountPrefix    = "acct:limit:"
	debtAccountPrefix     = "acct:debt:"
	reserveTransferPref   = "xfer:reserve:"
	voidTransferPref      = "xfer:void:"
	rereserveTransferPref = "xfer:rereserve:"
	debtTransferPref      = "xfer:debt:"
	decreaseTransferPref  = "xfer:decrease:"
)

// ID128 deterministically maps a string label to a TigerBeetle Uint128.
func ID128(label string) tbtypes.Uint128 {
	sum := sha256.Sum256([]byte(label))
	var raw [16]byte
	copy(raw[:], sum[:16])
	if isZero(raw) || isMax(raw) {
		raw[0] ^= 0x01
	}
	return tbtypes.BytesToUint128(raw)
}

// OperatorAccountID returns the operator account ID.
func OperatorAccountID() tbtypes.Uint128 {
	return ID128(operatorAccountLabel)
}

// LimitAccountID returns the account ID for a limit key.
func LimitAccountID(key ratelimiter.LimitKey) tbtypes.Uint128 {
	return ID128(limitAccountPrefix + string(key))
}

// DebtAccountID returns the debt account ID for a limit key.
func DebtAccountID(key ratelimiter.LimitKey) tbtypes.Uint128 {
	return ID128(debtAccountPrefix + string(key))
}

// ReserveTransferID returns the transfer ID for a reserve attempt.
func ReserveTransferID(leaseID string, key ratelimiter.LimitKey) tbtypes.Uint128 {
	return ID128(reserveTransferPref + leaseID + ":" + string(key))
}

// VoidTransferID returns the transfer ID used to void a pending reservation.
func VoidTransferID(leaseID string, key ratelimiter.LimitKey) tbtypes.Uint128 {
	return ID128(voidTransferPref + leaseID + ":" + string(key))
}

// RereserveTransferID returns the transfer ID for reconciliation transfers.
func RereserveTransferID(leaseID string, key ratelimiter.LimitKey) tbtypes.Uint128 {
	return ID128(rereserveTransferPref + leaseID + ":" + string(key))
}

// DebtTransferID returns the transfer ID for debt tracking.
func DebtTransferID(leaseID string, key ratelimiter.LimitKey) tbtypes.Uint128 {
	return ID128(debtTransferPref + leaseID + ":" + string(key))
}

// DecreaseTransferID returns the transfer ID for capacity decreases.
func DecreaseTransferID(key ratelimiter.LimitKey, target uint64) tbtypes.Uint128 {
	label := decreaseTransferPref + string(key) + ":" + ratelimiterKeySuffix(target)
	return ID128(label)
}

// isZero reports whether the 16-byte array is all zeros.
func isZero(raw [16]byte) bool {
	for _, b := range raw[:] {
		if b != 0 {
			return false
		}
	}
	return true
}

// isMax reports whether the 16-byte array is all 0xFF.
func isMax(raw [16]byte) bool {
	for _, b := range raw[:] {
		if b != 0xFF {
			return false
		}
	}
	return true
}

// ratelimiterKeySuffix formats a numeric suffix for IDs.
func ratelimiterKeySuffix(target uint64) string {
	return strconv.FormatUint(target, 10)
}
