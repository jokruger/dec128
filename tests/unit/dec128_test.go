package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalBasics(t *testing.T) {
	dec128.SetDefaultPrecision(19)

	var d dec128.Dec128
	var a dec128.Dec128
	var b dec128.Dec128

	d = dec128.FromString("NaN")
	if !d.IsNaN() {
		t.Errorf("expected NaN, got: %s", d.String())
	}
	if d.IsZero() {
		t.Errorf("expected false, got: %s", d.String())
	}
	if d.IsNegative() {
		t.Errorf("expected false, got: %s", d.String())
	}
	if d.IsPositive() {
		t.Errorf("expected false, got: %s", d.String())
	}

	d = dec128.FromString("0")
	if !d.IsZero() {
		t.Errorf("expected zero, got: %s", d.String())
	}
	if d.IsNegative() {
		t.Errorf("expected false, got: %s", d.String())
	}
	if d.IsPositive() {
		t.Errorf("expected false, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("expected false, got: %s", d.String())
	}

	d = dec128.FromString("1")
	if d.IsZero() {
		t.Errorf("expected false, got: %s", d.String())
	}
	if d.IsNegative() {
		t.Errorf("expected false, got: %s", d.String())
	}
	if !d.IsPositive() {
		t.Errorf("expected true, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("expected false, got: %s", d.String())
	}

	d = dec128.FromString("-1")
	if d.IsZero() {
		t.Errorf("expected false, got: %s", d.String())
	}
	if !d.IsNegative() {
		t.Errorf("expected true, got: %s", d.String())
	}
	if d.IsPositive() {
		t.Errorf("expected false, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("expected false, got: %s", d.String())
	}

	d = dec128.FromString("-123.456")
	if d.Abs().String() != "123.456" {
		t.Errorf("expected 123.456, got: %s", d.String())
	}
	if d.Neg().String() != "123.456" {
		t.Errorf("expected 123.456, got: %s", d.String())
	}

	d = dec128.FromString("123.456")
	if d.Abs().String() != "123.456" {
		t.Errorf("expected 123.456, got: %s", d.String())
	}
	if d.Neg().String() != "-123.456" {
		t.Errorf("expected -123.456, got: %s", d.String())
	}

	a = dec128.FromString("123.456")
	b = dec128.FromString("123.5")
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
