package dec128

import (
	"github.com/jokruger/dec128/errors"
)

func (self Dec128) tryAdd(other Dec128) (Dec128, bool) {
	prec := max(self.exp, other.exp)

	a := self.Rescale(prec)
	if a.IsNaN() {
		return a, false
	}

	b := other.Rescale(prec)
	if b.IsNaN() {
		return b, false
	}

	if a.neg == b.neg {
		coef, err := a.coef.Add(b.coef)
		if err != errors.None {
			return NaN(err), false
		}
		return Dec128{coef: coef, exp: prec, neg: a.neg}, true
	}

	switch a.coef.Compare(b.coef) {
	case 1:
		coef, err := a.coef.Sub(b.coef)
		if err != errors.None {
			return NaN(err), false
		}
		return Dec128{coef: coef, exp: prec, neg: a.neg}, true
	case 0:
		return Zero, true
	default:
		coef, err := b.coef.Sub(a.coef)
		if err != errors.None {
			return NaN(err), false
		}
		return Dec128{coef: coef, exp: prec, neg: b.neg}, true
	}
}

func (self Dec128) trySub(other Dec128) (Dec128, bool) {
	prec := max(self.exp, other.exp)

	a := self.Rescale(prec)
	if a.IsNaN() {
		return a, false
	}

	b := other.Rescale(prec)
	if b.IsNaN() {
		return b, false
	}

	if a.neg != b.neg {
		coef, err := a.coef.Add(b.coef)
		if err != errors.None {
			return NaN(err), false
		}
		return Dec128{coef: coef, exp: prec, neg: a.neg}, true
	}

	switch a.coef.Compare(b.coef) {
	case 1:
		coef, err := a.coef.Sub(b.coef)
		if err != errors.None {
			return NaN(err), false
		}
		return Dec128{coef: coef, exp: prec, neg: a.neg}, true
	case 0:
		return Zero, true
	default:
		coef, err := b.coef.Sub(a.coef)
		if err != errors.None {
			return NaN(err), false
		}
		return Dec128{coef: coef, exp: prec, neg: !a.neg}, true
	}
}
