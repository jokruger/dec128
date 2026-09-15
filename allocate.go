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
func allocInputs(d Dec128, ratios []Dec128, scale uint8) (quanta uint128.Uint128, rs uint8, total uint128.Uint128, ok bool) {
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
