package unit

import (
	"testing"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/uint128"
)

func TestDecimalCompare(t *testing.T) {
	var a, b dec128.Dec128

	a = dec128.FromString("NaN")
	b = dec128.FromString("NaN")
	if a.Compare(b) != 0 {
		t.Errorf("Expected 0, got %d", a.Compare(b))
	}

	a = dec128.FromString("0")
	b = dec128.FromString("NaN")
	if a.Compare(b) != 1 {
		t.Errorf("Expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("Expected -1, got %d", b.Compare(a))
	}

	a = dec128.FromString("0")
	b = dec128.FromString("0")
	if a.Compare(b) != 0 {
		t.Errorf("Expected 0, got %d", a.Compare(b))
	}

	a = dec128.FromString("1")
	b = dec128.FromString("-1")
	if a.Compare(b) != 1 {
		t.Errorf("Expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("Expected -1, got %d", b.Compare(a))
	}

	a = dec128.FromString(uint128.MaxUint128Str)
	b = dec128.FromString("0.0001")
	if a.Compare(b) != 1 {
		t.Errorf("Expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("Expected -1, got %d", b.Compare(a))
	}

	a = dec128.New(uint128.FromUint64(1000), 1, false)
	b = dec128.New(uint128.FromUint64(10000), 2, false)
	if a.Compare(b) != 0 {
		t.Errorf("Expected 0, got %d", a.Compare(b))
	}

	a = dec128.FromString("123.456")
	b = dec128.FromString("123.4560000")
	if a.Compare(b) != 0 {
		t.Errorf("Expected 0, got %d", a.Compare(b))
	}

	a = dec128.FromString("123.456")
	b = dec128.FromString("123.4560001")
	if a.Compare(b) != -1 {
		t.Errorf("Expected -1, got %d", a.Compare(b))
	}
	if b.Compare(a) != 1 {
		t.Errorf("Expected 1, got %d", b.Compare(a))
	}
}
