package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalToInt(t *testing.T) {
	dec128.SetDefaultPrecision(19)

	type testCase struct {
		s string
		i int
	}

	testCases := [...]testCase{
		{"0", 0},
		{"1", 1},
		{"-1", -1},
		{"123456.123456", 123456},
		{"1.999", 1},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalToInt(%v)", tc.s), func(t *testing.T) {
			d := dec128.FromString(tc.s)
			i, err := d.Int()
			if err != nil {
				t.Errorf("Int() returned error: %v", err)
			}
			if i != tc.i {
				t.Errorf("Int() returned %v, expected %v", i, tc.i)
			}
		})
	}
}

func TestDecimalToInt64(t *testing.T) {
	dec128.SetDefaultPrecision(19)

	type testCase struct {
		s string
		i int64
	}

	testCases := [...]testCase{
		{"0", 0},
		{"1", 1},
		{"-1", -1},
		{"123456.123456", 123456},
		{"1.999", 1},
		{"9223372036854775807", 9223372036854775807},
		{"-9223372036854775807", -9223372036854775807},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalToInt(%v)", tc.s), func(t *testing.T) {
			d := dec128.FromString(tc.s)
			i, err := d.Int64()
			if err != nil {
				t.Errorf("Int() returned error: %v", err)
			}
			if i != tc.i {
				t.Errorf("Int() returned %v, expected %v", i, tc.i)
			}
		})
	}
}
