package uint128

import (
	"math/big"
	"testing"
)

// TestFromBigIntDoesNotMutate pins that the conversion reads its argument and leaves it alone. It used to shift the
// caller's value right by 64 bits in place, silently corrupting any *big.Int that was still in use.
func TestFromBigIntDoesNotMutate(t *testing.T) {
	for _, s := range []string{"0", "1", "18446744073709551616", "342619950006507814151882", "340282366920938463463374607431768211455"} {
		i, ok := new(big.Int).SetString(s, 10)
		if !ok {
			t.Fatalf("bad literal %q", s)
		}
		want := new(big.Int).Set(i)
		u, st := FromBigInt(i)
		if st != 0 {
			t.Fatalf("FromBigInt(%s) state = %v", s, st)
		}
		if i.Cmp(want) != 0 {
			t.Errorf("FromBigInt mutated its argument: %s became %s", want, i)
		}
		if got := u.String(); got != s {
			t.Errorf("FromBigInt(%s) = %s", s, got)
		}
	}
}
