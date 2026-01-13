package tbutil

import (
	"encoding/binary"
	"fmt"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// Uint128FromUint64 converts a uint64 to a TigerBeetle Uint128.
func Uint128FromUint64(value uint64) tbtypes.Uint128 {
	return tbtypes.ToUint128(value)
}

// Uint128ToUint64 converts a TigerBeetle Uint128 to uint64 and panics on overflow.
func Uint128ToUint64(value tbtypes.Uint128) uint64 {
	bytes := value.Bytes()
	high := binary.LittleEndian.Uint64(bytes[8:])
	if high != 0 {
		panic(fmt.Errorf("uint128 overflows uint64"))
	}
	return binary.LittleEndian.Uint64(bytes[:8])
}
