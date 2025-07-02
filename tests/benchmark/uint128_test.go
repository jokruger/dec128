package benchmark

import (
	"testing"

	"github.com/jokruger/dec128/uint128"
)

func BenchmarkUint128FromString(b *testing.B) {
	ss := []string{
		"0",
		"1",
		"123",
		"1234567890",
		"12345678901234567890",
		"1234567890123456789012345678901234567890",
		"9999999999",
		"1111111111",
		"987654321987654321",
		"9182736451029384756",
	}

	sz := len(ss)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//_, _ = uint128.FromString(ss[i%sz])
		_, _ = uint128.FromSafeString(ss[i%sz])
	}
}

func BenchmarkUint128ToString(b *testing.B) {
	ss := []string{
		"0",
		"1",
		"123",
		"1234567890",
		"12345678901234567890",
		"1234567890123456789012345678901234567890",
		"9999999999",
		"1111111111",
		"987654321987654321",
		"9182736451029384756",
	}
	sz := len(ss)

	vs := make([]uint128.Uint128, sz)
	for i, s := range ss {
		j, _ := uint128.FromString(s)
		vs[i] = j
	}

	buf := [uint128.MaxStrLen]byte{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vs[i%sz].StringToBuf(buf[:])
	}
}
