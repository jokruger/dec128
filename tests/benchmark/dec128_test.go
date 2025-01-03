package benchmark

import (
	"testing"

	"github.com/jokruger/dec128"
)

func BenchmarkDec128FromString(b *testing.B) {
	s1 := "12345"
	s2 := "1234567890"
	s3 := "123456789012345678901234567890"
	s4 := "12345.12"
	s5 := "1234567890.12345"
	s6 := "123456789012345678901234567890.123456789"
	for i := 0; i < b.N; i++ {
		_, _ = dec128.FromString(s1)
		_, _ = dec128.FromString(s2)
		_, _ = dec128.FromString(s3)
		_, _ = dec128.FromString(s4)
		_, _ = dec128.FromString(s5)
		_, _ = dec128.FromString(s6)
	}
}

func BenchmarkDec128ToString(b *testing.B) {
	s1, _ := dec128.FromString("12345")
	s2, _ := dec128.FromString("1234567890")
	s3, _ := dec128.FromString("123456789012345678901234567890")
	s4, _ := dec128.FromString("12345.12")
	s5, _ := dec128.FromString("1234567890.12345")
	s6, _ := dec128.FromString("123456789012345678901234567890.123456789")
	for i := 0; i < b.N; i++ {
		_ = s1.String()
		_ = s2.String()
		_ = s3.String()
		_ = s4.String()
		_ = s5.String()
		_ = s6.String()
	}
}
