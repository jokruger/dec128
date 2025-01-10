package dec128

import (
	"bytes"
	"database/sql/driver"
	"fmt"

	"github.com/jokruger/dec128/errors"
)

// MarshalText implements the encoding.TextMarshaler interface.
func (self Dec128) MarshalText() ([]byte, error) {
	if self.err != errors.None {
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

	t := FromString(string(data[:]))
	if t.IsNaN() {
		return t.ErrorDetails()
	}
	*self = t

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (self Dec128) MarshalJSON() ([]byte, error) {
	return []byte(`"` + self.String() + `"`), nil
}

var nullValue = []byte("null")

// UnmarshalJSON implements the json.Unmarshaler interface.
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

// Scan implements the sql.Scanner interface.
func (self *Dec128) Scan(src any) error {
	var err error
	switch v := src.(type) {
	case string:
		*self = FromString(v)
		if self.IsNaN() {
			err = self.ErrorDetails()
		}
	case int:
		*self = FromInt(v)
	case int64:
		*self = FromInt64(v)
	case nil:
		*self = Zero
	default:
		err = fmt.Errorf("can't scan %T to Dec128: %T is not supported", src, src)
	}

	return err
}

// Value implements the driver.Valuer interface.
func (self Dec128) Value() (driver.Value, error) {
	return self.String(), nil
}
