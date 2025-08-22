package uint128

import (
	"math"
	"math/big"
	"testing"
)

func TestBasic1(t *testing.T) {
	u := FromUint64(math.MaxUint64)
	u, _ = u.Add64(1)
	_, e := u.Uint64()
	if e.IsOK() {
		t.Errorf("expected overflow error")
	}
	u, _ = u.Sub64(1)
	i, e := u.Uint64()
	if !e.IsOK() {
		t.Errorf("unexpected error: %s", e.String())
	}
	if i != math.MaxUint64 {
		t.Errorf("expected %d, got %d", uint(math.MaxUint64), i)
	}
}

func TestBasic2(t *testing.T) {
	i1, e := FromString("0")
	if e.IsError() {
		t.Errorf("error creating uint128: %s", e.String())
	}
	if i1.IsZero() != true {
		t.Errorf("expected true, got false")
	}
	if i1.BitLen() != 0 {
		t.Errorf("expected 0, got %v", i1.BitLen())
	}

	i2, e := FromString("1")
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
}

func TestBasic3(t *testing.T) {
	i3, e := FromString("123456789012345678901234567890")
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

func TestBasic4(t *testing.T) {
	i, s := FromString("")
	if s.IsError() {
		t.Errorf("expected no error, got: %s", s.String())
	}
	if i.String() != "0" {
		t.Errorf("expected '0', got '%s'", i.String())
	}

	i, s = FromSafeString("")
	if s.IsError() {
		t.Errorf("expected no error, got: %s", s.String())
	}
	if i.String() != "0" {
		t.Errorf("expected '0', got '%s'", i.String())
	}
}

func TestPow10(t *testing.T) {
	u := FromUint64(1)
	for i := range len(Pow10Uint128) {
		if !u.Equal(Pow10Uint128[i]) {
			t.Errorf("expected %v, got %v", Pow10Uint128[i], u)
		}
		u, _ = u.Mul(FromUint64(10))
	}
}

func TestConvUint64(t *testing.T) {
	testCases := [...]uint64{0, 1, 1234567890, math.MaxUint64 - 1, math.MaxUint64}
	for _, i := range testCases {
		u := FromUint64(i)
		j, e := u.Uint64()
		if e.IsError() {
			t.Errorf("error converting uint128 to uint64: %s", e.String())
		}
		if i != j {
			t.Errorf("expected %v, got %v", i, j)
		}
	}
}

func TestConvString(t *testing.T) {
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
		u, e := FromString(tc)
		if e.IsError() {
			t.Errorf("error converting string to uint128: %s", e.String())
		}
		u, e = FromSafeString(tc)
		if e.IsError() {
			t.Errorf("error converting safe string to uint128: %s", e.String())
		}
		s := u.String()
		if tc != s {
			t.Errorf("expected %v, got %v", tc, s)
		}
	}
}

func TestConvBytes(t *testing.T) {
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
		u, _ := FromString(tc.s)
		be := u.BytesBigEndian()
		le := u.Bytes()
		u2 := FromBytesBigEndian(be)
		u3 := FromBytes(le)
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

func TestConvBigInt1(t *testing.T) {
	testCases := [...]string{"0", "1", "1234567890", "18446744073709551615", "18446744073709551616", "340282366920938463463374607431768211455", "123456789012345678901234567890"}
	for _, tc := range testCases {
		u, _ := FromString(tc)
		i := u.BigInt()
		u2, _ := FromBigInt(i)
		if !u2.Equal(u) {
			t.Errorf("expected %v, got %v", u, u2)
		}
	}
	for _, tc := range testCases {
		u, _ := FromString(tc)
		i, _ := big.NewInt(0).SetString(tc, 10)
		u2, _ := FromBigInt(i)
		if !u2.Equal(u) {
			t.Errorf("expected %v, got %v", u, u2)
		}
		s := u2.String()
		if tc != s {
			t.Errorf("expected %v, got %v", tc, s)
		}
	}
}

func TestConvBigInt2(t *testing.T) {
	_, s := FromBigInt(big.NewInt(-10))
	if !s.IsError() {
		t.Errorf("expected error converting negative big.Int to uint128, got no error")
	}
}

func TestConvBigInt3(t *testing.T) {
	b := big.NewInt(1234567890123456789)
	b = b.Mul(b, b)
	b = b.Mul(b, b)
	_, s := FromBigInt(b)
	if !s.IsError() {
		t.Errorf("expected error converting big.Int greater than max uint128 to uint128, got no error")
	}
}

func TestConvBigInt4(t *testing.T) {
	u, s := FromBigInt(nil)
	if !u.IsZero() {
		t.Errorf("expected zero uint128, got %v", u)
	}
	if !s.IsOK() {
		t.Errorf("expected no error, got: %s", s.String())
	}
}

func TestAdd1(t *testing.T) {
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
		a, _ := FromString(tc.a)
		b, _ := FromString(tc.b)
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

func TestAdd2(t *testing.T) {
	i, _ := FromString("340282366920938463463374607431768211455")
	if _, s := i.Add64(1); !s.IsError() {
		t.Errorf("expected overflow error when adding 1 to max uint128, got no error")
	}
}

func TestSub(t *testing.T) {
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
		a, _ := FromString(tc.a)
		b, _ := FromString(tc.b)
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

func TestMul1(t *testing.T) {
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
		a, _ := FromString(tc.a)
		b, _ := FromString(tc.b)
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

func TestMul2(t *testing.T) {
	i, _ := FromString("340282366920938463463374607431768211455")
	if _, s := i.Mul64(1234567890); !s.IsError() {
		t.Errorf("expected overflow error when multiplying max uint128 by 1234567890, got no error")
	}
	if _, s := i.MulAdd64(1234567890, 2); !s.IsError() {
		t.Errorf("expected overflow error when multiplying max uint128 by 1234567890, got no error")
	}
}

func TestDiv(t *testing.T) {
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
		a, _ := FromString(tc.a)
		b, _ := FromString(tc.b)
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

func TestMulAdd64(t *testing.T) {
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
		u, _ := FromString(tc.u)
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

func TestSubUnsafe(t *testing.T) {
	a, _ := FromString("170141183460469231731687303715884105727")
	b, _ := FromString("170141183460469231731687303715884105726")
	c := SubUnsafe(a, b)
	if c.String() != "1" {
		t.Errorf("expected 1, got %s", c.String())
	}
}

func TestSub64(t *testing.T) {
	type tc struct {
		a string
		b uint64
		r string
		e string // expected error string ("" means OK)
	}
	tests := [...]tc{
		{"0", 0, "0", ""},
		{"1", 0, "1", ""},
		{"1", 1, "0", ""},
		{"0", 1, "0", "underflow"},
		{"2", 1, "1", ""},
		{"18446744073709551615", 1, "18446744073709551614", ""},
		{"18446744073709551616", 1, "18446744073709551615", ""},
		{"340282366920938463463374607431768211455", 1, "340282366920938463463374607431768211454", ""},
		{"340282366920938463463374607431768211455", 18446744073709551615, "340282366920938463444927863358058659840", ""},
		{"5", 7, "0", "underflow"},
	}

	for i, tt := range tests {
		a, _ := FromString(tt.a)
		got, st := a.Sub64(tt.b)
		gs := got.String()
		if gs != tt.r {
			t.Fatalf("case %d: %s - %d => got %s want %s", i, tt.a, tt.b, gs, tt.r)
		}
		if tt.e == "" {
			if st.IsError() {
				t.Fatalf("case %d: unexpected error %s", i, st.String())
			}
		} else {
			if st.IsOK() || st.String() != tt.e {
				t.Fatalf("case %d: expected error %q got %q", i, tt.e, st.String())
			}
		}
	}
}

func TestMulCarry(t *testing.T) {
	type tc struct {
		i string
		o string
		r string
		c string
	}

	tcs := [...]tc{
		{"0", "0", "0", "0"},
		{"1", "1", "1", "0"},
		{"10", "10", "100", "0"},
		{"18446744073709551616", "18446744073709551616", "0", "1"},
		{"340282366920938463463374607431768211455", "18446744073709551616", "340282366920938463444927863358058659840", "18446744073709551615"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		o, _ := FromString(e.o)
		r, c := i.MulCarry(o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
		if c.String() != e.c {
			t.Errorf("expected %s, got %s", e.c, c.String())
		}
	}
}

func TestDiv64(t *testing.T) {
	type tc struct {
		i string
		o uint64
		r string
		c string
	}

	tcs := [...]tc{
		{"0", 0, "0", "division by zero"},
		{"1", 1, "1", "default"},
		{"1", 10, "0", "default"},
		{"18446744073709551616", 123, "149973529054549200", "default"},
		{"340282366920938463463374607431768211455", 1844674407370955161, "184467440737095516220", "default"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r, c := i.Div64(e.o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
		if c.String() != e.c {
			t.Errorf("expected %s, got %s", e.c, c.String())
		}
	}
}

func TestMod(t *testing.T) {
	type tc struct {
		i string
		o string
		r string
		c string
	}

	tcs := [...]tc{
		{"0", "0", "0", "division by zero"},
		{"1", "1", "0", "default"},
		{"1", "10", "1", "default"},
		{"18446744073709551616", "3", "1", "default"},
		{"340282366920938463463374607431768211455", "18446744073709551616", "18446744073709551615", "default"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		o, _ := FromString(e.o)
		r, c := i.Mod(o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
		if c.String() != e.c {
			t.Errorf("expected %s, got %s", e.c, c.String())
		}
	}
}

func TestMod64(t *testing.T) {
	type tc struct {
		i string
		o uint64
		r uint64
		c string
	}

	tcs := [...]tc{
		{"0", 0, 0, "division by zero"},
		{"1", 1, 0, "default"},
		{"1", 10, 1, "default"},
		{"18446744073709551616", 3, 1, "default"},
		{"340282366920938463463374607431768211455", 123, 9, "default"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r, c := i.Mod64(e.o)
		if r != e.r {
			t.Errorf("expected %d, got %d", e.r, r)
		}
		if c.String() != e.c {
			t.Errorf("expected %s, got %s", e.c, c.String())
		}
	}
}

func TestAnd(t *testing.T) {
	type tc struct {
		i string
		o string
		r string
	}

	tcs := [...]tc{
		{"0", "0", "0"},
		{"1", "1", "1"},
		{"1", "0", "0"},
		{"0", "1", "0"},
		{"123456789012345678901234567890", "987654321098765432109876543210", "1943960184490269435062782658"},
		{"18446744073709551615", "18446744073709551615", "18446744073709551615"},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211455", "340282366920938463463374607431768211455"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		o, _ := FromString(e.o)
		r := i.And(o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestAnd64(t *testing.T) {
	type tc struct {
		i string
		o uint64
		r string
	}

	tcs := [...]tc{
		{"0", 0, "0"},
		{"1", 1, "1"},
		{"1", 0, "0"},
		{"0", 1, "0"},
		{"123456789", 987654321, "39471121"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r := i.And64(e.o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestOr(t *testing.T) {
	type tc struct {
		i string
		o string
		r string
	}

	tcs := [...]tc{
		{"0", "0", "0"},
		{"1", "1", "1"},
		{"1", "0", "1"},
		{"0", "1", "1"},
		{"123456789012345678901234567890", "987654321098765432109876543210", "1109167149926620841576048328442"},
		{"18446744073709551615", "18446744073709551615", "18446744073709551615"},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211455", "340282366920938463463374607431768211455"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		o, _ := FromString(e.o)
		r := i.Or(o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestOr64(t *testing.T) {
	type tc struct {
		i string
		o uint64
		r string
	}

	tcs := [...]tc{
		{"0", 0, "0"},
		{"1", 1, "1"},
		{"1", 0, "1"},
		{"0", 1, "1"},
		{"123456789", 987654321, "1071639989"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r := i.Or64(e.o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestXor(t *testing.T) {
	type tc struct {
		i string
		o string
		r string
	}

	tcs := [...]tc{
		{"0", "0", "0"},
		{"1", "1", "0"},
		{"1", "0", "1"},
		{"0", "1", "1"},
		{"123456789012345678901234567890", "987654321098765432109876543210", "1107223189742130572140985545784"},
		{"18446744073709551615", "18446744073709551615", "0"},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211455", "0"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		o, _ := FromString(e.o)
		r := i.Xor(o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestXor64(t *testing.T) {
	type tc struct {
		i string
		o uint64
		r string
	}

	tcs := [...]tc{
		{"0", 0, "0"},
		{"1", 1, "0"},
		{"1", 0, "1"},
		{"0", 1, "1"},
		{"123456789", 987654321, "1032168868"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r := i.Xor64(e.o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestTZBC(t *testing.T) {
	type tc struct {
		i string
		r int
	}

	tcs := [...]tc{
		{"0", 128},
		{"1", 0},
		{"10", 1},
		{"100", 2},
		{"12345678901234567890", 1},
		{"34028236692093846346337460743176821145", 0},
		{"340282366920938463463374607431768211455", 0},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r := i.TrailingZeroBitsCount()
		if r != e.r {
			t.Errorf("expected %d, got %d", e.r, r)
		}
	}
}

func TestNZBC(t *testing.T) {
	type tc struct {
		i string
		r int
	}

	tcs := [...]tc{
		{"0", 0},
		{"1", 1},
		{"10", 2},
		{"100", 3},
		{"12345678901234567890", 32},
		{"34028236692093846346337460743176821145", 63},
		{"340282366920938463463374607431768211455", 128},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r := i.NonZeroBitsCount()
		if r != e.r {
			t.Errorf("expected %d, got %d", e.r, r)
		}
	}
}

func TestRBL(t *testing.T) {
	type tc struct {
		i string
		o int
		r string
	}

	tcs := [...]tc{
		{"0", 0, "0"},
		{"0", 1, "0"},
		{"1", 5, "32"},
		{"1", 100, "1267650600228229401496703205376"},
		{"1234567890", 10, "1264197519360"},
		{"12345678901234567890", 10, "12641975194864197519360"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r := i.RotateBitsLeft(e.o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestRBR(t *testing.T) {
	type tc struct {
		i string
		o int
		r string
	}

	tcs := [...]tc{
		{"0", 0, "0"},
		{"0", 1, "0"},
		{"1", 5, "10633823966279326983230456482242756608"},
		{"1", 100, "268435456"},
		{"1234567890", 10, "239925653239177315059137174380603401600"},
		{"12345678901234567890", 10, "239925653239177315059149230707654182850"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r := i.RotateBitsRight(e.o)
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestRBits(t *testing.T) {
	type tc struct {
		i string
		r string
	}

	tcs := [...]tc{
		{"0", "0"},
		{"1", "170141183460469231731687303715884105728"},
		{"1234567890", "100026547903135029943999284404897710080"},
		{"12345678901234567890", "100112530519533220556082947097410666496"},
		{"123456789012345678901234567890", "100112603668620278587303334147801481216"},
	}

	for _, e := range tcs {
		i, _ := FromString(e.i)
		r := i.ReverseBits()
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
	}
}

func TestMarshalText1(t *testing.T) {
	a, _ := FromString("1234567890")
	bs, err := a.MarshalText()
	if err != nil {
		t.Errorf("error marshaling uint128: %s", err)
	}
	if string(bs) != "1234567890" {
		t.Errorf("expected 1234567890, got %s", string(bs))
	}

	var b Uint128
	if err := b.UnmarshalText(bs); err != nil {
		t.Errorf("error unmarshaling uint128: %s", err)
	}
	if b.String() != "1234567890" {
		t.Errorf("expected %s, got %s", "1234567890", b.String())
	}
}

func TestMarshalText2(t *testing.T) {
	a, _ := FromString("12345678901234567890")
	bs, err := a.MarshalText()
	if err != nil {
		t.Errorf("error marshaling uint128: %s", err)
	}
	if string(bs) != "12345678901234567890" {
		t.Errorf("expected 12345678901234567890, got %s", string(bs))
	}

	var b Uint128
	if err := b.UnmarshalText(bs); err != nil {
		t.Errorf("error unmarshaling uint128: %s", err)
	}
	if b.String() != "12345678901234567890" {
		t.Errorf("expected %s, got %s", "12345678901234567890", b.String())
	}
}

func TestMarshalText3(t *testing.T) {
	a, _ := FromString("123456789012345678901234567890")
	bs, err := a.MarshalText()
	if err != nil {
		t.Errorf("error marshaling uint128: %s", err)
	}
	if string(bs) != "123456789012345678901234567890" {
		t.Errorf("expected 123456789012345678901234567890, got %s", string(bs))
	}

	var b Uint128
	if err := b.UnmarshalText(bs); err != nil {
		t.Errorf("error unmarshaling uint128: %s", err)
	}
	if b.String() != "123456789012345678901234567890" {
		t.Errorf("expected %s, got %s", "123456789012345678901234567890", b.String())
	}
}

func TestMarshalText4(t *testing.T) {
	var b Uint128
	if err := b.UnmarshalText(nil); err != nil {
		t.Errorf("error unmarshaling uint128: %s", err)
	}
	if b.String() != "0" {
		t.Errorf("expected %s, got %s", "0", b.String())
	}

	if err := b.UnmarshalText([]byte("NaN")); err == nil {
		t.Errorf("expected error unmarshaling uint128 from NaN, got nil")
	}
}

func TestQR256b128(t *testing.T) {
	type tc struct {
		u string
		c string
		v string
		q string
		r string
		s string
	}

	tcs := [...]tc{
		{"0", "0", "0", "0", "0", "division by zero"},
		{"0", "1", "1", "0", "0", "overflow"},
		{"1234567890", "0", "123", "10037137", "39", "default"},
		{"1234567890", "12345", "123", "0", "0", "overflow"},
		{"123456789012345678901234567890", "0", "123", "1003713731807688446351500551", "117", "default"},
		{"1", "2", "3", "226854911280625642308916404954512140971", "0", "default"},
		{"1", "2", "3", "226854911280625642308916404954512140971", "0", "default"},
		{"12345678901234567890", "0", "987654321", "12499999887", "339506163", "default"},
		{"42", "1", "170141183460469231731687303715884105728", "2", "42", "default"},
		{"34028236692093846346337460743176821", "17", "340282366920938463463374607431768211455", "17", "34028236692093846346337460743176838", "default"},
		{"98765432109876543210", "281474976710656", "1329227995784915872903807060280344576", "72057594037927936", "98765432109876543210", "default"},
		{"34028236692093846346337460743176821", "0", "4294967296", "7922816251426433759354395", "144310901", "default"},
		{"340282366920938463463374607431768211455", "18446744073709551616", "340282366920938463463374607431768211455", "18446744073709551617", "18446744073709551616", "default"},
		{"555", "1000", "100", "0", "0", "overflow"},
		{"1", "18446744073709551616", "4294967295", "0", "0", "overflow"},
	}

	for _, e := range tcs {
		u, _ := FromString(e.u)
		c, _ := FromString(e.c)
		v, _ := FromString(e.v)
		q, r, s := QuoRem256By128(u, c, v)
		if q.String() != e.q {
			t.Errorf("expected %s, got %s", e.q, q.String())
		}
		if r.String() != e.r {
			t.Errorf("expected %s, got %s", e.r, r.String())
		}
		if s.String() != e.s {
			t.Errorf("expected %s, got %s", e.s, s.String())
		}
	}
}

func TestAppendBytes(t *testing.T) {
	a, _ := FromString("123456789012345678901234567890")
	bs := a.AppendBytes([]byte("xxx"))
	if len(bs) != 19 {
		t.Errorf("expected 19 byte, got %d", len(bs))
	}
	bs = a.AppendBytesBigEndian(bs)
	if len(bs) != 35 {
		t.Errorf("expected 35 byte, got %d", len(bs))
	}
}

func TestReverseBytes(t *testing.T) {
	a, _ := FromString("123456789012345678901234567890")
	a = a.ReverseBytes()
	if a.String() == "123456789012345678901234567890" {
		t.Errorf("expected reversed bytes, got %s", a.String())
	}
	a = a.ReverseBytes()
	if a.String() != "123456789012345678901234567890" {
		t.Errorf("expected original bytes, got %s", a.String())
	}
}

func TestPutBytes(t *testing.T) {
	i := FromUint64(1234567890)

	bs := make([]byte, 2)
	if !i.PutBytes(bs).IsError() {
		t.Errorf("expected error when putting bytes into too small slice")
	}
	if !i.PutBytesBigEndian(bs).IsError() {
		t.Errorf("expected error when putting bytes into too small slice")
	}
}
