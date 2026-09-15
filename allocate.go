package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Splitting an amount so that nothing is lost.
//
// This is the most-used money operation in a core-banking system - fee splitting, the waterfall across principal,
// interest and fees, syndicated participation, VAT apportionment - and it is pure integer arithmetic, so it belongs
// here rather than in a layer above. The rule is the largest-remainder method: give every party the whole quanta its
// share is worth, then hand the quanta left over to the parties whose fractions were largest, one each, ties to the
// lowest index. The post-condition is exact: the shares add up to the amount, digit for digit.

// allocScratch is the number of shares whose remainders are kept on the stack. Beyond it an allocation takes one
// working slice, which a fee split or a payment waterfall never reaches.
const allocScratch = 32

// Allocate divides d into shares proportional to ratios, at the given scale, losing nothing: the shares sum to
// exactly d.
//
// Every share gets the whole number of quanta its exact proportion is worth, and the quanta left over by those
// truncations go one at a time to the shares with the largest fractional parts, ties to the lowest index. There is no
// rounding mode to choose: the shares are as close to their exact proportions as a set of multiples of the quantum
// can be, and the tie rule makes the result the same whoever runs it.
//
// ok is false, and the shares are not produced, when the split cannot be exact or cannot be represented:
//
//   - d is NaN, a ratio is NaN, or ratios is empty;
//   - scale is above MaxScale, or below d's own scale, which would mean d is not a whole number of quanta;
//   - a ratio is negative, or they are all zero;
//   - d in quanta, a ratio at the common scale of the ratios, or their total does not fit a coefficient.
//
// It reads no process-global configuration.
func (d Dec128) Allocate(ratios []Dec128, scale uint8) ([]Dec128, bool) {
	return d.AppendAllocate(nil, ratios, scale)
}

// AppendAllocate is Allocate appending to dst, for a caller that keeps the slice: pass shares[:0] and a split of up
// to 32 ways costs no allocation. dst is returned unchanged when the split fails.
func (d Dec128) AppendAllocate(dst []Dec128, ratios []Dec128, scale uint8) ([]Dec128, bool) {
	quanta, rs, total, ok := allocInputs(d, ratios, scale)
	if !ok {
		return dst, false
	}

	var stack [allocScratch]uint128.Uint128
	rems := stack[:0]
	if len(ratios) > allocScratch {
		rems = make([]uint128.Uint128, 0, len(ratios))
	}

	// Each share is floor(quanta * weight / total), and the remainders record how much of a quantum each was owed
	// beyond that. The numerator is a 256-bit product of two coefficients and the divisor is one, which is the
	// division uint128 does exactly; the quotient fits because the total is at least any single weight.
	base := len(dst)
	left := quanta
	for i := range ratios {
		w, _ := ratios[i].coef.Mul(Pow10Uint128[rs-ratios[i].scale])
		lo, hi := quanta.MulCarry(w)
		q, r, _ := uint128.QuoRem256By128(lo, hi, total)
		rems = append(rems, r)
		left = uint128.SubUnsafe(left, q)
		dst = append(dst, Dec128{coef: q, scale: scale, state: d.state})
	}

	// The leftover is below the number of shares, and strictly more remainders are nonzero than there are quanta to
	// hand out (their sum is the leftover times the total, and each is below the total), so taking the largest one
	// each time always finds a real fraction to reward and never revisits a share.
	for ; !left.IsZero(); left, _ = left.Sub64(1) {
		best := 0
		for i := 1; i < len(rems); i++ {
			if rems[i].Compare(rems[best]) > 0 {
				best = i
			}
		}
		rems[best] = uint128.Zero
		dst[base+best].coef, _ = dst[base+best].coef.AddCarry(uint128.One)
	}

	for i := base; i < len(dst); i++ {
		if dst[i].coef.IsZero() {
			dst[i].state = state.Default // a zero is never negative
		}
	}

	return dst, true
}

// Split divides d into n equal shares at the given scale, losing nothing: the shares sum to exactly d and the first
// few carry the extra quantum. It is Allocate with equal ratios, and fails on the same conditions plus a
// non-positive n.
func (d Dec128) Split(n int, scale uint8) ([]Dec128, bool) {
	return d.AppendSplit(nil, n, scale)
}

// AppendSplit is Split appending to dst.
func (d Dec128) AppendSplit(dst []Dec128, n int, scale uint8) ([]Dec128, bool) {
	quanta, ok := allocQuanta(d, scale)
	if !ok || n <= 0 {
		return dst, false
	}

	q, r, _ := quanta.QuoRem64(uint64(n))
	for i := range n {
		c := q
		if uint64(i) < r {
			c, _ = c.AddCarry(uint128.One)
		}
		st := d.state
		if c.IsZero() {
			st = state.Default // a zero is never negative
		}
		dst = append(dst, Dec128{coef: c, scale: scale, state: st})
	}

	return dst, true
}

// allocQuanta returns |d| counted in quanta of the given scale. That count is exact only when the scale is at least
// d's own, and representable only when it still fits a coefficient.
func allocQuanta(d Dec128, scale uint8) (uint128.Uint128, bool) {
	if d.state >= state.Error || scale > MaxScale || scale < d.scale {
		return uint128.Zero, false
	}
	q, s := d.coef.Mul(Pow10Uint128[scale-d.scale])
	if s >= state.Error {
		return uint128.Zero, false
	}
	return q, true
}

// allocInputs validates an allocation and reduces it to integers: the amount in quanta, the common scale of the
// ratios, and their total. Every weight and the total have to fit a coefficient, which is what keeps each share's
// numerator inside the 256-bit dividend.
func allocInputs(
	d Dec128,
	ratios []Dec128,
	scale uint8,
) (quanta uint128.Uint128, rs uint8, total uint128.Uint128, ok bool) {
	if len(ratios) == 0 {
		return uint128.Zero, 0, uint128.Zero, false
	}
	quanta, ok = allocQuanta(d, scale)
	if !ok {
		return uint128.Zero, 0, uint128.Zero, false
	}

	for _, r := range ratios {
		if r.state >= state.Error || r.state == state.Neg {
			return uint128.Zero, 0, uint128.Zero, false
		}
		rs = max(rs, r.scale)
	}

	for _, r := range ratios {
		w, s := r.coef.Mul(Pow10Uint128[rs-r.scale])
		if s >= state.Error {
			return uint128.Zero, 0, uint128.Zero, false
		}
		if total, s = total.Add(w); s >= state.Error {
			return uint128.Zero, 0, uint128.Zero, false
		}
	}
	if total.IsZero() {
		return uint128.Zero, 0, uint128.Zero, false
	}

	return quanta, rs, total, true
}

// AllocateResidual divides d into shares proportional to ratios at the given scale, rounding every share but one
// with mode and giving the one at index residual whatever is left, so that the shares still sum to exactly d.
//
// This is the other convention a ledger splits by, and the one an amortization schedule and a syndicated facility
// use: every party gets its proportion rounded the agreed way, and the difference that rounding creates goes to a
// named party - the last installment, the lead bank, the house account - rather than to whoever happened to have the
// largest fraction. Allocate is the choice when no party is named, since it keeps every share within a quantum of
// its proportion; here only the other shares have that property. The residual may be up to len(ratios)-1 quanta away
// from its own proportion, and may even come out with the opposite sign to d when d is a few quanta and the others
// round away from zero.
//
// ok is false, and the shares are not produced, on every condition Allocate fails on, and also when residual is not
// an index of ratios, when mode is undefined, or when mode is ROUND_NAN and a share is not exact - a NaN cannot be
// returned through a bool, so ROUND_NAN reads here as "refuse unless the split is exact".
//
// It reads no process-global configuration.
func (d Dec128) AllocateResidual(ratios []Dec128, scale uint8, residual int, mode RoundingMode) ([]Dec128, bool) {
	return d.AppendAllocateResidual(nil, ratios, scale, residual, mode)
}

// AppendAllocateResidual is AllocateResidual appending to dst, for a caller that keeps the slice: pass shares[:0] and
// the split costs no allocation. dst is returned unchanged when the split fails.
func (d Dec128) AppendAllocateResidual(
	dst []Dec128,
	ratios []Dec128,
	scale uint8,
	residual int,
	mode RoundingMode,
) ([]Dec128, bool) {
	quanta, rs, total, ok := allocInputs(d, ratios, scale)
	if !ok || residual < 0 || residual >= len(ratios) || !mode.IsValid() {
		return dst, false
	}

	// Every share but the residual is its exact proportion rounded with mode. Their sum is within len(ratios)-1
	// quanta of the whole on either side, which can pass 2^128 when d is near the largest coefficient, so it is
	// accumulated in 256 bits.
	base := len(dst)
	var sumLo, sumHi uint128.Uint128
	for i := range ratios {
		if i == residual {
			dst = append(dst, Dec128{scale: scale}) // filled in below
			continue
		}
		w, _ := ratios[i].coef.Mul(Pow10Uint128[rs-ratios[i].scale])
		lo, hi := quanta.MulCarry(w)
		q, r, _ := uint128.QuoRem256By128(lo, hi, total)
		if mode == ROUND_NAN && !r.IsZero() {
			return dst[:base], false
		}
		if up, _ := roundUp(q, r, total, d.state, mode); up {
			// This cannot carry out. A weight below the total makes the quotient strictly below the quanta, which
			// is itself a coefficient; a weight equal to the total makes the quotient the quanta exactly and the
			// remainder zero, so there is nothing to round up.
			q, _ = q.AddCarry(uint128.One)
		}

		var carry uint64
		sumLo, carry = sumLo.AddCarry(q)
		sumHi, _ = sumHi.AddCarry(uint128.Uint128{Lo: carry})

		st := d.state
		if q.IsZero() {
			st = state.Default // a zero is never negative
		}
		dst = append(dst, Dec128{coef: q, scale: scale, state: st})
	}

	coef, st := residualOf(quanta, sumLo, sumHi, d.state)
	dst[base+residual] = Dec128{coef: coef, scale: scale, state: st}

	return dst, true
}

// SplitResidual divides d into n equal shares at the given scale: each share is d/n rounded with mode except the one
// at index residual, which takes what is left, so the shares sum to exactly d. It is AllocateResidual with equal
// ratios and fails on the same conditions, plus a non-positive n.
func (d Dec128) SplitResidual(n int, scale uint8, residual int, mode RoundingMode) ([]Dec128, bool) {
	return d.AppendSplitResidual(nil, n, scale, residual, mode)
}

// AppendSplitResidual is SplitResidual appending to dst.
func (d Dec128) AppendSplitResidual(
	dst []Dec128,
	n int,
	scale uint8,
	residual int,
	mode RoundingMode,
) ([]Dec128, bool) {
	quanta, ok := allocQuanta(d, scale)
	if !ok || n <= 0 || residual < 0 || residual >= n || !mode.IsValid() {
		return dst, false
	}

	// Equal shares take one rounding decision between them.
	share, r, _ := quanta.QuoRem64(uint64(n))
	if mode == ROUND_NAN && r != 0 {
		return dst, false
	}
	divisor := uint128.FromUint64(uint64(n))
	if up, _ := roundUp(share, uint128.FromUint64(r), divisor, d.state, mode); up {
		// As in AppendAllocateResidual this cannot carry out: a quotient equal to the quanta means n is one, and
		// then the remainder is zero and there is nothing to round up.
		share, _ = share.AddCarry(uint128.One)
	}

	sumLo, sumHi := share.MulCarry(uint128.FromUint64(uint64(n - 1)))
	coef, rst := residualOf(quanta, sumLo, sumHi, d.state)

	st := d.state
	if share.IsZero() {
		st = state.Default // a zero is never negative
	}
	for i := range n {
		if i == residual {
			dst = append(dst, Dec128{coef: coef, scale: scale, state: rst})
			continue
		}
		dst = append(dst, Dec128{coef: share, scale: scale, state: st})
	}

	return dst, true
}

// residualOf returns what is left of the whole, given in quanta, after the other shares have taken the 256-bit total
// (sumLo, sumHi) of them: the magnitude of the residual share and its sign. The other shares carry the sign st, so a
// sum that overshoots the whole leaves a residual of the opposite sign.
//
// The gap is always a coefficient wide at most, whichever way it falls: the exact proportions sum to the whole, and
// each rounded share is within one quantum of its own, so the sum is within the share count of the whole. The
// subtraction below therefore always lands in the low word.
func residualOf(quanta, sumLo, sumHi uint128.Uint128, st state.State) (uint128.Uint128, state.State) {
	if compare256(sumLo, sumHi, quanta, uint128.Zero) <= 0 {
		// sumHi is zero here: quanta has no high word for it to have compared equal against
		coef := uint128.SubUnsafe(quanta, sumLo)
		if coef.IsZero() {
			return coef, state.Default // a zero is never negative
		}
		return coef, st
	}

	lo, borrow := sumLo.SubBorrow(quanta)
	_, _ = sumHi.SubBorrow(uint128.Uint128{Lo: borrow})
	if st == state.Neg {
		return lo, state.Default
	}
	return lo, state.Neg
}
