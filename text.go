package dec128

import (
	"github.com/jokruger/dec128/state"
)

// MarshalText implements the encoding.TextMarshaler interface.
func (d Dec128) MarshalText() ([]byte, error) {
	switch {
	case d.state >= state.Error:
		return NaNStrBytes, nil
	case d.IsZero():
		return ZeroStrBytes, nil
	}

	buf := [MaxStrLen]byte{}
	sb, trim := d.appendString(buf[:0])
	if trim {
		return trimTrailingZeros(sb), nil
	}

	return sb, nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (d *Dec128) UnmarshalText(data []byte) error {
	if len(data) == 0 {
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
