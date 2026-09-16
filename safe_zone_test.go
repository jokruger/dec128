package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/uint128"
)

// TestSafeZoneClaim pins the safe operating range the documentation promises: operands with at most 9 decimal places
// and values below 10^20 make Add, Sub and Mul exact, so nothing is ever rounded.
//
// The bound on the scale is the binding one and it is not about magnitude at all: Mul adds the scales, so two
// operands at scale 9 produce a scale-18 product that still fits under MaxScale, while two at scale 10 produce a
// scale-20 product that never does. The bound on the value keeps the product coefficient inside 128 bits:
// 10^20 * 10^20 * 10^18 = 10^58 is too wide, so the product itself is checked rather than the operands.
//
// If either number in the documented rule changes, this test is the thing that must change with it.
func TestSafeZoneClaim(t *testing.T) {
	const (
		maxSafeScale = 9
		valueDigits  = 20 // values stay below 10^20
	)
	restoreConfig(t)
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossRound)

	limit := bigPow10(valueDigits)
	r := rand.New(rand.NewSource(20260911))

	// draw returns a decimal below 10^valueDigits at a scale in 0..maxSafeScale, and its exact value.
	draw := func() (Dec128, *big.Rat) {
		s := uint8(r.Intn(maxSafeScale + 1))
		coef := new(big.Int).Rand(r, new(big.Int).Mul(limit, bigPow10(s)))
		u, st := uint128.FromBigInt(coef)
		if st != 0 {
			t.Fatalf("coefficient %s does not fit: %v", coef, st)
		}
		return Dec128{coef: u, scale: s}, new(big.Rat).SetFrac(coef, bigPow10(s))
	}

	n := 100000
	if testing.Short() {
		n = 2000
	}
	for i := 0; i < n; i++ {
		x, wx := draw()
		y, wy := draw()

		check := func(op string, got Dec128, want *big.Rat) {
			t.Helper()
			if got.IsNaN() {
				t.Fatalf("%s(%s, %s) = NaN(%v), want %s", op, x.StringFixed(), y.StringFixed(), got.ErrorDetails(), want.FloatString(40))
			}
			g, ok := new(big.Rat).SetString(got.StringFixed())
			if !ok || g.Cmp(want) != 0 {
				t.Fatalf("%s(%s, %s) = %s, want exactly %s", op, x.StringFixed(), y.StringFixed(), got.StringFixed(), want.FloatString(40))
			}
		}

		check("Add", x.Add(y), new(big.Rat).Add(wx, wy))
		check("Sub", x.Sub(y), new(big.Rat).Sub(wx, wy))

		// the claim covers products that themselves stay below 10^20
		wp := new(big.Rat).Mul(wx, wy)
		if new(big.Int).Abs(wp.Num()).Cmp(new(big.Int).Mul(limit, wp.Denom())) < 0 {
			check("Mul", x.Mul(y), wp)
		}
	}
}

// TestSafeZoneScaleCliff pins the other half of the rule: one decimal place past the safe scale, multiplication stops
// being exact and small products collapse to zero. Both symptoms are the same cause, a product scale above MaxScale.
func TestSafeZoneScaleCliff(t *testing.T) {
	restoreConfig(t)
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossRound)

	inside := FromString("0.000000001")   // scale 9
	outside := FromString("0.0000000001") // scale 10

	if got := inside.Mul(inside); got.StringFixed() != "0.000000000000000001" {
		t.Errorf("scale 9 squared = %s, want 0.000000000000000001 (exact)", got.StringFixed())
	}
	if got := outside.Mul(outside); !got.IsZero() {
		t.Errorf("scale 10 squared = %s, want 0 (underflow)", got.StringFixed())
	}

	// and the documented envelopes
	if got := MaxAtScale(MaxScale).StringFixed(); got != "34028236692093846346.3374607431768211455" {
		t.Errorf("largest value holding all %d places = %s", MaxScale, got)
	}
	if got := QuantumAtScale(MaxScale).StringFixed(); got != "0.0000000000000000001" {
		t.Errorf("smallest non-zero value = %s", got)
	}
	if d := FromString("0.00000000000000000001"); !d.IsNaN() {
		t.Errorf("1e-20 parsed to %s, want NaN: it is below the smallest representable value", d.StringFixed())
	}
}
