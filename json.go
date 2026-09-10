package dec128

import (
	"bytes"

	"github.com/jokruger/dec128/state"
)

// MarshalJSON implements json.Marshaler: a quoted decimal in the fixed form (see SetTrimOutput), "NaN" for a NaN, and
// null for a NULL value. The returned slice belongs to the caller and shares nothing with the package: the shortcut
// cases copy their constant rather than handing it out, so a caller that writes into the result cannot corrupt it.
func (d Dec128) MarshalJSON() ([]byte, error) {
	switch {
	case d.state == state.Null:
		return bytes.Clone(nullValue), nil
	case d.state >= state.Error:
		return bytes.Clone(NaNJsonStrBytes), nil
	case d.IsZero():
		if trimOutput || d.scale == 0 {
			return bytes.Clone(ZeroJsonStrBytes), nil
		}
	}

	buf := [MaxStrLen + 2]byte{}
	buf[0] = '"'
	sb, trim := d.appendString(buf[:1])
	if trim && trimOutput {
		sb = trimTrailingZeros(sb)
	}
	sb = append(sb, '"')

	// copy into an exactly sized slice: returning a slice of buf would move the whole scratch array to the heap
	out := make([]byte, len(sb))
	copy(out, sb)

	return out, nil
}

var nullValue = []byte("null")

// UnmarshalJSON implements json.Unmarshaler. It accepts the value as a JSON string or as a bare JSON number; null, an
// empty string and the string "null" decode to the NULL value configured with SetNullValue (Zero by default). Only an
// invalid format is an error: "NaN", "Infinity" and "-Infinity" decode to NaN values, and the reason a NaN carried
// before marshaling does not survive the round trip (it comes back as state.NaN).
func (d *Dec128) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}

	if len(data) == 0 || bytes.Equal(data, nullValue) {
		*d = nullDec
		return nil
	}

	t := FromString(data[:])
	if t.state == state.InvalidFormat {
		return t.ErrorDetails()
	}
	*d = t

	return nil
}
