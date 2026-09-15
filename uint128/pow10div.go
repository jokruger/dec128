package uint128

import "math/bits"

// Division by powers of ten without a hardware divide, replaced by multiplications with precomputed constants:
// - one-limb dividends use the Granlund-Montgomery-Warren method:
//   10^k = 2^k * 5^k, so u / 10^k = (u >> k) / 5^k, and the division by the odd 5^k is a multiply-high by
//   m = ceil(2^(64+p) / 5^k) followed by a shift by p. With p the smallest exponent such that 5^k <= 2^(p+k),
//   the error term of m is small enough that no correction step is needed for any u >> k < 2^(64-k).
// - two-limb dividends use Moller and Granlund, "Improved division by invariant integers", algorithm 4:
//   after normalizing the divisor so that its top bit is set, each 64-bit quotient limb costs two multiplies
//   and a few additions, driven by the precomputed reciprocal v = floor((2^128-1)/dn) - 2^64.

type pow10Div struct {
	d  uint64 // divisor 10^k
	m  uint64 // GMW magic
	dn uint64 // normalized divisor dn = d << s
	v  uint64 // Moller-Granlund reciprocal of dn
	p  uint8  // GMW shift
	s  uint8  // normalization shift for dn = d << s
}

var pow10DivTab = [20]pow10Div{
	{},
	{d: 10, m: 0xcccccccccccccccd, dn: 0xa000000000000000, v: 0x9999999999999999, p: 2, s: 60},                   // 10^1
	{d: 100, m: 0x51eb851eb851eb86, dn: 0xc800000000000000, v: 0x47ae147ae147ae14, p: 3, s: 57},                  // 10^2
	{d: 1000, m: 0x20c49ba5e353f7cf, dn: 0xfa00000000000000, v: 0x0624dd2f1a9fbe76, p: 4, s: 54},                 // 10^3
	{d: 10000, m: 0x1a36e2eb1c432ca6, dn: 0x9c40000000000000, v: 0xa36e2eb1c432ca57, p: 6, s: 50},                // 10^4
	{d: 100000, m: 0x0a7c5ac471b47843, dn: 0xc350000000000000, v: 0x4f8b588e368f0846, p: 7, s: 47},               // 10^5
	{d: 1000000, m: 0x0431bde82d7b634e, dn: 0xf424000000000000, v: 0x0c6f7a0b5ed8d36b, p: 8, s: 44},              // 10^6
	{d: 10000000, m: 0x035afe535795e90b, dn: 0x9896800000000000, v: 0xad7f29abcaf48578, p: 10, s: 40},            // 10^7
	{d: 100000000, m: 0x015798ee2308c39e, dn: 0xbebc200000000000, v: 0x5798ee2308c39df9, p: 11, s: 37},           // 10^8
	{d: 1000000000, m: 0x0089705f4136b4a6, dn: 0xee6b280000000000, v: 0x12e0be826d694b2e, p: 12, s: 34},          // 10^9
	{d: 10000000000, m: 0x006df37f675ef6eb, dn: 0x9502f90000000000, v: 0xb7cdfd9d7bdbab7d, p: 14, s: 30},         // 10^10
	{d: 100000000000, m: 0x002bfaffc2f2c92b, dn: 0xba43b74000000000, v: 0x5fd7fe17964955fd, p: 15, s: 27},        // 10^11
	{d: 1000000000000, m: 0x00119799812dea12, dn: 0xe8d4a51000000000, v: 0x19799812dea11197, p: 16, s: 24},       // 10^12
	{d: 10000000000000, m: 0x000e12e13424bb41, dn: 0x9184e72a00000000, v: 0xc25c268497681c26, p: 18, s: 20},      // 10^13
	{d: 100000000000000, m: 0x0005a126e1a84ae7, dn: 0xb5e620f480000000, v: 0x6849b86a12b9b01e, p: 19, s: 17},     // 10^14
	{d: 1000000000000000, m: 0x00024075f3dceac3, dn: 0xe35fa931a0000000, v: 0x203af9ee756159b2, p: 20, s: 14},    // 10^15
	{d: 10000000000000000, m: 0x0001cd2b297d889c, dn: 0x8e1bc9bf04000000, v: 0xcd2b297d889bc2b6, p: 22, s: 10},   // 10^16
	{d: 100000000000000000, m: 0x0000b877aa3236a5, dn: 0xb1a2bc2ec5000000, v: 0x70ef54646d496892, p: 23, s: 7},   // 10^17
	{d: 1000000000000000000, m: 0x000049c97747490f, dn: 0xde0b6b3a76400000, v: 0x2725dd1d243aba0e, p: 24, s: 4},  // 10^18
	{d: 10000000000000000000, m: 0x00003b07929f6da6, dn: 0x8ac7230489e80000, v: 0xd83c94fb6d2ac34a, p: 26, s: 0}, // 10^19
}

// div2by1 divides the 128-bit value {u1, u0} by the normalized divisor dn (top bit set) with reciprocal v,
// returning the 64-bit quotient and the remainder. Requires u1 < dn.
// Moller-Granlund algorithm 4.
func div2by1(u1, u0, dn, v uint64) (q, r uint64) {
	// candidate quotient {qh, ql} = (2^64 + v) * u1 + u0; qh is at most 2 below the truth
	qh, ql := bits.Mul64(v, u1)
	ql, c := bits.Add64(ql, u0, 0)
	qh, _ = bits.Add64(qh, u1, c)

	// step qh up so that it is correct or one too large; the remainder then wraps exactly when it overshot,
	// which r > ql detects
	qh++
	r = u0 - qh*dn
	if r > ql {
		qh--
		r += dn
	}

	// taken with probability about 2^-64
	if r >= dn {
		qh++
		r -= dn
	}

	return qh, r
}

// quoRem64Pow10 returns u / 10^k and u % 10^k for 1 <= k <= 19 with one multiply-high.
func quoRem64Pow10(u uint64, k uint8) (q, r uint64) {
	e := &pow10DivTab[k]
	q, _ = bits.Mul64(u>>k, e.m)
	q >>= e.p
	return q, u - q*e.d
}

// quoRem128Pow10 returns u / 10^k and u % 10^k for a two-limb u and 1 <= k <= 19, with two Moller-Granlund steps
// over the normalized dividend.
func quoRem128Pow10(u Uint128, k uint8) (q Uint128, r uint64) {
	e := &pow10DivTab[k]
	s := uint(e.s)
	// normalize: u << s spans three limbs, the top one below 2^s <= dn (for s == 0, x >> 64 is 0 in Go, so the top limb
	// is 0 and the shifts are no-ops)
	a2 := u.Hi >> (64 - s)
	a1 := u.Hi<<s | u.Lo>>(64-s)
	a0 := u.Lo << s
	q1, r1 := div2by1(a2, a1, e.dn, e.v)
	q0, r0 := div2by1(r1, a0, e.dn, e.v)
	return Uint128{Lo: q0, Hi: q1}, r0 >> s
}

// QuoRem256ByPow10 returns (hi*2^128 + lo) / 10^k and its remainder for 1 <= k <= 19, or ok == false if the quotient
// does not fit in 128 bits. It is the reduction step of the decimal layer's round-to-fit rule for products and sums,
// done with four Moller-Granlund steps instead of a general 256-by-128 division.
func QuoRem256ByPow10(lo, hi Uint128, k uint8) (q Uint128, r uint64, ok bool) {
	e := &pow10DivTab[k]
	// the quotient fits iff hi < 10^k
	if hi.Hi != 0 || hi.Lo >= e.d {
		return Zero, 0, false
	}
	s := uint(e.s)
	// normalize the three live limbs (hi.Hi is zero). The top normalized limb would be hi.Lo >> (64-s), which is zero
	// because hi.Lo < 10^k < 2^(64-s); and the first Moller-Granlund step would return quotient 0 with remainder n2,
	// because hi.Lo << s plus the bits shifted in from lo.Hi is below dn. So two steps suffice.
	n2 := hi.Lo<<s | lo.Hi>>(64-s)
	n1 := lo.Hi<<s | lo.Lo>>(64-s)
	n0 := lo.Lo << s
	q1, r1 := div2by1(n2, n1, e.dn, e.v)
	q0, r0 := div2by1(r1, n0, e.dn, e.v)
	return Uint128{Lo: q0, Hi: q1}, r0 >> s, true
}

// Pow10Reciprocal returns what is needed to divide by 10^k without a hardware divide: the normalized divisor
// dn = 10^k << s, whose top bit is set, the Moller-Granlund reciprocal v of dn, and the normalization shift s.
// ok is false for a k outside 1..19, in which case the other results are zero.
//
// It is exported together with Div2By1Unsafe so that a caller can divide a value of any width by a power of ten with
// the same reciprocals this package uses internally: normalize the dividend by s, run one Div2By1Unsafe per limb from
// the most significant down, carrying the remainder, and shift the final remainder right by s. The decimal layer does
// exactly that for the 384-bit intermediates of its fused operations.
func Pow10Reciprocal(k uint8) (dn, v uint64, s uint8, ok bool) {
	if k == 0 || int(k) >= len(pow10DivTab) {
		return 0, 0, 0, false
	}
	e := &pow10DivTab[k]
	return e.dn, e.v, e.s, true
}

// Div2By1Unsafe divides the 128-bit value {u1, u0} by the normalized divisor dn with reciprocal v, both as returned by
// Pow10Reciprocal, and returns the 64-bit quotient and the remainder.
//
// It requires u1 < dn, which also implies that the quotient fits in one limb. The result is wrong rather than
// reported, as the name says, when that does not hold; SubUnsafe is the same bargain.
func Div2By1Unsafe(u1, u0, dn, v uint64) (q, r uint64) {
	return div2by1(u1, u0, dn, v)
}
