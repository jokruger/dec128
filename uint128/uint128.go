// Package uint128 provides 128-bit unsigned integer type and basic operations.
package uint128

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
