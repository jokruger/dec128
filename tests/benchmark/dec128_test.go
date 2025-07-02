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
	ss := []string{
		"12345",
		"1234567890",
		"123456789012345678901234567890",
		"12345.12",
		"1234567890.12345",
		"123456789012345678901234567890.123456789",
		"-123.456",
		"0",
		"0.1",
		"9876.54321",
	}

	sz := len(ss)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//_ = dec128.FromString(ss[i%sz])
		_ = dec128.FromSafeString(ss[i%sz])
	}
}

func BenchmarkDec128ToString(b *testing.B) {
	ss := []string{
		"12345",
		"1234567890",
		"123456789012345678901234567890",
		"12345.12",
		"1234567890.12345",
		"123456789012345678901234567890.123456789",
		"-123.456",
		"0",
		"0.1",
		"9876.54321",
	}
	sz := len(ss)

	vs := make([]dec128.Dec128, sz)
	for i, s := range ss {
		vs[i] = dec128.FromString(s)
	}

	buf := [dec128.MaxStrLen]byte{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//_ = vs[i%sz].String()
		_ = vs[i%sz].StringToBuf(buf[:])
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
