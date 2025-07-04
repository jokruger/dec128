package unit

import (
	"math/big"
	"testing"

	"github.com/jokruger/dec128/uint128"
)

func TestUint128Pow10(t *testing.T) {
	u := uint128.FromUint64(1)
	for i := range len(uint128.Pow10Uint128) {
		if !u.Equal(uint128.Pow10Uint128[i]) {
			t.Errorf("expected %v, got %v", uint128.Pow10Uint128[i], u)
		}
		u, _ = u.Mul(uint128.FromUint64(10))
	}
}

func TestUint128ConvUint64(t *testing.T) {
	testCases := [...]uint64{0, 1, 1234567890, 18446744073709551615}
	for _, i := range testCases {
		u := uint128.FromUint64(i)
		j, e := u.Uint64()
		if e.IsError() {
			t.Errorf("error converting uint128 to uint64: %s", e.String())
		}
		if i != j {
			t.Errorf("expected %v, got %v", i, j)
		}
	}

	u := uint128.FromUint64(18446744073709551615)
	u, _ = u.Add(uint128.FromUint64(1))
	_, e := u.Uint64()
	if e.IsOK() {
		t.Errorf("expected overflow error")
	}
}

func TestUint128ConvString(t *testing.T) {
	testCases := [...]string{
		"0",
		"1",
		"12",
		"123",
		"1234",
		"12345",
		"123456",
		"1234567",
		"12345678",
		"123456789",
		"1234567890",
		"12345678901",
		"123456789012",
		"1234567890123",
		"12345678901234",
		"123456789012345",
		"1234567890123456",
		"12345678901234567",
		"123456789012345678",
		"1234567890123456789",
		"12345678901234567890",
		"123456789012345678901",
		"1234567890123456789012",
		"12345678901234567890123",
		"123456789012345678901234",
		"1234567890123456789012345",
		"12345678901234567890123456",
		"123456789012345678901234567",
		"1234567890123456789012345678",
		"12345678901234567890123456789",
		"123456789012345678901234567890",
		"1234567890123456789012345678901",
		"12345678901234567890123456789012",
		"123456789012345678901234567890123",
		"1234567890123456789012345678901234",
		"12345678901234567890123456789012345",
		"123456789012345678901234567890123456",
		"1234567890123456789012345678901234567",
		"12345678901234567890123456789012345678",
		"123456789012345678901234567890123456789",
		"254",
		"255",
		"256",
		"65534",
		"65535",
		"65536",
		"16777215",
		"16777216",
		"16777217",
		"4294967294",
		"4294967295",
		"4294967296",
		"1099511627774",
		"1099511627775",
		"1099511627776",
		"281474976710654",
		"281474976710655",
		"281474976710656",
		"72057594037927934",
		"72057594037927935",
		"72057594037927936",
		"18446744073709551614",
		"18446744073709551615",
		"18446744073709551616",
		"4722366482869645213694",
		"4722366482869645213695",
		"4722366482869645213696",
		"1208925819614629174706174",
		"1208925819614629174706175",
		"1208925819614629174706176",
		"309485009821345068724781054",
		"309485009821345068724781055",
		"309485009821345068724781056",
		"79228162514264337593543950334",
		"79228162514264337593543950335",
		"79228162514264337593543950336",
		"20282409603651670423947251286014",
		"20282409603651670423947251286015",
		"20282409603651670423947251286016",
		"5192296858534827628530496329220094",
		"5192296858534827628530496329220095",
		"5192296858534827628530496329220096",
		"1329227995784915872903807060280344574",
		"1329227995784915872903807060280344575",
		"1329227995784915872903807060280344576",
		"340282366920938463463374607431768211454",
		"340282366920938463463374607431768211455",
	}
	for _, tc := range testCases {
		u, e := uint128.FromString(tc)
		if e.IsError() {
			t.Errorf("error converting string to uint128: %s", e.String())
		}
		u, e = uint128.FromSafeString(tc)
		if e.IsError() {
			t.Errorf("error converting safe string to uint128: %s", e.String())
		}
		s := u.String()
		if tc != s {
			t.Errorf("expected %v, got %v", tc, s)
		}
	}
}

func TestUint128ConvBytes(t *testing.T) {
	type testCase struct {
		s  string
		be [16]byte
		le [16]byte
	}

	testCases := [...]testCase{
		{"0", [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"1", [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, [16]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"18446744073709551614", [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 254}, [16]byte{254, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"18446744073709551615", [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255}, [16]byte{255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"18446744073709551616", [16]byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0}, [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}},
		{"340282366920938463463374607431768211454", [16]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 254}, [16]byte{254, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
		{"340282366920938463463374607431768211455", [16]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}, [16]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
		{"123456789012345678901234567890", [16]byte{0, 0, 0, 1, 142, 233, 15, 246, 195, 115, 224, 238, 78, 63, 10, 210}, [16]byte{210, 10, 63, 78, 238, 224, 115, 195, 246, 15, 233, 142, 1, 0, 0, 0}},
	}

	for _, tc := range testCases {
		u, _ := uint128.FromString(tc.s)
		be := u.BytesBigEndian()
		le := u.Bytes()
		u2 := uint128.FromBytesBigEndian(be)
		u3 := uint128.FromBytes(le)
		if !u2.Equal(u) {
			t.Errorf("[big endian] Expected %v, got %v", u, u2)
		}
		if !u3.Equal(u) {
			t.Errorf("[little endian] Expected %v, got %v", u, u3)
		}
		s := u2.String()
		if tc.s != s {
			t.Errorf("expected %v, got %v", tc.s, s)
		}
		s = u3.String()
		if tc.s != s {
			t.Errorf("expected %v, got %v", tc.s, s)
		}
		if be != tc.be {
			t.Errorf("[be] expected %v, got %v", tc.be, be)
		}
		if le != tc.le {
			t.Errorf("[le] expected %v, got %v", tc.le, le)
		}
	}
}

func TestUint128ConvBigInt(t *testing.T) {
	testCases := [...]string{"0", "1", "1234567890", "18446744073709551615", "18446744073709551616", "340282366920938463463374607431768211455", "123456789012345678901234567890"}
	for _, tc := range testCases {
		u, _ := uint128.FromString(tc)
		i := u.BigInt()
		u2, _ := uint128.FromBigInt(i)
		if !u2.Equal(u) {
			t.Errorf("expected %v, got %v", u, u2)
		}
	}
	for _, tc := range testCases {
		u, _ := uint128.FromString(tc)
		i, _ := big.NewInt(0).SetString(tc, 10)
		u2, _ := uint128.FromBigInt(i)
		if !u2.Equal(u) {
			t.Errorf("expected %v, got %v", u, u2)
		}
		s := u2.String()
		if tc != s {
			t.Errorf("expected %v, got %v", tc, s)
		}
	}
}

func TestUint128(t *testing.T) {
	i1, e := uint128.FromString("0")
	if e.IsError() {
		t.Errorf("error creating uint128: %s", e.String())
	}
	if i1.IsZero() != true {
		t.Errorf("expected true, got false")
	}
	if i1.BitLen() != 0 {
		t.Errorf("expected 0, got %v", i1.BitLen())
	}

	i2, e := uint128.FromString("1")
	if e.IsError() {
		t.Errorf("error creating uint128: %s", e.String())
	}
	if i2.IsZero() != false {
		t.Errorf("expected false, got true")
	}
	if i2.BitLen() != 1 {
		t.Errorf("expected 1, got %v", i2.BitLen())
	}

	if i1.Equal(i2) != false {
		t.Errorf("expected false, got true")
	}

	i3, e := uint128.FromString("123456789012345678901234567890")
	if e.IsError() {
		t.Errorf("error creating uint128: %s", e.String())
	}
	if i3.IsZero() != false {
		t.Errorf("expected false, got true")
	}
	if i3.BitLen() != 97 {
		t.Errorf("expected 97, got %v", i3.BitLen())
	}
}

func TestUint128Add(t *testing.T) {
	type testCase struct {
		a string
		b string
		c string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", "0", ""},
		{"0", "1", "1", ""},
		{"1", "0", "1", ""},
		{"1", "1", "2", ""},
		{"18446744073709551615", "1", "18446744073709551616", ""},
		{"1", "18446744073709551615", "18446744073709551616", ""},
		{"18446744073709551615", "18446744073709551615", "36893488147419103230", ""},
		{"340282366920938463463374607431768211455", "1", "0", "overflow"},
		{"1", "340282366920938463463374607431768211455", "0", "overflow"},
	}

	for _, tc := range testCases {
		a, _ := uint128.FromString(tc.a)
		b, _ := uint128.FromString(tc.b)
		c, e := a.Add(b)
		s := c.String()
		if tc.c != s {
			t.Errorf("expected %v, got %v", tc.c, s)
		}
		if tc.e == "" && e.IsError() {
			t.Errorf("expected no error, got: %s", e.String())
		}
		if tc.e != "" && (e.IsOK() || e.String() != tc.e) {
			t.Errorf("expected error '%s', got '%s'", tc.e, e.String())
		}
	}
}

func TestUint128Sub(t *testing.T) {
	type testCase struct {
		a string
		b string
		c string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", "0", ""},
		{"0", "1", "0", "underflow"},
		{"1", "0", "1", ""},
		{"1", "1", "0", ""},
		{"18446744073709551615", "1", "18446744073709551614", ""},
		{"1", "18446744073709551615", "0", "underflow"},
		{"18446744073709551615", "18446744073709551615", "0", ""},
		{"340282366920938463463374607431768211455", "1", "340282366920938463463374607431768211454", ""},
		{"1", "340282366920938463463374607431768211455", "0", "underflow"},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211455", "0", ""},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211454", "1", ""},
		{"18446744073709551616", "1", "18446744073709551615", ""},
	}

	for _, tc := range testCases {
		a, _ := uint128.FromString(tc.a)
		b, _ := uint128.FromString(tc.b)
		c, e := a.Sub(b)
		s := c.String()
		if tc.c != s {
			t.Errorf("expected %v, got %v", tc.c, s)
		}
		if tc.e == "" && e.IsError() {
			t.Errorf("expected no error, got: %s", e.String())
		}
		if tc.e != "" && (e.IsOK() || e.String() != tc.e) {
			t.Errorf("expected error '%s', got '%s'", tc.e, e.String())
		}
	}
}

func TestUint128Mul(t *testing.T) {
	type testCase struct {
		a string
		b string
		c string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", "0", ""},
		{"0", "1", "0", ""},
		{"1", "0", "0", ""},
		{"1", "1", "1", ""},
		{"18446744073709551615", "1", "18446744073709551615", ""},
		{"1", "18446744073709551615", "18446744073709551615", ""},
		{"18446744073709551615", "18446744073709551615", "340282366920938463426481119284349108225", ""},
		{"340282366920938463463374607431768211455", "1", "340282366920938463463374607431768211455", ""},
		{"1", "340282366920938463463374607431768211455", "340282366920938463463374607431768211455", ""},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211455", "0", "overflow"},
		{"1", "10", "10", ""},
		{"10", "10", "100", ""},
		{"100", "100", "10000", ""},
		{"10000", "10000", "100000000", ""},
		{"100000000", "100000000", "10000000000000000", ""},
		{"10000000000000000", "10000000000000000", "100000000000000000000000000000000", ""},
	}

	for _, tc := range testCases {
		a, _ := uint128.FromString(tc.a)
		b, _ := uint128.FromString(tc.b)
		c, e := a.Mul(b)
		s := c.String()
		if tc.c != s {
			t.Errorf("expected %v, got %v", tc.c, s)
		}
		if tc.e == "" && e.IsError() {
			t.Errorf("expected no error, got: %s", e.String())
		}
		if tc.e != "" && (e.IsOK() || e.String() != tc.e) {
			t.Errorf("expected error '%s', got '%s'", tc.e, e.String())
		}
	}
}

func TestUint128Div(t *testing.T) {
	type testCase struct {
		a string
		b string
		c string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", "0", "division by zero"},
		{"0", "1", "0", ""},
		{"1", "1", "1", ""},
		{"18446744073709551615", "1", "18446744073709551615", ""},
		{"1", "18446744073709551615", "0", ""},
		{"18446744073709551615", "18446744073709551615", "1", ""},
		{"340282366920938463463374607431768211455", "1", "340282366920938463463374607431768211455", ""},
		{"1", "340282366920938463463374607431768211455", "0", ""},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211455", "1", ""},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211454", "1", ""},
		{"100000000000000000000000000000000", "10000000000000000", "10000000000000000", ""},
		{"10000000000000000", "100", "100000000000000", ""},
		{"1", "2", "0", ""},
		{"2", "2", "1", ""},
		{"3", "2", "1", ""},
		{"4", "2", "2", ""},
		{"5", "2", "2", ""},
		{"6", "2", "3", ""},
		{"7", "2", "3", ""},
		{"8", "2", "4", ""},
		{"9", "2", "4", ""},
		{"10", "2", "5", ""},
		{"340282366920938463463374607431768211455", "2", "170141183460469231731687303715884105727", ""},
	}

	for _, tc := range testCases {
		a, _ := uint128.FromString(tc.a)
		b, _ := uint128.FromString(tc.b)
		c, e := a.Div(b)
		s := c.String()
		if tc.c != s {
			t.Errorf("expected %v, got %v", tc.c, s)
		}
		if tc.e == "" && e.IsError() {
			t.Errorf("expected no error, got: %s", e.String())
		}
		if tc.e != "" && (e.IsOK() || e.String() != tc.e) {
			t.Errorf("expected error '%s', got '%s'", tc.e, e.String())
		}
	}
}

func TestUint128MulAdd64(t *testing.T) {
	type testCase struct {
		u string
		a uint64
		b uint64
		r string
		e string
	}

	testCases := [...]testCase{
		{"0", 0, 0, "0", ""},
		{"1", 1, 1, "2", ""},
		{"1", 2, 3, "5", ""},
		{"123", 456, 789, "56877", ""},
		{"18446744073709551615", 18446744073709551615, 0, "340282366920938463426481119284349108225", ""},
		{"18446744073709551615", 18446744073709551615, 1, "340282366920938463426481119284349108226", ""},
		{"170141183460469231731687303715884105727", 2, 0, "340282366920938463463374607431768211454", ""},
		{"170141183460469231731687303715884105727", 2, 1, "340282366920938463463374607431768211455", ""},
		{"170141183460469231731687303715884105727", 2, 2, "0", "overflow"},
	}

	for _, tc := range testCases {
		u, _ := uint128.FromString(tc.u)
		x, e := u.MulAdd64(tc.a, tc.b)
		s := x.String()
		if tc.r != s {
			t.Errorf("expected %v, got %v", tc.r, s)
		}
		if tc.e == "" && e.IsError() {
			t.Errorf("expected no error, got: %s", e.String())
		}
		if tc.e != "" && (e.IsOK() || e.String() != tc.e) {
			t.Errorf("expected error '%s', got '%s'", tc.e, e.String())
		}
	}
}
