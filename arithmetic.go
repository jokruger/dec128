package dec128

import (
	"github.com/jokruger/dec128/errors"
)

func (self Dec128) Add(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	r, ok := self.add(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.add(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}
