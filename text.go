package dec128

import (
	"github.com/jokruger/dec128/state"
)

// MarshalText implements the encoding.TextMarshaler interface.
func (self Dec128) MarshalText() ([]byte, error) {
	if self.state >= state.Error {
		return NaNStrBytes, nil
	}

	if self.IsZero() {
		return ZeroStrBytes, nil
	}

	buf := [MaxStrLen]byte{}
	sb, trim := self.appendString(buf[:0])
	if trim {
		return trimTrailingZeros(sb), nil
	}

	return sb, nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (self *Dec128) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*self = Zero
		return nil
	}

	t := FromString(data[:])
	if t.IsNaN() {
		return t.ErrorDetails()
	}
	*self = t

	return nil
}
