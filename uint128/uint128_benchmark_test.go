package uint128

import (
	"testing"
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
		//_, _ = FromString(ss[i%sz])
		_, _ = FromSafeString(ss[i%sz])
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

	vs := make([]Uint128, sz)
	for i, s := range ss {
		j, _ := FromString(s)
		vs[i] = j
	}

	buf := [MaxStrLen]byte{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vs[i%sz].StringToBuf(buf[:])
	}
}
