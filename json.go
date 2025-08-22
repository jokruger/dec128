package dec128

import (
	"bytes"

	"github.com/jokruger/dec128/state"
)

// MarshalJSON implements the json.Marshaler interface.
func (d Dec128) MarshalJSON() ([]byte, error) {
	switch {
	case d.state >= state.Error:
		return NaNJsonStrBytes, nil
	case d.IsZero():
		return ZeroJsonStrBytes, nil
	}

	buf := [MaxStrLen + 2]byte{}
	buf[0] = '"'
	sb, trim := d.appendString(buf[:1])
	if trim {
		sb = trimTrailingZeros(sb)
	}
	return append(sb, '"'), nil
}

var nullValue = []byte("null")

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Dec128) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}

	if len(data) == 0 || bytes.Equal(data, nullValue) {
		*d = Zero
		return nil
	}

	t := FromString(data[:])
	if t.IsNaN() {
		return t.ErrorDetails()
	}
	*d = t

	return nil
}
