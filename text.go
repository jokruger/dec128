package dec128

import (
	"bytes"

	"github.com/jokruger/dec128/state"
)

// MarshalText implements encoding.TextMarshaler: the decimal in the fixed form (see SetTrimOutput), "NaN" for a NaN,
// and empty text for a NULL-marked value (see SetNullValue), which UnmarshalText maps back so the value round-trips.
// The returned slice belongs to the caller and shares nothing with the package: the shortcut cases copy ZeroStrBytes
// and NaNStrBytes rather than handing them out, so a caller that writes into the result cannot corrupt them.
func (d Dec128) MarshalText() ([]byte, error) {
	switch {
	case d.state == state.Null:
		return []byte{}, nil
	case d.state >= state.Error:
		return bytes.Clone(NaNStrBytes), nil
	case d.IsZero():
		if trimOutput || d.scale == 0 {
			return bytes.Clone(ZeroStrBytes), nil
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

// AppendText implements encoding.TextAppender: it appends the text MarshalText would produce to b and returns the
// extended slice, without allocating when b has room. It never returns an error.
func (d Dec128) AppendText(b []byte) ([]byte, error) {
	if d.state == state.Null {
		return b, nil
	}

	buf := [MaxStrLen]byte{}
	if trimOutput {
		return append(b, d.StringToBuf(buf[:])...), nil
	}

	return append(b, d.StringFixedToBuf(buf[:])...), nil
}

// UnmarshalText implements encoding.TextUnmarshaler. Empty text decodes to the NULL value configured with
// SetNullValue (Zero by default), as an empty or null JSON value does. Only an invalid format is an error; "NaN",
// "Infinity" and "-Infinity" decode to NaN values.
func (d *Dec128) UnmarshalText(data []byte) error {
	if len(data) == 0 {
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
