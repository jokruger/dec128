package dec128

import (
	"bytes"
	"database/sql/driver"
	"fmt"
)

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

func (self *Dec128) Scan(src any) error {
	var err error
	switch v := src.(type) {
	case string:
		*self = FromString(v)
		if self.IsNaN() {
			err = self.ErrorDetails()
		}
	//case int64:
	//	*d, err = NewFromInt64(v, 0)
	//case int:
	//	*d, err = NewFromInt64(int64(v), 0)
	//case int32:
	//	*d, err = NewFromInt64(int64(v), 0)
	//case float64:
	//	*d, err = NewFromFloat64(v)
	case nil:
		*self = Zero
	default:
		err = fmt.Errorf("can't scan %T to Dec128: %T is not supported", src, src)
	}

	return err
}

func (self Dec128) Value() (driver.Value, error) {
	return self.String(), nil
}
