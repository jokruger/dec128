package dec128

import "testing"

// Regression: Compare returned the magnitude ordering, not the value ordering, when a
// negative operand could not be rescaled to the common scale (rescale overflow).
// -3e38 < -1.5, but the pre-fix code reported the opposite.
func TestCompareNegativeRescaleOverflow(t *testing.T) {
	big := FromString("-300000000000000000000000000000000000000") // -3e38, scale 0
	small := FromString("-1.5")
	if big.IsNaN() || small.IsNaN() {
		t.Fatal("test operands must be valid")
	}

	cases := []struct {
		name string
		a, b Dec128
		want int
	}{
		{"neg big vs neg small", big, small, -1},
		{"neg small vs neg big", small, big, 1},
		{"pos big vs pos small", big.Abs(), small.Abs(), 1},
		{"pos small vs pos big", small.Abs(), big.Abs(), -1},
		{"neg big vs pos small", big, small.Abs(), -1},
		{"pos big vs neg small", big.Abs(), small, 1},
	}
	for _, c := range cases {
		if got := c.a.Compare(c.b); got != c.want {
			t.Errorf("%s: Compare = %d, want %d", c.name, got, c.want)
		}
	}

	if !big.LessThan(small) {
		t.Error("-3e38 must be less than -1.5")
	}
	if big.Equal(small) || small.Equal(big) {
		t.Error("values of different magnitude must not be equal")
	}
	if Min(big, small).Compare(big) != 0 {
		t.Error("Min must pick -3e38")
	}
	if Max(big, small).Compare(small) != 0 {
		t.Error("Max must pick -1.5")
	}
}
