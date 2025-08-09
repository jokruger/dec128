package uint128

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

	r, st := FromString(string(b))
	if st.IsError() {
		return st.Error()
	}

	*self = r
	return nil
}
