package uint128

import (
	"math"
	"math/big"
	"math/bits"
	"math/rand"
	"testing"
)

// TestPow10DivTable recomputes every constant from its definition with math/big.
func TestPow10DivTable(t *testing.T) {
	one := big.NewInt(1)
	two64 := new(big.Int).Lsh(one, 64)
	two128m1 := new(big.Int).Sub(new(big.Int).Lsh(one, 128), one)
	for k := 1; k <= 19; k++ {
		e := pow10DivTab[k]
		d := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)
		d5 := new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(k)), nil)
		p := 0
		for new(big.Int).Lsh(one, uint(p+k)).Cmp(d5) < 0 {
			p++
		}
		m := new(big.Int).Lsh(one, uint(64+p))
		m.Add(m, new(big.Int).Sub(d5, one)).Div(m, d5)
		s := bits.LeadingZeros64(d.Uint64())
		dn := d.Uint64() << s
		v := new(big.Int).Div(two128m1, new(big.Int).SetUint64(dn))
		v.Sub(v, two64)
		if e.d != d.Uint64() || e.m != m.Uint64() || e.p != uint8(p) || e.s != uint8(s) || e.dn != dn || e.v != v.Uint64() {
			t.Errorf("entry %d: %+v, want d=%d m=%#x p=%d s=%d dn=%#x v=%#x", k, e, d, m, p, s, dn, v)
		}
		if e.d != Pow10Uint64[k] {
			t.Errorf("entry %d disagrees with Pow10Uint64", k)
		}
	}
}

func checkPow10Div(t *testing.T, u Uint128, k uint8) {
	t.Helper()
	q, r, s := u.QuoRemPow10(k)
	if s.IsError() {
		t.Fatalf("QuoRemPow10(%s, %d): %s", u, k, s)
	}
	d := Pow10Uint64[k]
	var wq Uint128
	var wr uint64
	if u.Hi < d {
		wq.Lo, wr = bits.Div64(u.Hi, u.Lo, d)
	} else {
		wq.Hi, wr = bits.Div64(0, u.Hi, d)
		wq.Lo, wr = bits.Div64(wr, u.Lo, d)
	}
	if q != wq || r != wr {
		t.Fatalf("QuoRemPow10(%s, %d) = (%s, %d), want (%s, %d)", u, k, q, r, wq, wr)
	}
}

func TestPow10DivAgainstHardware(t *testing.T) {
	// boundaries: 0, 1, powers of two, 2^64 +- 1, multiples of 10^k +- 1, max
	var vals []Uint128
	for i := 0; i < 128; i++ {
		v := One.Lsh(uint(i))
		vals = append(vals, v, add1(v), sub1(v))
	}
	vals = append(vals, Zero, One, Uint128{Lo: math.MaxUint64}, Uint128{Lo: math.MaxUint64, Hi: math.MaxUint64}, Uint128{Hi: 1})
	for k := uint8(0); k <= 38; k++ {
		vals = append(vals, Pow10Uint128[k], add1(Pow10Uint128[k]))
		if !Pow10Uint128[k].IsZero() {
			vals = append(vals, sub1(Pow10Uint128[k]))
		}
	}
	for _, u := range vals {
		for k := uint8(0); k <= 19; k++ {
			checkPow10Div(t, u, k)
		}
	}
	r := rand.New(rand.NewSource(20260917))
	for range 300000 {
		checkPow10Div(t, randUint128(r), uint8(r.Intn(20)))
	}
}

func TestQuoRem256ByPow10(t *testing.T) {
	r := rand.New(rand.NewSource(20260918))
	for range 200000 {
		lo, hi := randUint128(r), randUint128(r)
		switch r.Intn(3) {
		case 0:
			hi = Uint128{}
		case 1:
			hi = Uint128{Lo: r.Uint64() % 100000}
		}
		k := uint8(1 + r.Intn(19))
		q, rem, ok := QuoRem256ByPow10(lo, hi, k)
		n := u256ToBig(lo, hi)
		d := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)
		wq, wr := new(big.Int).QuoRem(n, d, new(big.Int))
		if wq.Cmp(two128) >= 0 {
			if ok {
				t.Fatalf("QuoRem256ByPow10(%s, %s, %d) must not fit", lo, hi, k)
			}
			continue
		}
		if !ok || toBig(q).Cmp(wq) != 0 || new(big.Int).SetUint64(rem).Cmp(wr) != 0 {
			t.Fatalf("QuoRem256ByPow10(%s, %s, %d) = (%s, %d, %v), want (%s, %s)", lo, hi, k, q, rem, ok, wq, wr)
		}
	}
}

func FuzzQuoRemPow10Reciprocal(f *testing.F) {
	f.Add(uint64(0), uint64(0), uint8(1))
	f.Add(uint64(math.MaxUint64), uint64(math.MaxUint64), uint8(19))
	f.Fuzz(func(t *testing.T, lo, hi uint64, k uint8) {
		checkPow10Div(t, Uint128{Lo: lo, Hi: hi}, k%20)
	})
}

func add1(v Uint128) Uint128 { r, _ := v.Add64(1); return r }
func sub1(v Uint128) Uint128 { r, _ := v.Sub64(1); return r }
