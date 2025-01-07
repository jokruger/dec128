package dec128

import "bytes"

func (self Dec128) MarshalText() ([]byte, error) {
	return []byte(self.String()), nil
}

func (self *Dec128) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*self = Zero
		return nil
	}

	t := FromString(string(data[:]))
	if t.IsNaN() {
		return t.ErrorDetails()
	}
	*self = t

	return nil
}

func (self Dec128) MarshalJSON() ([]byte, error) {
	return []byte(`"` + self.String() + `"`), nil
}

var nullValue = []byte("null")

func (self *Dec128) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}

	if len(data) == 0 || bytes.Equal(data, nullValue) {
		*self = Zero
		return nil
	}

	t := FromString(string(data[:]))
	if t.IsNaN() {
		return t.ErrorDetails()
	}
	*self = t

	return nil
}
