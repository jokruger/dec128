package unit

import (
	"math/big"
	"testing"

	"github.com/jokruger/dec128/uint128"
)

func TestUint128ConvUint64(t *testing.T) {
	testCases := [...]uint64{0, 1, 1234567890, 18446744073709551615}
	for _, i := range testCases {
		u := uint128.FromUint64(i)
		j, err := u.Uint64()
		if err.Value() != nil {
			t.Errorf("Error converting uint128 to uint64: %v", err.Value())
		}
		if i != j {
			t.Errorf("Expected %v, got %v", i, j)
		}
	}

	u := uint128.FromUint64(18446744073709551615)
	u, _ = u.Add(uint128.FromUint64(1))
	_, err := u.Uint64()
	if err.Value() == nil {
		t.Errorf("Expected error: %v", err.Value())
	}
}

func TestUint128ConvString(t *testing.T) {
	testCases := [...]string{"0", "1", "1234567890", "18446744073709551615", "18446744073709551616", "340282366920938463463374607431768211455"}
	for _, tc := range testCases {
		u, err := uint128.FromString(tc)
		if err.Value() != nil {
			t.Errorf("Error converting string to uint128: %v", err.Value())
		}
		s := u.String()
		if tc != s {
			t.Errorf("Expected %v, got %v", tc, s)
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
			t.Errorf("Expected %v, got %v", tc.s, s)
		}
		s = u3.String()
		if tc.s != s {
			t.Errorf("Expected %v, got %v", tc.s, s)
		}
		if be != tc.be {
			t.Errorf("[be] Expected %v, got %v", tc.be, be)
		}
		if le != tc.le {
			t.Errorf("[le] Expected %v, got %v", tc.le, le)
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
			t.Errorf("Expected %v, got %v", u, u2)
		}
	}
	for _, tc := range testCases {
		u, _ := uint128.FromString(tc)
		i, _ := big.NewInt(0).SetString(tc, 10)
		u2, _ := uint128.FromBigInt(i)
		if !u2.Equal(u) {
			t.Errorf("Expected %v, got %v", u, u2)
		}
		s := u2.String()
		if tc != s {
			t.Errorf("Expected %v, got %v", tc, s)
		}
	}
}

func TestUint128(t *testing.T) {
	i1, err := uint128.FromString("0")
	if err.Value() != nil {
		t.Errorf("Error creating uint128: %v", err.Value())
	}
	if i1.IsZero() != true {
		t.Errorf("Expected true, got false")
	}
	if i1.BitLen() != 0 {
		t.Errorf("Expected 0, got %v", i1.BitLen())
	}

	i2, err := uint128.FromString("1")
	if err.Value() != nil {
		t.Errorf("Error creating uint128: %v", err.Value())
	}
	if i2.IsZero() != false {
		t.Errorf("Expected false, got true")
	}
	if i2.BitLen() != 1 {
		t.Errorf("Expected 1, got %v", i2.BitLen())
	}

	if i1.Equal(i2) != false {
		t.Errorf("Expected false, got true")
	}

	i3, err := uint128.FromString("123456789012345678901234567890")
	if err.Value() != nil {
		t.Errorf("Error creating uint128: %v", err.Value())
	}
	if i3.IsZero() != false {
		t.Errorf("Expected false, got true")
	}
	if i3.BitLen() != 97 {
		t.Errorf("Expected 97, got %v", i3.BitLen())
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
		c, err := a.Add(b)
		s := c.String()
		if tc.c != s {
			t.Errorf("Expected %v, got %v", tc.c, s)
		}
		if tc.e == "" && err.Value() != nil {
			t.Errorf("Expected no error, got: %v", err.Value())
		}
		if tc.e != "" && (err.Value() == nil || err.Error() != tc.e) {
			t.Errorf("Expected error, got %v", err.Value())
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
		c, err := a.Sub(b)
		s := c.String()
		if tc.c != s {
			t.Errorf("Expected %v, got %v", tc.c, s)
		}
		if tc.e == "" && err.Value() != nil {
			t.Errorf("Expected no error, got: %v", err.Value())
		}
		if tc.e != "" && (err.Value() == nil || err.Error() != tc.e) {
			t.Errorf("Expected error, got %v", err.Value())
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
		c, err := a.Mul(b)
		s := c.String()
		if tc.c != s {
			t.Errorf("Expected %v, got %v", tc.c, s)
		}
		if tc.e == "" && err.Value() != nil {
			t.Errorf("Expected no error, got: %v", err.Value())
		}
		if tc.e != "" && (err.Value() == nil || err.Error() != tc.e) {
			t.Errorf("Expected error, got %v", err.Value())
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
	}

	for _, tc := range testCases {
		a, _ := uint128.FromString(tc.a)
		b, _ := uint128.FromString(tc.b)
		c, err := a.Div(b)
		s := c.String()
		if tc.c != s {
			t.Errorf("Expected %v, got %v", tc.c, s)
		}
		if tc.e == "" && err.Value() != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if tc.e != "" && (err.Value() == nil || err.Error() != tc.e) {
			t.Errorf("Expected error, got %v", err.Value())
		}
	}
}
