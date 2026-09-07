package dec128

import (
	"github.com/jokruger/dec128/state"
)

// MarshalText implements encoding.TextMarshaler: the decimal in the fixed form (see SetTrimOutput), or "NaN".
func (d Dec128) MarshalText() ([]byte, error) {
	switch {
	case d.state >= state.Error:
		return NaNStrBytes, nil
	case d.IsZero():
		if trimOutput || d.scale == 0 {
			return ZeroStrBytes, nil
		}
	}

	buf := [MaxStrLen]byte{}
	sb, trim := d.appendString(buf[:0])
	if trim && trimOutput {
		sb = trimTrailingZeros(sb)
	}

	// copy into an exactly sized slice: returning a slice of buf would move the whole scratch array to the heap
	out := make([]byte, len(sb))
	copy(out, sb)

	return out, nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (d *Dec128) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*d = Zero
		return nil
	}

	t := FromString(data[:])
	if t.state == state.InvalidFormat {
		return t.ErrorDetails()
	}
	*d = t

	return nil
}
