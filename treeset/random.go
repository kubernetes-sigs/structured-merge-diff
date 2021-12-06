package treeset

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/rand"
)

func (t *IntWithComparator) randomPriority() int {
	if t.Rand != nil {
		return t.Rand.Int()
	}
	return rand.Int()
}

func NewRand() *rand.Rand {
	var b [8]byte // 8 byte mini slice in the stack space
	_, err := cryptorand.Read(b[:])
	if err != nil {
		return nil
	}
	return rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(b[:]))))
}
