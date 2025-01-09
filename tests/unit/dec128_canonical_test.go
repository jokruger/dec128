package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalCanonical(t *testing.T) {
	dec128.SetDefaultPrecision(19)

	type testCase struct {
		i  string
		s  string
		e1 uint8
		e2 uint8
	}

	testCases := [...]testCase{
		{"0", "0", 0, 0},
		{"1", "1", 0, 0},
		{"10", "10", 0, 0},
		{"100", "100", 0, 0},
		{"1.0", "1", 1, 0},
		{"1.00", "1", 2, 0},
		{"1.000", "1", 3, 0},
		{"1.01", "1.01", 2, 2},
		{"1.010", "1.01", 3, 2},
		{"1.001", "1.001", 3, 3},
		{"1.0010", "1.001", 4, 3},
		{"1.00100", "1.001", 5, 3},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalCanonical(%s)", tc.i), func(t *testing.T) {
			d := dec128.FromString(tc.i)
			if d.IsNaN() {
				t.Errorf("Expected no error, got: %v", d.ErrorDetails())
			}
			c := d.Canonical()
			if c.IsNaN() {
				t.Errorf("Expected no error, got: %v", c.ErrorDetails())
			}
			s := c.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
			if d.Precision() != tc.e1 {
				t.Errorf("Expected %d, got: %d", tc.e1, d.Precision())
			}
			if c.Precision() != tc.e2 {
				t.Errorf("Expected %d, got: %d", tc.e2, c.Precision())
			}
		})
	}
}
