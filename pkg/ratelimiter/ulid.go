package ratelimiter

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"sync/atomic"
	"time"
)

const ulidAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var (
	ulidEncoding = base32.NewEncoding(ulidAlphabet).WithPadding(base32.NoPadding)
	ulidCounter  uint64
)

// NewULID returns a ULID string suitable for LeaseID and JobID values.
func NewULID() string {
	var data [16]byte
	ms := uint64(time.Now().UnixMilli())
	data[0] = byte(ms >> 40)
	data[1] = byte(ms >> 32)
	data[2] = byte(ms >> 24)
	data[3] = byte(ms >> 16)
	data[4] = byte(ms >> 8)
	data[5] = byte(ms)
	if _, err := rand.Read(data[6:]); err != nil {
		fillULIDFallback(&data, ms)
	}
	return ulidEncoding.EncodeToString(data[:])
}

// fillULIDFallback populates the randomness bytes when crypto/rand is unavailable.
func fillULIDFallback(data *[16]byte, ms uint64) {
	counter := atomic.AddUint64(&ulidCounter, 1)
	binary.BigEndian.PutUint64(data[6:], counter)
	data[14] = byte(ms >> 8)
	data[15] = byte(ms)
}
