package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func assertDecimal(s string, isNaN bool, isZero bool, isNegative bool, isPositive bool) error {
	d := dec128.FromString(s)

	if isNaN {
		if !d.IsNaN() {
			return fmt.Errorf("expected NaN, got: %s", d.String())
		}
	} else {
		if d.IsNaN() {
			return fmt.Errorf("expected not NaN")
		}
		if d.String() != s {
			return fmt.Errorf("expected %s, got: %s", s, d.String())
		}
	}

	if isZero && !d.IsZero() {
		return fmt.Errorf("expected zero")
	}
	if !isZero && d.IsZero() {
		return fmt.Errorf("expected not zero")
	}

	if isNegative && !d.IsNegative() {
		return fmt.Errorf("expected negative")
	}
	if !isNegative && d.IsNegative() {
		return fmt.Errorf("expected not negative")
	}

	if isPositive && !d.IsPositive() {
		return fmt.Errorf("expected positive")
	}
	if !isPositive && d.IsPositive() {
		return fmt.Errorf("expected not positive")
	}

	return nil
}

func assertDecimalAbsNeg(s string, abs string, neg string) error {
	d := dec128.FromString("-123.456")
	if d.Abs().String() != abs {
		return fmt.Errorf("expected %s, got: %s", abs, d.String())
	}
	if d.Neg().String() != neg {
		fmt.Errorf("expected %s, got: %s", neg, d.String())
	}
	return nil
}

func TestDecimalBasics(t *testing.T) {
	dec128.SetDefaultPrecision(19)

	type dt struct {
		s      string
		isNaN  bool
		isZero bool
		isNeg  bool
		isPos  bool
	}
	dts := []dt{
		{"NaN", true, false, false, false},
		{"0", false, true, false, false},
		{"0.1", false, false, false, true},
		{"1", false, false, false, true},
		{"-0.1", false, false, true, false},
		{"-1", false, false, true, false},
	}
	for _, e := range dts {
		if err := assertDecimal(e.s, e.isNaN, e.isZero, e.isNeg, e.isPos); err != nil {
			t.Errorf("assertDecimal failed for %s: %s", e.s, err.Error())
		}
	}

	type dtan struct {
		s   string
		abs string
		neg string
	}
	dtans := []dtan{
		{"-123.456", "123.456", "123.456"},
		{"123.456", "123.456", "-123.456"},
	}
	for _, e := range dtans {
		if err := assertDecimalAbsNeg(e.s, e.abs, e.neg); err != nil {
			t.Errorf("assertDecimalAbsNeg failed for %s: %s", e.s, err.Error())
		}
	}

	a := dec128.FromString("123.456")
	b := dec128.FromString("123.5")
	if a.Compare(b) != -1 {
		t.Errorf("expected -1, got: %d", a.Compare(b))
	}
	if b.Compare(a) != 1 {
		t.Errorf("expected 1, got: %d", b.Compare(a))
	}
	if a.Compare(a) != 0 {
		t.Errorf("expected 0, got: %d", a.Compare(a))
	}
	if !a.LessThan(b) {
		t.Errorf("expected true, got: %t", a.LessThan(b))
	}
	if b.LessThan(a) {
		t.Errorf("expected false, got: %t", b.LessThan(a))
	}
	if a.LessThan(a) {
		t.Errorf("expected false, got: %t", a.LessThan(a))
	}
	if a.GreaterThan(b) {
		t.Errorf("expected false, got: %t", a.GreaterThan(b))
	}
	if !b.GreaterThan(a) {
		t.Errorf("expected true, got: %t", b.GreaterThan(a))
	}
	if a.GreaterThan(a) {
		t.Errorf("expected false, got: %t", a.GreaterThan(a))
	}
	if !a.LessThanOrEqual(b) {
		t.Errorf("expected true, got: %t", a.LessThanOrEqual(b))
	}
	if !a.GreaterThanOrEqual(a) {
		t.Errorf("expected true, got: %t", a.GreaterThanOrEqual(a))
	}
}

func TestSign(t *testing.T) {
	dec128.SetDefaultPrecision(19)

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
				t.Errorf("expected no error, got: %v", a.ErrorDetails())
			}

			c := a.Sign()
			if c != tc.want {
				t.Errorf("expected %d, got: %d", tc.want, c)
			}
		})
	}
}
