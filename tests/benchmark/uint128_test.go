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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range ss {
			_, _ = uint128.FromString(s)
		}
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

	vs := make([]uint128.Uint128, len(ss))
	for i, s := range ss {
		j, _ := uint128.FromString(s)
		vs[i] = j
	}

	buf := [uint128.MaxStrLen]byte{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range vs {
			_ = v.StringToBuf(buf[:])
		}
	}
}
