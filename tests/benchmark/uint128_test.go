package benchmark

import (
	"testing"

	"github.com/jokruger/dec128/uint128"
)

func BenchmarkUint128FromString(b *testing.B) {
	s1 := "1234567890"
	s2 := "12345678901234567890"
	s3 := "1234567890123456789012345678901234567890"
	for i := 0; i < b.N; i++ {
		_, _ = uint128.FromString(s1)
		_, _ = uint128.FromString(s2)
		_, _ = uint128.FromString(s3)
	}
}

func BenchmarkUint128StringConv(b *testing.B) {
	strs := []string{
		"1234567890",
		"12345678901234567890",
		"1234567890123456789012345678901234567890",
	}
	vals := make([]uint128.Uint128, len(strs))
	for i, s := range strs {
		vals[i], _ = uint128.FromString(s)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range vals {
			s := v.String()
			_, _ = uint128.FromString(s)
		}
	}
}

func BenchmarkUint128BigIntConv(b *testing.B) {
	strs := []string{
		"1234567890",
		"12345678901234567890",
		"1234567890123456789012345678901234567890",
	}
	vals := make([]uint128.Uint128, len(strs))
	for i, s := range strs {
		vals[i], _ = uint128.FromString(s)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range vals {
			b := v.BigInt()
			_, _ = uint128.FromBigInt(b)
		}
	}
}
