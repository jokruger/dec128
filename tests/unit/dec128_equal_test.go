package unit

import (
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalEqual(t *testing.T) {
	var a, b dec128.Dec128

	a = dec128.FromString("NaN")
	b = dec128.FromString("NaN")
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}

	a = dec128.FromString("0")
	b = dec128.FromString("NaN")
	if a.Equal(b) {
		t.Errorf("Expected false, got true")
	}

	a = dec128.FromString("0")
	b = dec128.FromString("0")
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}

	a = dec128.FromString("1")
	b = dec128.FromString("-1")
	if a.Equal(b) {
		t.Errorf("Expected false, got true")
	}

	a = dec128.FromUint64(1000, 1)
	b = dec128.FromUint64(10000, 2)
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}

	a = dec128.FromUint64(123456, 3)
	b = dec128.FromUint64(123456000, 6)
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}

	a = dec128.FromString("123.456")
	b = dec128.FromString("123.4560000")
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}
}
