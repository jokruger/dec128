package uint128

import (
	"fmt"
	"math/big"

	"github.com/jokruger/dec128/errors"
)

type Uint128 struct {
	Lo uint64
	Hi uint64
}

func (self Uint128) IsZero() bool {
	return self.Lo == 0 && self.Hi == 0
}

func (self Uint128) Equal(other Uint128) bool {
	return self.Lo == other.Lo && self.Hi == other.Hi
}

func (self Uint128) Compare(other Uint128) int {
	if self == other {
		return 0
	}

	if self.Hi < other.Hi || (self.Hi == other.Hi && self.Lo < other.Lo) {
		return -1
	}

	return 1
}

func (self Uint128) BitLen() int {
	return 128 - self.LeadingZeroBitsCount()
}

func (self *Uint128) Scan(s fmt.ScanState, ch rune) errors.Error {
	i := new(big.Int)

	if err := i.Scan(s, ch); err != nil {
		return errors.InvalidFormat
	}

	if i.Sign() < 0 {
		return errors.Negative
	}

	if i.BitLen() > 128 {
		return errors.Overflow
	}

	self.Lo = i.Uint64()
	self.Hi = i.Rsh(i, 64).Uint64()

	return errors.None
}
