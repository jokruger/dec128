package dec128

import (
	"github.com/jokruger/dec128/errors"
)

// Add returns the sum of the Dec128 and the other Dec128.
func (self Dec128) Add(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	r, ok := self.tryAdd(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.tryAdd(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// Sub returns the difference of the Dec128 and the other Dec128.
func (self Dec128) Sub(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	r, ok := self.trySub(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.trySub(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// Mul returns self * other.
func (self Dec128) Mul(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	if self.IsZero() || other.IsZero() {
		return Zero
	}

	r, ok := self.tryMul(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.tryMul(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// Div returns self / other.
func (self Dec128) Div(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	if other.IsZero() {
		return NaN(errors.DivisionByZero)
	}

	if self.IsZero() {
		return Zero
	}

	r, ok := self.tryDiv(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.tryDiv(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}
