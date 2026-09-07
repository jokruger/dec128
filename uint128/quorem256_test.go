package uint128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

func big256(u, carry Uint128) *big.Int {
	x := carry.BigInt()
	x.Lsh(x, 128)
	return x.Add(x, u.BigInt())
}

// checkQuoRem256 compares QuoRem256By128 with math/big for one dividend and divisor.
func checkQuoRem256(t *testing.T, u, carry, v Uint128) {
	t.Helper()
	q, r, s := QuoRem256By128(u, carry, v)
	n := big256(u, carry)
	wantQ, wantR := new(big.Int).QuoRem(n, v.BigInt(), new(big.Int))
	if wantQ.BitLen() > 128 {
		if s != state.Overflow {
			t.Fatalf("QuoRem256By128(%v / %v): want Overflow, got %v", n, v.BigInt(), s)
		}
		return
	}
	if s != state.OK || q.BigInt().Cmp(wantQ) != 0 || r.BigInt().Cmp(wantR) != 0 {
		t.Fatalf("QuoRem256By128(%v / %v) = %v r %v (%v), want %v r %v", n, v.BigInt(), q.BigInt(), r.BigInt(), s, wantQ, wantR)
	}
}

// TestQuoRem256By128EqualTopLimbs covers the trial-quotient step of Knuth's Algorithm D when the top limb of the
// dividend window equals the top limb of the normalized divisor. bits.Div64 panics in that case, so before the fix
// these inputs crashed instead of dividing; dec128 reached them through Div and Mul.
func TestQuoRem256By128EqualTopLimbs(t *testing.T) {
	// constructed: carry and v share the top limb, carry < v
	checkQuoRem256(t, Zero, Uint128{Lo: 3, Hi: 1 << 62}, Uint128{Lo: 5, Hi: 1 << 62})
	// 2^128 / (2^64 + 1): a3 == 0 and a2 == v.Hi after normalization
	checkQuoRem256(t, Zero, One, Uint128{Lo: 1, Hi: 1})
	checkQuoRem256(t, Zero, One, Uint128{Lo: 1 << 62, Hi: 1})
	// maximal values
	checkQuoRem256(t, Max, Uint128{Lo: ^uint64(0), Hi: ^uint64(0) >> 1}, Max)
	checkQuoRem256(t, Max, Uint128{Lo: ^uint64(0) - 1, Hi: ^uint64(0)}, Max)
	checkQuoRem256(t, Max, Uint128{Lo: 0, Hi: ^uint64(0)}, Max)

	r := rand.New(rand.NewSource(20260907))
	rnd := func() Uint128 { return Uint128{Lo: r.Uint64(), Hi: r.Uint64() >> uint(r.Intn(65))} }
	for i := 0; i < 200000; i++ {
		u, v := rnd(), rnd()
		if v.IsZero() {
			continue
		}
		carry := rnd()
		switch r.Intn(4) {
		case 0: // carry just below v
			carry = v
			carry.Lo -= uint64(r.Intn(1000) + 1)
			if carry.Lo > v.Lo {
				carry.Hi--
			}
		case 1: // same top limb, smaller low limb
			carry.Hi = v.Hi
			if carry.Lo >= v.Lo {
				carry.Lo = v.Lo - 1 - uint64(r.Intn(16))
			}
		case 2: // one-limb carry
			carry.Hi = 0
		}
		if carry.Compare(v) >= 0 {
			// the overflow branch, checked as well
			checkQuoRem256(t, u, carry, v)
			continue
		}
		checkQuoRem256(t, u, carry, v)
	}
}
