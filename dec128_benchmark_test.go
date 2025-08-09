package dec128

import (
	"encoding/json"
	"testing"
)

type testJsonStruct struct {
	A Dec128
	B Dec128
	C Dec128
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
		_ = FromSafeString(ss[i%sz])
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

	vs := make([]Dec128, sz)
	for i, s := range ss {
		vs[i] = FromString(s)
	}

	buf := [MaxStrLen]byte{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//_ = vs[i%sz].String()
		_ = vs[i%sz].StringToBuf(buf[:])
	}
}

func BenchmarkDec128JsonUnmarshal(b *testing.B) {
	x := testJsonStruct{
		A: FromString("123.456789"),
		B: FromString("1234567890.1234"),
		C: FromString("123456789012345678901234567890.12"),
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
		A: FromString("123.456789"),
		B: FromString("1234567890.1234"),
		C: FromString("123456789012345678901234567890.12"),
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
	x := FromString("123.456789")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := x.MarshalBinary()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128BinUnmarshal(b *testing.B) {
	x := FromString("123.456789")
	bs, err := x.MarshalBinary()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var y Dec128
		err := y.UnmarshalBinary(bs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128Add(b *testing.B) {
	x := FromString("1234567890.123456789")
	y := FromString("1234567890.123456789")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.Add(y)
	}
}
