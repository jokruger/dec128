// Package uint128 provides 128-bit unsigned integer type and basic operations.
package uint128

import (
	"fmt"
	"math/big"

	"github.com/jokruger/dec128/errors"
)

// Uint128 is a 128-bit unsigned integer type.
type Uint128 struct {
	Lo uint64
	Hi uint64
}

// IsZero returns true if the value is zero.
func (self Uint128) IsZero() bool {
	return self.Lo == 0 && self.Hi == 0
}

// Equal returns true if the value is equal to the other value.
func (self Uint128) Equal(other Uint128) bool {
	return self.Lo == other.Lo && self.Hi == other.Hi
}

// Compare returns -1 if the value is less than the other value, 0 if the value is equal to the other value, and 1 if the value is greater than the other value.
func (self Uint128) Compare(other Uint128) int {
	if self == other {
		return 0
	}

	if self.Hi < other.Hi || (self.Hi == other.Hi && self.Lo < other.Lo) {
		return -1
	}

	return 1
}

// BitLen returns the number of bits required to represent the value.
func (self Uint128) BitLen() int {
	return 128 - self.LeadingZeroBitsCount()
}

// Scan scans the value.
func (self *Uint128) Scan(s fmt.ScanState, ch rune) error {
	i := new(big.Int)

	if err := i.Scan(s, ch); err != nil {
		return errors.InvalidFormat.Value()
	}

	if i.Sign() < 0 {
		return errors.Negative.Value()
	}

	if i.BitLen() > 128 {
		return errors.Overflow.Value()
	}

	self.Lo = i.Uint64()
	self.Hi = i.Rsh(i, 64).Uint64()

	return nil
}
