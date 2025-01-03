package dec128

import (
	"fmt"

	"github.com/jokruger/dec128/uint128"
)

const maxPrecision = uint8(uint128.MaxSafeStrLen64)

var (
	ErrOverflow      = uint128.ErrOverflow
	ErrUnderflow     = uint128.ErrUnderflow
	ErrNegative      = uint128.ErrNegative
	ErrDivByZero     = uint128.ErrDivByZero
	ErrInvalidFormat = uint128.ErrInvalidFormat
	ErrPrecision     = fmt.Errorf("precision out of range")
	ErrNaN           = fmt.Errorf("not a number")

	Zero = Dec128{}
	NaN  = Dec128{nan: true}

	ZeroStr = "0"
	NaNStr  = "NaN"

	defaultPrecision = maxPrecision
)

var (
	pow10 = [...]uint64{
		1,                    // 10^0
		10,                   // 10^1
		100,                  // 10^2
		1000,                 // 10^3
		10000,                // 10^4
		100000,               // 10^5
		1000000,              // 10^6
		10000000,             // 10^7
		100000000,            // 10^8
		1000000000,           // 10^9
		10000000000,          // 10^10
		100000000000,         // 10^11
		1000000000000,        // 10^12
		10000000000000,       // 10^13
		100000000000000,      // 10^14
		1000000000000000,     // 10^15
		10000000000000000,    // 10^16
		100000000000000000,   // 10^17
		1000000000000000000,  // 10^18
		10000000000000000000, // 10^19
	}
)

func SetDefaultPrecision(prec uint8) {
	if prec > maxPrecision {
		panic(ErrPrecision)
	}

	defaultPrecision = prec
}
