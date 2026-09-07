package uint128

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

var two128 = new(big.Int).Lsh(big.NewInt(1), 128)

func toBig(u Uint128) *big.Int {
	b := new(big.Int).SetUint64(u.Hi)
	b.Lsh(b, 64)
	return b.Or(b, new(big.Int).SetUint64(u.Lo))
}

func randUint128(r *rand.Rand) Uint128 {
	// Mix widths so that one-limb, two-limb and near-max operands all appear.
	switch r.Intn(4) {
	case 0:
		return Uint128{Lo: r.Uint64()}
	case 1:
		return Uint128{Lo: r.Uint64() % 1000}
	case 2:
		return Uint128{Lo: math.MaxUint64 - uint64(r.Intn(1000)), Hi: math.MaxUint64 - uint64(r.Intn(1000))}
	default:
		return Uint128{Lo: r.Uint64(), Hi: r.Uint64()}
	}
}

func checkAddCarry(t *testing.T, a, b Uint128) {
	t.Helper()
	sum, carry := a.AddCarry(b)
	want := new(big.Int).Add(toBig(a), toBig(b))
	wantCarry := uint64(0)
	if want.Cmp(two128) >= 0 {
		wantCarry = 1
		want.Sub(want, two128)
	}
	if toBig(sum).Cmp(want) != 0 || carry != wantCarry {
		t.Errorf("AddCarry(%s, %s) = (%s, %d), want (%s, %d)", a, b, sum, carry, want, wantCarry)
	}
}

func checkSubBorrow(t *testing.T, a, b Uint128) {
	t.Helper()
	diff, borrow := a.SubBorrow(b)
	want := new(big.Int).Sub(toBig(a), toBig(b))
	wantBorrow := uint64(0)
	if want.Sign() < 0 {
		wantBorrow = 1
		want.Add(want, two128)
	}
	if toBig(diff).Cmp(want) != 0 || borrow != wantBorrow {
		t.Errorf("SubBorrow(%s, %s) = (%s, %d), want (%s, %d)", a, b, diff, borrow, want, wantBorrow)
	}
}

func checkMul64Carry(t *testing.T, a Uint128, m uint64) {
	t.Helper()
	lo, hi := a.Mul64Carry(m)
	got := new(big.Int).SetUint64(hi)
	got.Lsh(got, 128)
	got.Or(got, toBig(lo))
	want := new(big.Int).Mul(toBig(a), new(big.Int).SetUint64(m))
	if got.Cmp(want) != 0 {
		t.Errorf("Mul64Carry(%s, %d) = %s, want %s", a, m, got, want)
	}
}

func checkQuoRemPow10(t *testing.T, a Uint128, k uint8) {
	t.Helper()
	q, r, s := a.QuoRemPow10(k)
	if k > MaxSafeStrLen64 {
		if s != state.ScaleOutOfRange {
			t.Errorf("QuoRemPow10(%s, %d): state %s, want ScaleOutOfRange", a, k, s)
		}
		return
	}
	if s != state.OK {
		t.Fatalf("QuoRemPow10(%s, %d): unexpected state %s", a, k, s)
	}
	d := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)
	wantQ, wantR := new(big.Int).QuoRem(toBig(a), d, new(big.Int))
	if toBig(q).Cmp(wantQ) != 0 || new(big.Int).SetUint64(r).Cmp(wantR) != 0 {
		t.Errorf("QuoRemPow10(%s, %d) = (%s, %d), want (%s, %s)", a, k, q, r, wantQ, wantR)
	}
}

func TestPrimitivesTable(t *testing.T) {
	maxU := Uint128{Lo: math.MaxUint64, Hi: math.MaxUint64}
	one := Uint128{Lo: 1}
	pairs := []struct{ a, b Uint128 }{
		{Zero, Zero}, {Zero, one}, {one, Zero}, {one, one},
		{maxU, Zero}, {maxU, one}, {one, maxU}, {maxU, maxU},
		{Uint128{Lo: math.MaxUint64}, one},             // carry into Hi
		{Uint128{Hi: 1}, one},                          // borrow from Hi
		{Uint128{Lo: 5, Hi: 7}, Uint128{Lo: 9, Hi: 7}}, // same Hi, Lo borrow
	}
	for _, p := range pairs {
		checkAddCarry(t, p.a, p.b)
		checkSubBorrow(t, p.a, p.b)
		checkSubBorrow(t, p.b, p.a)
	}
	for _, a := range []Uint128{Zero, one, maxU, {Lo: math.MaxUint64}, {Hi: 1}, {Lo: 12345678901234567890, Hi: 3}} {
		for _, m := range []uint64{0, 1, 10, 1e19, math.MaxUint64} {
			checkMul64Carry(t, a, m)
		}
		for k := uint8(0); k <= 21; k++ {
			checkQuoRemPow10(t, a, k)
		}
	}
}

func TestPrimitivesRandom(t *testing.T) {
	r := rand.New(rand.NewSource(20260907))
	for range 20000 {
		a, b := randUint128(r), randUint128(r)
		checkAddCarry(t, a, b)
		checkSubBorrow(t, a, b)
		checkMul64Carry(t, a, r.Uint64())
		checkQuoRemPow10(t, a, uint8(r.Intn(20)))
	}
}

func FuzzAddSubCarry(f *testing.F) {
	f.Add(uint64(0), uint64(0), uint64(0), uint64(0))
	f.Add(uint64(math.MaxUint64), uint64(math.MaxUint64), uint64(1), uint64(0))
	f.Fuzz(func(t *testing.T, aLo, aHi, bLo, bHi uint64) {
		a, b := Uint128{Lo: aLo, Hi: aHi}, Uint128{Lo: bLo, Hi: bHi}
		checkAddCarry(t, a, b)
		checkSubBorrow(t, a, b)
	})
}

func FuzzMul64Carry(f *testing.F) {
	f.Add(uint64(0), uint64(0), uint64(0))
	f.Add(uint64(math.MaxUint64), uint64(math.MaxUint64), uint64(math.MaxUint64))
	f.Fuzz(func(t *testing.T, lo, hi, m uint64) {
		checkMul64Carry(t, Uint128{Lo: lo, Hi: hi}, m)
	})
}

func FuzzQuoRemPow10(f *testing.F) {
	f.Add(uint64(0), uint64(0), uint8(0))
	f.Add(uint64(math.MaxUint64), uint64(math.MaxUint64), uint8(19))
	f.Fuzz(func(t *testing.T, lo, hi uint64, k uint8) {
		checkQuoRemPow10(t, Uint128{Lo: lo, Hi: hi}, k%24)
	})
}

// Regression: Mul folded an overflow out of bit 127 back into the high word instead of reporting it.
// Hi*Lo just below 2^64 plus the carry from the low product triggers it.
func TestMulOverflowCarry(t *testing.T) {
	a := Uint128{Lo: math.MaxUint64, Hi: math.MaxUint64 / 3}
	b := Uint128{Lo: 3}
	if _, s := a.Mul(b); s != state.Overflow {
		t.Errorf("Mul must report overflow, got state %s", s)
	}
	r := rand.New(rand.NewSource(3))
	for range 200000 {
		x, y := randUint128(r), randUint128(r)
		got, s := x.Mul(y)
		want := new(big.Int).Mul(toBig(x), toBig(y))
		if want.Cmp(two128) >= 0 {
			if s != state.Overflow {
				t.Fatalf("Mul(%s, %s) must overflow, got %s", x, y, got)
			}
			continue
		}
		if s != state.OK || toBig(got).Cmp(want) != 0 {
			t.Fatalf("Mul(%s, %s) = %s (%s), want %s", x, y, got, s, want)
		}
	}
}
