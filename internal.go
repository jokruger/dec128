package dec128

import (
	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

var (
	zeroStrs = [...]string{
		"0",                     // 10^0
		"0.0",                   // 10^1
		"0.00",                  // 10^2
		"0.000",                 // 10^3
		"0.0000",                // 10^4
		"0.00000",               // 10^5
		"0.000000",              // 10^6
		"0.0000000",             // 10^7
		"0.00000000",            // 10^8
		"0.000000000",           // 10^9
		"0.0000000000",          // 10^10
		"0.00000000000",         // 10^11
		"0.000000000000",        // 10^12
		"0.0000000000000",       // 10^13
		"0.00000000000000",      // 10^14
		"0.000000000000000",     // 10^15
		"0.0000000000000000",    // 10^16
		"0.00000000000000000",   // 10^17
		"0.000000000000000000",  // 10^18
		"0.0000000000000000000", // 10^19
	}
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

func (self Dec128) tryMul(other Dec128) (Dec128, bool) {
	neg := self.neg != other.neg
	prec := self.exp + other.exp
	rcoef, rcarry := self.coef.MulCarry(other.coef)

	if rcarry.IsZero() {
		r := Dec128{coef: rcoef, exp: prec, neg: neg}
		if prec <= MaxPrecision {
			return r, true
		}
		r = r.Canonical()
		return r, r.exp <= MaxPrecision
	}

	i := prec
	for {
		if i == 0 {
			return NaN(errors.Overflow), false
		}
		q, r, err := uint128.QuoRem256By128(rcoef, rcarry, Pow10Uint128[i])
		if err == errors.None && r.IsZero() {
			return Dec128{coef: q, exp: prec - i, neg: neg}, true
		}
		if err == errors.Overflow {
			return NaN(errors.Overflow), false
		}
		i--
		if prec-i > MaxPrecision {
			return NaN(errors.Overflow), false
		}
	}
}

func (self Dec128) tryDiv(other Dec128) (Dec128, bool) {
	neg := self.neg != other.neg
	factor := other.exp
	prec := self.exp
	if prec < defaultPrecision {
		factor = factor + defaultPrecision - prec
		prec = defaultPrecision
	}
	u, c := self.coef.MulCarry(Pow10Uint128[factor])
	q, _, err := uint128.QuoRem256By128(u, c, other.coef)
	if err != errors.None {
		return NaN(err), false
	}
	return Dec128{coef: q, exp: prec, neg: neg}, true
}
