package dec128

import (
	"encoding/json"
	"testing"

	"github.com/jokruger/dec128/state"
)

func TestFixedFormOutput(t *testing.T) {
	defer SetTrimOutput(TrimOutput())
	if TrimOutput() {
		t.Fatal("output must not be trimmed by default")
	}

	cases := []struct{ in, fixed, trimmed string }{
		{"1.50", "1.50", "1.5"},
		{"1.5", "1.5", "1.5"},
		{"100", "100", "100"},
		{"-0.10", "-0.10", "-0.1"},
		{"0.00", "0.00", "0"},
		{"0", "0", "0"},
		{"123456789012345678901234567890.1200", "123456789012345678901234567890.1200", "123456789012345678901234567890.12"},
	}
	for _, c := range cases {
		d := FromString(c.in)
		for _, trim := range []bool{false, true} {
			SetTrimOutput(trim)
			want := c.fixed
			if trim {
				want = c.trimmed
			}
			if v, err := d.Value(); err != nil || v != want {
				t.Errorf("Value(%s) trim=%v = %v, want %s", c.in, trim, v, want)
			}
			if b, err := d.MarshalJSON(); err != nil || string(b) != `"`+want+`"` {
				t.Errorf("MarshalJSON(%s) trim=%v = %s, want %q", c.in, trim, b, want)
			}
			if b, err := d.MarshalText(); err != nil || string(b) != want {
				t.Errorf("MarshalText(%s) trim=%v = %s, want %s", c.in, trim, b, want)
			}
			// String and StringFixed do not depend on the setting.
			if d.String() != c.trimmed || d.StringFixed() != c.fixed {
				t.Errorf("String/StringFixed(%s) changed with the setting", c.in)
			}
		}
	}

	SetTrimOutput(false)
	// NaN and NULL are unaffected.
	if b, _ := NaN(state.Overflow).MarshalJSON(); string(b) != `"NaN"` {
		t.Errorf("NaN JSON = %s", b)
	}
	if b, _ := Null().MarshalJSON(); string(b) != "null" {
		t.Errorf("NULL JSON = %s", b)
	}
	if v, _ := Null().Value(); v != nil {
		t.Errorf("NULL Value = %v", v)
	}

	// The scale survives a round trip through the text paths.
	d := FromString("12.30")
	v, _ := d.Value()
	var back Dec128
	if err := back.Scan(v); err != nil || back != d {
		t.Errorf("Value/Scan round trip: %v -> %v (%v)", v, back, err)
	}
	b, _ := json.Marshal(d)
	if err := json.Unmarshal(b, &back); err != nil || back != d {
		t.Errorf("JSON round trip: %s -> %v (%v)", b, back, err)
	}
	if err := back.UnmarshalText([]byte("0.000")); err != nil || back.Scale() != 3 {
		t.Errorf("text zero keeps its scale: %v (%v)", back, err)
	}
}
