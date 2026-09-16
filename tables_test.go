package dec128

import (
	"math/big"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"
)

// Structural invariants: the hand-written tables, the size of the value type, and the inlining the hot paths depend
// on. None of these is exercised by an ordinary arithmetic test - a wrong table entry or a helper that quietly stops
// inlining changes results or throughput without failing anything - so they are pinned here.

// Compile-time layout guard. Both differences must be valid array lengths, which proves equality rather than an
// upper or lower bound, and does so even when the package is only cross-compiled. Dec128 is a value type that is
// copied by every operation and returned by value, so its width is part of the performance contract: 16 bytes of
// coefficient plus the scale and state bytes, padded to the alignment of the coefficient's limbs.
//
// That alignment is the word, so the expected width is 24 bytes on a 64-bit platform and 20 on a 32-bit one, where a
// uint64 aligns to 4. The shift is the usual way to write the word size as a constant.
const (
	wordIs64   = ^uintptr(0) >> 63 // 1 on a 64-bit platform, 0 on a 32-bit one
	dec128Size = 20 + 4*wordIs64
	dec128Algn = 4 + 4*wordIs64
)

var (
	_ [unsafe.Sizeof(Dec128{}) - dec128Size]byte
	_ [dec128Size - unsafe.Sizeof(Dec128{})]byte
	_ [unsafe.Alignof(Dec128{}) - dec128Algn]byte
	_ [dec128Algn - unsafe.Alignof(Dec128{})]byte
)

// TestTablesAgainstBig recomputes every precalculated table with math/big or from its definition. A typo in one of
// these is invisible to the arithmetic tests until it lands on the one input that uses that entry: bitLen10 drives
// the scale rule's estimate of how many digits must go, and zeroStrs is what StringFixed returns for every zero.
func TestTablesAgainstBig(t *testing.T) {
	pow10 := func(k int) *big.Int {
		return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)
	}

	// bitLen10[k] == bits.Len(10^k), the width used to bound the reduction in fitWide
	if len(bitLen10) != 39 {
		t.Errorf("bitLen10 has %d entries, want 39 (10^0..10^38)", len(bitLen10))
	}
	for k := range len(bitLen10) {
		if want := pow10(k).BitLen(); int(bitLen10[k]) != want {
			t.Errorf("bitLen10[%d] = %d, want %d", k, bitLen10[k], want)
		}
	}

	// the powers of ten, re-exported from package uint128
	if len(Pow10Uint64) != int(MaxScale)+1 {
		t.Errorf("Pow10Uint64 has %d entries, want %d", len(Pow10Uint64), MaxScale+1)
	}
	for k := range len(Pow10Uint64) {
		if want := pow10(k); want.Cmp(new(big.Int).SetUint64(Pow10Uint64[k])) != 0 {
			t.Errorf("Pow10Uint64[%d] = %d, want %s", k, Pow10Uint64[k], want)
		}
	}
	if len(Pow10Uint128) != 39 {
		t.Errorf("Pow10Uint128 has %d entries, want 39", len(Pow10Uint128))
	}
	for k := range len(Pow10Uint128) {
		if want := pow10(k); want.Cmp(Pow10Uint128[k].BigInt()) != 0 {
			t.Errorf("Pow10Uint128[%d] = %s, want %s", k, Pow10Uint128[k], want)
		}
	}
	// 10^38 is the largest power that fits, and 10^39 would not: the table stops in the right place
	if Pow10Uint128[38].BitLen() > 128 {
		t.Error("Pow10Uint128[38] does not fit in 128 bits")
	}
	if pow10(39).BitLen() <= 128 {
		t.Error("10^39 fits in 128 bits, so the table is one entry short")
	}

	// the precalculated zero strings and the zero-padding array
	if len(zeroStrs) != int(MaxScale)+1 {
		t.Errorf("zeroStrs has %d entries, want %d (one per scale)", len(zeroStrs), MaxScale+1)
	}
	for k := range len(zeroStrs) {
		want := "0"
		if k > 0 {
			want = "0." + strings.Repeat("0", k)
		}
		if zeroStrs[k] != want {
			t.Errorf("zeroStrs[%d] = %q, want %q", k, zeroStrs[k], want)
		}
		// and it has to be what StringFixed actually produces for a zero at that scale
		if got := (Dec128{scale: uint8(k)}).StringFixed(); got != want {
			t.Errorf("StringFixed of a zero at scale %d = %q, want %q", k, got, want)
		}
	}
	if len(zeros) < int(MaxScale) {
		t.Errorf("zeros has %d entries, want at least %d", len(zeros), MaxScale)
	}
	for i, c := range zeros {
		if c != '0' {
			t.Errorf("zeros[%d] = %q, want '0'", i, c)
		}
	}

	// the rounding mode names line up with the constants
	if len(roundingModeNames) != int(ROUND_BANK)+1 {
		t.Errorf("roundingModeNames has %d entries, want %d", len(roundingModeNames), ROUND_BANK+1)
	}
	for m := RoundingMode(0); m <= ROUND_BANK; m++ {
		if roundingModeNames[m] == "" {
			t.Errorf("roundingModeNames[%d] is empty", m)
		}
		if m.String() != roundingModeNames[m] {
			t.Errorf("RoundingMode(%d).String() = %q, want %q", m, m.String(), roundingModeNames[m])
		}
	}
}

// inlinedHelpers are the leaf helpers the hot paths call per operation. They carry no loops and no calls, so the
// compiler inlines them today; if one grows past the budget every arithmetic operation starts paying a call for it,
// which no correctness test would notice. The list is deliberately short - it is the ones on the per-operation path,
// not everything that happens to be small.
var inlinedHelpers = []string{
	"signOf",            // every Mul and Div
	"roundDecision",     // every rounding decision
	"compareWidened",    // every unaligned Add, Sub and Compare
	"compare256",        // Sum and the sqrt tie test
	"idealDivScale",     // every Div
	"top64",             // the sqrt seed
	"trimTrailingZeros", // String and the trimmed output forms
	"Dec128.IsZero",
	"Dec128.IsNaN",
	"Dec128.Sign",
	"Dec128.Neg",
	"Dec128.Abs",
}

// TestHotPathHelpersStayInlined re-compiles the package with the compiler's inlining diagnostic and fails if one of
// the pinned leaf helpers is no longer inlinable. It shells out to the toolchain because that decision is not
// observable from inside the test binary.
func TestHotPathHelpersStayInlined(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on PATH")
	}

	// A unique build tag defeats the build cache: a cached compilation replays no diagnostics, so a re-run against
	// unchanged sources would pass vacuously. The tag matches no build constraint here, so the file set is unchanged.
	nonce := "dec128_inlinepin_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	out, err := exec.Command("go", "build", "-tags="+nonce, "-gcflags=-m", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go build -gcflags=-m: %v\n%s", err, out)
	}

	// Diagnostic lines look like "./fit.go:34:6: can inline signOf".
	inlined := map[string]bool{}
	for line := range strings.Lines(string(out)) {
		_, name, ok := strings.Cut(strings.TrimSpace(line), "can inline ")
		if !ok {
			continue
		}
		// generic functions are reported with a shape suffix, e.g. "indexExp[go.shape.string]"
		if i := strings.IndexByte(name, '['); i >= 0 {
			name = name[:i]
		}
		inlined[strings.TrimSpace(name)] = true
	}
	if len(inlined) == 0 {
		t.Fatalf("no inlining diagnostics parsed from:\n%s", out)
	}

	for _, name := range inlinedHelpers {
		if !inlined[name] {
			t.Errorf("%s is no longer inlinable; every operation on its path now pays a call", name)
		}
	}
}
