package uint128

// MarshalText implements the encoding.TextMarshaler interface.
func (ui Uint128) MarshalText() ([]byte, error) {
	return []byte(ui.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (ui *Uint128) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		*ui = Zero
		return nil
	}

	r, st := FromString(string(b))
	if st.IsError() {
		return st.Error()
	}

	*ui = r
	return nil
}
