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
