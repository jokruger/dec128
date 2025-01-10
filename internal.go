package dec128

import (
	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

var (
	// precalculated StringFixed values for 0 Dec128 in all possible prec
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

	// precalculated array of zero characters
	zeros = [...]byte{'0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0'}
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

// appendString appends the string representation of the decimal to sb. Returns the new slice and whether the decimal contains a decimal point.
func (self Dec128) appendString(sb []byte) ([]byte, bool) {
	buf := [uint128.MaxStrLen]byte{}
	coef := self.coef.StringToBuf(buf[:])

	prec := int(self.exp)
	sz := len(coef)

	if self.neg {
		sb = append(sb, '-')
	}

	if prec == 0 {
		return append(sb, coef...), false
	}

	if prec > sz {
		sb = append(sb, '0', '.')
		sb = append(sb, zeros[:prec-sz]...)
		sb = append(sb, coef...)
	} else if prec == sz {
		sb = append(sb, '0', '.')
		sb = append(sb, coef...)
	} else {
		sb = append(sb, coef[:sz-prec]...)
		sb = append(sb, '.')
		sb = append(sb, coef[sz-prec:]...)
	}

	return sb, true
}

func trimTrailingZeros(sb []byte) []byte {
	i := len(sb)

	for i > 0 && sb[i-1] == '0' {
		i--
	}

	if i > 0 && sb[i-1] == '.' {
		i--
	}

	return sb[:i]
}

func (self Dec128) trySqrt() (Dec128, bool) {
	prec := defaultPrecision
	prec2 := prec * 2
	d := self

	if d.exp > prec2 {
		// scale down to prec2
		coef, err := d.coef.Div(Pow10Uint128[d.exp-prec2])
		if err != errors.None {
			return NaN(err), false
		}
		d = Dec128{coef: coef, exp: prec2, neg: d.neg}
	}

	coef, carry := d.coef.MulCarry(Pow10Uint128[prec2-d.exp])
	if carry.Hi != 0 {
		return NaN(errors.Overflow), false
	}

	// 0 <= coef.bitLen() < 256, so it's safe to convert to uint
	bitLen := uint(coef.BitLen() + carry.BitLen())

	// initial guess = 2^((bitLen + 1) / 2) ≥ √coef
	x := uint128.One.Lsh((bitLen + 1) / 2)

	// Newton-Raphson method
	for {
		// calculate x1 = (x + coef/x) / 2
		y, _, err := uint128.QuoRem256By128(coef, carry, x)
		if err != errors.None {
			return NaN(err), false
		}

		x1, err := x.Add(y)
		if err != errors.None {
			return NaN(err), false
		}

		x1 = x1.Rsh(1)
		if x1.Compare(x) == 0 {
			break
		}

		x = x1
	}

	return Dec128{coef: x, exp: prec}, true
}
