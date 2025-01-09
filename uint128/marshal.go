package uint128

import "fmt"

// MarshalText implements the encoding.TextMarshaler interface.
func (self Uint128) MarshalText() ([]byte, error) {
	return []byte(self.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (self *Uint128) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		*self = Zero
		return nil
	}

	_, err := fmt.Sscan(string(b), self)
	return err
}
