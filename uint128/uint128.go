// Package uint128 provides 128-bit unsigned integer type and basic operations.
package uint128

// Uint128 is a 128-bit unsigned integer type.
type Uint128 struct {
	Lo uint64
	Hi uint64
}

// IsZero returns true if the value is zero.
func (ui Uint128) IsZero() bool {
	return ui.Lo == 0 && ui.Hi == 0
}

// Equal returns true if the value is equal to the other value.
func (ui Uint128) Equal(other Uint128) bool {
	return ui.Lo == other.Lo && ui.Hi == other.Hi
}

// Compare returns -1 if the value is less than the other value, 0 if the value is equal to the other value, and 1 if the value is greater than the other value.
func (ui Uint128) Compare(other Uint128) int {
	if ui == other {
		return 0
	}

	if ui.Hi < other.Hi || (ui.Hi == other.Hi && ui.Lo < other.Lo) {
		return -1
	}

	return 1
}

// BitLen returns the number of bits required to represent the value.
func (ui Uint128) BitLen() int {
	return 128 - ui.LeadingZeroBitsCount()
}
