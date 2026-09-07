package dec128

import (
	"testing"

	"github.com/jokruger/dec128/state"
)

// The PostgreSQL spellings of its special numeric values parse; nothing else does.
func TestFromStringSpecials(t *testing.T) {
	cases := []struct {
		in string
		st state.State
	}{
		{"NaN", state.NaN},
		{"Infinity", state.Overflow},
		{"-Infinity", state.Overflow},
		{"+Infinity", state.InvalidFormat},
		{"nan", state.InvalidFormat},
		{"NAN", state.InvalidFormat},
		{"inf", state.InvalidFormat},
		{"Inf", state.InvalidFormat},
		{"-NaN", state.InvalidFormat},
		{"NaNx", state.InvalidFormat},
		{"Infinity ", state.InvalidFormat},
		{" NaN", state.InvalidFormat},
		{"1.2.3", state.InvalidFormat},
		{"abc", state.InvalidFormat},
	}
	for _, c := range cases {
		d := FromString(c.in)
		if d.state != c.st {
			t.Errorf("FromString(%q): state %s, want %s", c.in, d.state, c.st)
		}
		if b := FromString([]byte(c.in)); b.state != c.st {
			t.Errorf("FromString([]byte(%q)): state %s, want %s", c.in, b.state, c.st)
		}
	}
	if err := FromString("NaN").ErrorDetails(); err == nil || err.Error() != "not a number" {
		t.Errorf("ErrorDetails of a parsed NaN = %v", err)
	}
	if !FromString("NaN").IsNaN() || FromString("NaN").IsNull() {
		t.Error("a parsed NaN is NaN and not NULL")
	}
}

func TestScanSpecials(t *testing.T) {
	var d Dec128
	for _, in := range []any{"NaN", []byte("NaN")} {
		if err := d.Scan(in); err != nil || d.state != state.NaN {
			t.Errorf("Scan(%v): err %v, state %s", in, err, d.state)
		}
	}
	if err := d.Scan("Infinity"); err != nil || d.state != state.Overflow {
		t.Errorf("Scan(Infinity): err %v, state %s", err, d.state)
	}
	if err := d.Scan("-Infinity"); err != nil || d.state != state.Overflow {
		t.Errorf("Scan(-Infinity): err %v, state %s", err, d.state)
	}
	// A numeric the type cannot hold scans to a NaN with the reason, not to an error.
	if err := d.Scan("1.12345678901234567890123"); err != nil || d.state != state.ScaleOutOfRange {
		t.Errorf("Scan(too many decimals): err %v, state %s", err, d.state)
	}
	if err := d.Scan("1e50"); err != nil || d.state != state.Overflow {
		t.Errorf("Scan(1e50): err %v, state %s", err, d.state)
	}
	// Input that is not a number at all is still an error.
	for _, in := range []any{"abc", "1.2.3", []byte("nan"), ""} {
		if err := d.Scan(in); (err == nil) != (in == "") {
			t.Errorf("Scan(%v): err %v", in, err)
		}
	}
	// Round trip: a NaN written with Value scans back as a NaN.
	v, err := NaN(state.Overflow).Value()
	if err != nil || v != "NaN" {
		t.Fatalf("Value of NaN = %v, %v", v, err)
	}
	if err := d.Scan(v); err != nil || !d.IsNaN() {
		t.Errorf("round trip of NaN through Value/Scan: err %v, %v", err, d)
	}
}

func TestUnmarshalSpecials(t *testing.T) {
	var d Dec128
	if err := d.UnmarshalJSON([]byte(`"NaN"`)); err != nil || d.state != state.NaN {
		t.Errorf("UnmarshalJSON(\"NaN\"): err %v, state %s", err, d.state)
	}
	if err := d.UnmarshalText([]byte("NaN")); err != nil || d.state != state.NaN {
		t.Errorf("UnmarshalText(NaN): err %v, state %s", err, d.state)
	}
	if err := d.UnmarshalJSON([]byte(`"abc"`)); err == nil {
		t.Error("UnmarshalJSON of a non-number must fail")
	}
	if err := d.UnmarshalText([]byte("abc")); err == nil {
		t.Error("UnmarshalText of a non-number must fail")
	}
	// MarshalJSON and UnmarshalJSON round-trip a NaN.
	b, _ := NaN(state.DivisionByZero).MarshalJSON()
	if err := d.UnmarshalJSON(b); err != nil || !d.IsNaN() {
		t.Errorf("JSON round trip of NaN: %s -> err %v, %v", b, err, d)
	}
}
