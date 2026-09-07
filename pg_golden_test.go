package dec128

import (
	"encoding/csv"
	"os"
	"strings"
	"testing"
)

// TestPostgreSQLGolden checks dec128 against PostgreSQL's numeric, the compliance oracle
// of the design: digit-for-digit on the exact operations, and at PostgreSQL's own scale
// and rounding (half away from zero) on division, rounding and square root.
//
// The golden file is generated once from a real PostgreSQL and
// checked in, so this test has no database dependency. It skips when the file is absent.
func TestPostgreSQLGolden(t *testing.T) {
	f, err := os.Open("testdata/pg_golden.csv")
	if err != nil {
		t.Skip("testdata/pg_golden.csv is not generated: run testdata/pg_golden.sql against PostgreSQL (see testdata/pggolden/main.go)")
	}
	defer func() { _ = f.Close() }()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatal("golden file has no rows")
	}
	col := map[string]int{}
	for i, name := range rows[0] {
		col[name] = i
	}
	get := func(row []string, name string) string {
		i, ok := col[name]
		if !ok {
			t.Fatalf("golden file has no column %q", name)
		}
		return row[i]
	}

	checked := 0
	for _, row := range rows[1:] {
		a, b := FromString(get(row, "a")), FromString(get(row, "b"))
		if a.IsNaN() || b.IsNaN() {
			t.Fatalf("row %s: operands must parse", get(row, "id"))
		}
		id := get(row, "id")

		// Exact operations: identical text in the fixed form, including the scale. A
		// product that needs more than MaxScale places or more than 39 digits is the one
		// case where dec128 reduces the scale (PostgreSQL keeps up to 1000 digits); under
		// the default mode the result is then PostgreSQL's exact result truncated, so
		// the text must be a prefix of PostgreSQL's.
		// PostgreSQL's round(a, n) and trunc(a, n) yield exactly n places, padding a
		// shorter value with zeros, which is RescaleRound's contract.
		for _, c := range []struct {
			name string
			got  Dec128
		}{
			{"add", a.Add(b)}, {"sub", a.Sub(b)}, {"mul", a.Mul(b)},
			{"round2", a.RescaleRound(2, ROUND_HALF_AWAY_FROM_ZERO)}, {"trunc2", a.RescaleRound(2, ROUND_TOWARD_ZERO)},
		} {
			want := get(row, c.name)
			got := c.got.StringFixed()
			if got == want {
				continue
			}
			if c.name == "mul" && strings.HasPrefix(want, got) && c.got.Scale() < scaleOf(want) &&
				(scaleOf(want) > MaxScale || len(strings.ReplaceAll(strings.TrimPrefix(want, "-"), ".", "")) > 39) {
				continue // reduced to fit (MaxScale or 128 bits): a truncated prefix of PostgreSQL's exact product
			}
			t.Errorf("row %s %s(%s, %s) = %s, PostgreSQL says %s", id, c.name, get(row, "a"), get(row, "b"), got, want)
		}

		// Comparison: sign(a - b).
		if want := get(row, "cmp"); strings.TrimSpace(want) != "" {
			got := a.Compare(b)
			if (got < 0 && want != "-1") || (got == 0 && want != "0") || (got > 0 && want != "1") {
				t.Errorf("row %s Compare(%s, %s) = %d, PostgreSQL says %s", id, get(row, "a"), get(row, "b"), got, want)
			}
		}

		// Division and square root: PostgreSQL chooses the result scale (at least 16
		// significant digits) and rounds half away from zero; dec128 reproduces it at
		// that scale with DivRound/SqrtRound. A quotient wider than 128 bits is the
		// one thing it cannot reproduce and is reported as NaN(Overflow).
		want := get(row, "div")
		scale := scaleOf(want)
		if scale <= MaxScale {
			got := a.DivRound(b, scale, ROUND_HALF_AWAY_FROM_ZERO)
			if got.IsNaN() {
				if len(strings.TrimLeft(strings.ReplaceAll(strings.TrimPrefix(want, "-"), ".", ""), "0")) <= 39 {
					t.Errorf("row %s DivRound(%s, %s, %d) = NaN(%v), PostgreSQL says %s", id, get(row, "a"), get(row, "b"), scale, got.ErrorDetails(), want)
				}
			} else if got.StringFixed() != want {
				t.Errorf("row %s DivRound(%s, %s, %d) = %s, PostgreSQL says %s", id, get(row, "a"), get(row, "b"), scale, got.StringFixed(), want)
			}
		}
		if want := get(row, "sqrt"); want != "" {
			scale := scaleOf(want)
			if scale <= MaxScale {
				if got := a.SqrtRound(scale, ROUND_HALF_AWAY_FROM_ZERO); got.StringFixed() != want {
					t.Errorf("row %s SqrtRound(%s, %d) = %s, PostgreSQL says %s", id, get(row, "a"), scale, got.StringFixed(), want)
				}
			}
		}
		checked++
	}
	t.Logf("%d rows checked against PostgreSQL", checked)
}

// scaleOf returns the number of digits after the decimal point in a PostgreSQL text
// rendering; values above MaxScale are returned as is and skipped by the caller.
func scaleOf(s string) uint8 {
	if i := strings.IndexByte(s, '.'); i >= 0 {
		n := len(s) - i - 1
		if n > 255 {
			return 255
		}
		return uint8(n)
	}
	return 0
}
