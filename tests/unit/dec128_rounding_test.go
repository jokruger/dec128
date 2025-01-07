package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalTrunc(t *testing.T) {
	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"0", 1, "0"},
		{"1.12345", 4, "1.1234"},
		{"1.12335", 4, "1.1233"},
		{"1234567890123456789012345678912345678.5", 0, "1234567890123456789012345678912345678"},
		{"-1234567890123456789012345678912345678.5", 0, "-1234567890123456789012345678912345678"},
		{"9999999999999999999.9999999999999999999", 0, "9999999999999999999"},
		{"-9999999999999999999.9999999999999999999", 0, "-9999999999999999999"},
		{"123.456000", 0, "123"},
		{"123.456000", 1, "123.4"},
		{"123.456000", 2, "123.45"},
		{"123.456000", 3, "123.456"},
		{"123.456000", 4, "123.4560"},
		{"123.456000", 5, "123.45600"},
		{"123.456000", 6, "123.456000"},
		{"123.456000", 7, "123.456000"},
		{"-123.456000", 0, "-123"},
		{"-123.456000", 1, "-123.4"},
		{"-123.456000", 2, "-123.45"},
		{"-123.456000", 3, "-123.456"},
		{"-123.456000", 4, "-123.4560"},
		{"-123.456000", 5, "-123.45600"},
		{"-123.456000", 6, "-123.456000"},
		{"-123.456000", 7, "-123.456000"},
		{"123.1234567890987654321", 0, "123"},
		{"123.1234567890987654321", 1, "123.1"},
		{"123.1234567890987654321", 2, "123.12"},
		{"123.1234567890987654321", 3, "123.123"},
		{"123.1234567890987654321", 4, "123.1234"},
		{"123.1234567890987654321", 5, "123.12345"},
		{"123.1234567890987654321", 6, "123.123456"},
		{"123.1234567890987654321", 7, "123.1234567"},
		{"123.1234567890987654321", 8, "123.12345678"},
		{"123.1234567890987654321", 9, "123.123456789"},
		{"123.1234567890987654321", 10, "123.1234567890"},
		{"123.1234567890987654321", 11, "123.12345678909"},
		{"123.1234567890987654321", 12, "123.123456789098"},
		{"123.1234567890987654321", 13, "123.1234567890987"},
		{"123.1234567890987654321", 14, "123.12345678909876"},
		{"123.1234567890987654321", 15, "123.123456789098765"},
		{"123.1234567890987654321", 16, "123.1234567890987654"},
		{"123.1234567890987654321", 17, "123.12345678909876543"},
		{"123.1234567890987654321", 18, "123.123456789098765432"},
		{"123.1234567890987654321", 19, "123.1234567890987654321"},
		{"123.1234567890987654321", 20, "123.1234567890987654321"},
		{"-123.1234567890987654321", 0, "-123"},
		{"-123.1234567890987654321", 1, "-123.1"},
		{"-123.1234567890987654321", 2, "-123.12"},
		{"-123.1234567890987654321", 3, "-123.123"},
		{"-123.1234567890987654321", 4, "-123.1234"},
		{"-123.1234567890987654321", 5, "-123.12345"},
		{"-123.1234567890987654321", 6, "-123.123456"},
		{"-123.1234567890987654321", 7, "-123.1234567"},
		{"-123.1234567890987654321", 8, "-123.12345678"},
		{"-123.1234567890987654321", 9, "-123.123456789"},
		{"-123.1234567890987654321", 10, "-123.1234567890"},
		{"-123.1234567890987654321", 11, "-123.12345678909"},
		{"-123.1234567890987654321", 12, "-123.123456789098"},
		{"-123.1234567890987654321", 13, "-123.1234567890987"},
		{"-123.1234567890987654321", 14, "-123.12345678909876"},
		{"-123.1234567890987654321", 15, "-123.123456789098765"},
		{"-123.1234567890987654321", 16, "-123.1234567890987654"},
		{"-123.1234567890987654321", 17, "-123.12345678909876543"},
		{"-123.1234567890987654321", 18, "-123.123456789098765432"},
		{"-123.1234567890987654321", 19, "-123.1234567890987654321"},
		{"-123.1234567890987654321", 20, "-123.1234567890987654321"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalTrunc(%v)", tc), func(t *testing.T) {
			d := dec128.FromString(tc.i)
			s := d.Trunc(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("Trunc(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}

}

func TestDecimalCeil(t *testing.T) {
	type testCase struct {
		i string
		s string
	}

	testCases := [...]testCase{
		{"Nan", "NaN"},
		{"0", "0"},
		{"123.456000", "124"},
		{"-123.456000", "-123"},
		{"123.1234567890987654321", "124"},
		{"-123.1234567890987654321", "-123"},
		{"123.12454", "124"},
		{"123.1235", "124"},
		{"123.1245", "124"},
		{"-123.12354", "-123"},
		{"-123.12454", "-123"},
		{"-123.1235", "-123"},
		{"-123.1245", "-123"},
		{"1.12345", "2"},
		{"1.12335", "2"},
		{"1.5", "2"},
		{"2.5", "3"},
		{"1", "1"},
		{"-1", "-1"},
		{"-1.5", "-1"},
		{"-2.5", "-2"},
		{"9999999999999999999.9999999999999999999", "10000000000000000000"},
		{"-9999999999999999999.9999999999999999999", "-9999999999999999999"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalCeil(%v)", tc), func(t *testing.T) {
			d := dec128.FromString(tc.i)
			s := d.Ceil().StringFixed()
			if s != tc.s {
				t.Errorf("Ceil(%v) = %v, want %v", tc.i, s, tc.s)
			}
		})
	}
}

func TestDecimalFloor(t *testing.T) {
	type testCase struct {
		i string
		s string
	}

	testCases := [...]testCase{
		{"Nan", "NaN"},
		{"0", "0"},
		{"123.456000", "123"},
		{"123.1234567890987654321", "123"},
		{"123.12354", "123"},
		{"123.12454", "123"},
		{"123.1235", "123"},
		{"123.1245", "123"},
		{"-123.456000", "-124"},
		{"-123.1234567890987654321", "-124"},
		{"-123.12354", "-124"},
		{"-123.12454", "-124"},
		{"-123.1235", "-124"},
		{"-123.1245", "-124"},
		{"1.12345", "1"},
		{"1.12335", "1"},
		{"1.5", "1"},
		{"2.5", "2"},
		{"1", "1"},
		{"-1.5", "-2"},
		{"-2.5", "-3"},
		{"-1", "-1"},
		{"9999999999999999999.9999999999999999999", "9999999999999999999"},
		{"-9999999999999999999.9999999999999999999", "-10000000000000000000"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalFloor(%v)", tc), func(t *testing.T) {
			d := dec128.FromString(tc.i)
			s := d.Floor().StringFixed()
			if s != tc.s {
				t.Errorf("Floor(%v) = %v, want %v", tc.i, s, tc.s)
			}
		})
	}
}

func TestDecimalRoundHalfTowardZero(t *testing.T) {
	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"1.12345", 4, "1.1234"},
		{"1.12335", 4, "1.1233"},
		{"1.5", 0, "1"},
		{"-1.5", 0, "-1"},
		{"123.456000", 0, "123"},
		{"123.456000", 1, "123.5"},
		{"123.456000", 2, "123.46"},
		{"123.456000", 3, "123.456"},
		{"123.456000", 4, "123.4560"},
		{"123.456000", 5, "123.45600"},
		{"123.456000", 6, "123.456000"},
		{"123.456000", 7, "123.456000"},
		{"-123.456000", 0, "-123"},
		{"-123.456000", 1, "-123.5"},
		{"-123.456000", 2, "-123.46"},
		{"-123.456000", 3, "-123.456"},
		{"-123.456000", 4, "-123.4560"},
		{"-123.456000", 5, "-123.45600"},
		{"-123.456000", 6, "-123.456000"},
		{"-123.456000", 7, "-123.456000"},
		{"123.1234567890987654321", 0, "123"},
		{"123.1234567890987654321", 1, "123.1"},
		{"123.1234567890987654321", 2, "123.12"},
		{"123.1234567890987654321", 3, "123.123"},
		{"123.1234567890987654321", 4, "123.1235"},
		{"123.1234567890987654321", 5, "123.12346"},
		{"123.1234567890987654321", 6, "123.123457"},
		{"123.1234567890987654321", 7, "123.1234568"},
		{"123.1234567890987654321", 8, "123.12345679"},
		{"123.1234567890987654321", 9, "123.123456789"},
		{"123.1234567890987654321", 10, "123.1234567891"},
		{"123.1234567890987654321", 11, "123.12345678910"},
		{"123.1234567890987654321", 12, "123.123456789099"},
		{"123.1234567890987654321", 13, "123.1234567890988"},
		{"123.1234567890987654321", 14, "123.12345678909877"},
		{"123.1234567890987654321", 15, "123.123456789098765"},
		{"123.1234567890987654321", 16, "123.1234567890987654"},
		{"123.1234567890987654321", 17, "123.12345678909876543"},
		{"123.1234567890987654321", 18, "123.123456789098765432"},
		{"123.1234567890987654321", 19, "123.1234567890987654321"},
		{"123.1234567890987654321", 20, "123.1234567890987654321"},
		{"-123.1234567890987654321", 0, "-123"},
		{"-123.1234567890987654321", 1, "-123.1"},
		{"-123.1234567890987654321", 2, "-123.12"},
		{"-123.1234567890987654321", 3, "-123.123"},
		{"-123.1234567890987654321", 4, "-123.1235"},
		{"-123.1234567890987654321", 5, "-123.12346"},
		{"-123.1234567890987654321", 6, "-123.123457"},
		{"-123.1234567890987654321", 7, "-123.1234568"},
		{"-123.1234567890987654321", 8, "-123.12345679"},
		{"-123.1234567890987654321", 9, "-123.123456789"},
		{"-123.1234567890987654321", 10, "-123.1234567891"},
		{"-123.1234567890987654321", 11, "-123.12345678910"},
		{"-123.1234567890987654321", 12, "-123.123456789099"},
		{"-123.1234567890987654321", 13, "-123.1234567890988"},
		{"-123.1234567890987654321", 14, "-123.12345678909877"},
		{"-123.1234567890987654321", 15, "-123.123456789098765"},
		{"-123.1234567890987654321", 16, "-123.1234567890987654"},
		{"-123.1234567890987654321", 17, "-123.12345678909876543"},
		{"-123.1234567890987654321", 18, "-123.123456789098765432"},
		{"-123.1234567890987654321", 19, "-123.1234567890987654321"},
		{"-123.1234567890987654321", 20, "-123.1234567890987654321"},
		{"123.12354", 3, "123.124"},
		{"123.12454", 3, "123.125"},
		{"123.1235", 3, "123.123"},
		{"123.1245", 3, "123.124"},
		{"2.5", 0, "2"},
		{"1", 0, "1"},
		{"-123.12354", 3, "-123.124"},
		{"-123.12454", 3, "-123.125"},
		{"-123.1235", 3, "-123.123"},
		{"-123.1245", 3, "-123.124"},
		{"-2.5", 0, "-2"},
		{"-1", 0, "-1"},
		{"9999999999999999999.9999999999999999999", 3, "10000000000000000000.000"},
		{"-9999999999999999999.9999999999999999999", 3, "-10000000000000000000.000"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalRoundHalfTowardZero(%v)", tc), func(t *testing.T) {
			d := dec128.FromString(tc.i)
			s := d.RoundHalfTowardZero(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundHalfTowardZero(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}

func TestDecimalRoundHalfAwayFromZero(t *testing.T) {
	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"1.12345", 4, "1.1235"},
		{"1.12335", 4, "1.1234"},
		{"1.5", 0, "2"},
		{"-1.5", 0, "-2"},
		{"123.456000", 0, "123"},
		{"123.456000", 1, "123.5"},
		{"123.456000", 2, "123.46"},
		{"123.456000", 3, "123.456"},
		{"123.456000", 4, "123.4560"},
		{"123.456000", 5, "123.45600"},
		{"123.456000", 6, "123.456000"},
		{"123.456000", 7, "123.456000"},
		{"-123.456000", 0, "-123"},
		{"-123.456000", 1, "-123.5"},
		{"-123.456000", 2, "-123.46"},
		{"-123.456000", 3, "-123.456"},
		{"-123.456000", 4, "-123.4560"},
		{"-123.456000", 5, "-123.45600"},
		{"-123.456000", 6, "-123.456000"},
		{"-123.456000", 7, "-123.456000"},
		{"123.1234567890987654321", 0, "123"},
		{"123.1234567890987654321", 1, "123.1"},
		{"123.1234567890987654321", 2, "123.12"},
		{"123.1234567890987654321", 3, "123.123"},
		{"123.1234567890987654321", 4, "123.1235"},
		{"123.1234567890987654321", 5, "123.12346"},
		{"123.1234567890987654321", 6, "123.123457"},
		{"123.1234567890987654321", 7, "123.1234568"},
		{"123.1234567890987654321", 8, "123.12345679"},
		{"123.1234567890987654321", 9, "123.123456789"},
		{"123.1234567890987654321", 10, "123.1234567891"},
		{"123.1234567890987654321", 11, "123.12345678910"},
		{"123.1234567890987654321", 12, "123.123456789099"},
		{"123.1234567890987654321", 13, "123.1234567890988"},
		{"123.1234567890987654321", 14, "123.12345678909877"},
		{"123.1234567890987654321", 15, "123.123456789098765"},
		{"123.1234567890987654321", 16, "123.1234567890987654"},
		{"123.1234567890987654321", 17, "123.12345678909876543"},
		{"123.1234567890987654321", 18, "123.123456789098765432"},
		{"123.1234567890987654321", 19, "123.1234567890987654321"},
		{"123.1234567890987654321", 20, "123.1234567890987654321"},
		{"-123.1234567890987654321", 0, "-123"},
		{"-123.1234567890987654321", 1, "-123.1"},
		{"-123.1234567890987654321", 2, "-123.12"},
		{"-123.1234567890987654321", 3, "-123.123"},
		{"-123.1234567890987654321", 4, "-123.1235"},
		{"-123.1234567890987654321", 5, "-123.12346"},
		{"-123.1234567890987654321", 6, "-123.123457"},
		{"-123.1234567890987654321", 7, "-123.1234568"},
		{"-123.1234567890987654321", 8, "-123.12345679"},
		{"-123.1234567890987654321", 9, "-123.123456789"},
		{"-123.1234567890987654321", 10, "-123.1234567891"},
		{"-123.1234567890987654321", 11, "-123.12345678910"},
		{"-123.1234567890987654321", 12, "-123.123456789099"},
		{"-123.1234567890987654321", 13, "-123.1234567890988"},
		{"-123.1234567890987654321", 14, "-123.12345678909877"},
		{"-123.1234567890987654321", 15, "-123.123456789098765"},
		{"-123.1234567890987654321", 16, "-123.1234567890987654"},
		{"-123.1234567890987654321", 17, "-123.12345678909876543"},
		{"-123.1234567890987654321", 18, "-123.123456789098765432"},
		{"-123.1234567890987654321", 19, "-123.1234567890987654321"},
		{"-123.1234567890987654321", 20, "-123.1234567890987654321"},
		{"123.12354", 3, "123.124"},
		{"123.12454", 3, "123.125"},
		{"123.1235", 3, "123.124"},
		{"123.1245", 3, "123.125"},
		{"2.5", 0, "3"},
		{"1", 0, "1"},
		{"-123.12354", 3, "-123.124"},
		{"-123.12454", 3, "-123.125"},
		{"-123.1235", 3, "-123.124"},
		{"-123.1245", 3, "-123.125"},
		{"-2.5", 0, "-3"},
		{"-1", 0, "-1"},
		{"9999999999999999999.9999999999999999999", 3, "10000000000000000000.000"},
		{"-9999999999999999999.9999999999999999999", 3, "-10000000000000000000.000"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalRoundHalfAwayFromZero(%v)", tc), func(t *testing.T) {
			d := dec128.FromString(tc.i)
			s := d.RoundHalfAwayFromZero(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundHalfAwayFromZero(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}

func TestDecimalRoundAwayFromZero(t *testing.T) {
	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"1.12345", 4, "1.1235"},
		{"1.12335", 4, "1.1234"},
		{"1.5", 0, "2"},
		{"-1.5", 0, "-2"},
		{"1.12", 1, "1.2"},
		{"1.15", 1, "1.2"},
		{"-1.12", 1, "-1.2"},
		{"-1.15", 1, "-1.2"},
		{"9999999999999999999.9999999999999999999", 3, "10000000000000000000.000"},
		{"-9999999999999999999.9999999999999999999", 3, "-10000000000000000000.000"},
		{"123.456000", 0, "124"},
		{"123.456000", 4, "123.4560"},
		{"123.1234567890987654321", 6, "123.123457"},
		{"-123.456000", 7, "-123.456000"},
		{"-123.1234567890987654321", 7, "-123.1234568"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalRoundAwayFromZero(%v)", tc), func(t *testing.T) {
			d := dec128.FromString(tc.i)
			s := d.RoundAwayFromZero(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundAwayFromZero(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}
