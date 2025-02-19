package benchmark

import (
	"encoding/json"
	"testing"

	"github.com/jokruger/dec128"
)

type testJsonStruct struct {
	A dec128.Dec128
	B dec128.Dec128
	C dec128.Dec128
}

func BenchmarkDec128FromString(b *testing.B) {
	s1 := "12345"
	s2 := "1234567890"
	s3 := "123456789012345678901234567890"
	s4 := "12345.12"
	s5 := "1234567890.12345"
	s6 := "123456789012345678901234567890.123456789"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dec128.FromString(s1)
		_ = dec128.FromString(s2)
		_ = dec128.FromString(s3)
		_ = dec128.FromString(s4)
		_ = dec128.FromString(s5)
		_ = dec128.FromString(s6)
	}
}

func BenchmarkDec128JsonUnmarshal(b *testing.B) {
	x := testJsonStruct{
		A: dec128.FromString("123.456789"),
		B: dec128.FromString("1234567890.1234"),
		C: dec128.FromString("123456789012345678901234567890.12"),
	}

	s, err := json.Marshal(x)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var y testJsonStruct
		err := json.Unmarshal(s, &y)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128ToString(b *testing.B) {
	s1 := dec128.FromString("12345")
	s2 := dec128.FromString("1234567890")
	s3 := dec128.FromString("123456789012345678901234567890")
	s4 := dec128.FromString("12345.12")
	s5 := dec128.FromString("1234567890.12345")
	s6 := dec128.FromString("123456789012345678901234567890.123456789")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s1.String()
		_ = s2.String()
		_ = s3.String()
		_ = s4.String()
		_ = s5.String()
		_ = s6.String()
	}
}

func BenchmarkDec128JsonMarshal(b *testing.B) {
	x := testJsonStruct{
		A: dec128.FromString("123.456789"),
		B: dec128.FromString("1234567890.1234"),
		C: dec128.FromString("123456789012345678901234567890.12"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128BinMarshal(b *testing.B) {
	x := dec128.FromString("123.456789")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := x.MarshalBinary()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128BinUnmarshal(b *testing.B) {
	x := dec128.FromString("123.456789")
	bs, err := x.MarshalBinary()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var y dec128.Dec128
		err := y.UnmarshalBinary(bs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128Add(b *testing.B) {
	x := dec128.FromString("1234567890.123456789")
	y := dec128.FromString("1234567890.123456789")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.Add(y)
	}
}
