package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalBasics(t *testing.T) {
	var d dec128.Dec128

	d = dec128.FromString("NaN")
	if !d.IsNaN() {
		t.Errorf("Expected NaN, got: %s", d.String())
	}
	if d.IsZero() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsNeg() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsPos() {
		t.Errorf("Expected false, got: %s", d.String())
	}

	d = dec128.FromString("0")
	if !d.IsZero() {
		t.Errorf("Expected zero, got: %s", d.String())
	}
	if d.IsNeg() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsPos() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("Expected false, got: %s", d.String())
	}

	d = dec128.FromString("1")
	if d.IsZero() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsNeg() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if !d.IsPos() {
		t.Errorf("Expected true, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("Expected false, got: %s", d.String())
	}

	d = dec128.FromString("-1")
	if d.IsZero() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if !d.IsNeg() {
		t.Errorf("Expected true, got: %s", d.String())
	}
	if d.IsPos() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("Expected false, got: %s", d.String())
	}
}

func TestSign(t *testing.T) {
	type testCase struct {
		a    string
		want int
	}

	testCases := [...]testCase{
		{"1234567890123456789", 1},
		{"123.123", 1},
		{"-123.123", -1},
		{"-123.1234567890123456789", -1},
		{"123.1234567890123456789", 1},
		{"123.1230000000000000001", 1},
		{"-123.1230000000000000001", -1},
		{"123.1230000000000000002", 1},
		{"-123.1230000000000000002", -1},
		{"123.123000000001", 1},
		{"-123.123000000001", -1},
		{"123.1230000", 1},
		{"123.1001", 1},
		{"0", 0},
		{"0.0", 0},
		{"-0", 0},
		{"-0.000", 0},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestSign(%s)", tc.a), func(t *testing.T) {
			a := dec128.FromString(tc.a)
			if a.IsNaN() {
				t.Errorf("Expected no error, got: %v", a.ErrorDetails())
			}

			c := a.Sign()
			if c != tc.want {
				t.Errorf("Expected %d, got: %d", tc.want, c)
			}
		})
	}
}
