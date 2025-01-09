package uint128

import (
	"math"
)

const (
	// MaxUint64Str is the string representation of the maximum uint64 value.
	MaxUint64Str = "18446744073709551615"

	// MaxUint128Str is the string representation of the maximum uint128 value.
	MaxUint128Str = "340282366920938463463374607431768211455"

	// MaxStrLen is the maximum number of digits in a 128-bit unsigned integer.
	MaxStrLen = 39

	// MaxStrLen64 is the maximum number of digits in a 64-bit unsigned integer.
	MaxStrLen64 = 20

	// MaxSafeStrLen64 is the maximum number of digits that can be safely parsed as a 64-bit unsigned integer.
	MaxSafeStrLen64 = 19

	// ZeroStr is the string representation of the zero value.
	ZeroStr = "0"
)

var (
	// Zero is the zero Uint128 value.
	Zero = Uint128{}

	// One is the Uint128 value of 1.
	One = Uint128{Lo: 1}

	// Max is the maximum Uint128 value.
	Max = Uint128{math.MaxUint64, math.MaxUint64}

	// Max64 is the maximum Uint128 value that fits in a 64-bit unsigned integer.
	Max64 = Uint128{math.MaxUint64, 0}

	// Pow10Uint64 is an array of precalculated powers of 10 for uint64 values (from 10^0 to 10^19).
	Pow10Uint64 = [...]uint64{
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

	// Pow10Uint128 is an array of precalculated powers of 10 for Uint128 values (from 10^0 to 10^38).
	Pow10Uint128 = [...]Uint128{
		{Lo: 1},                                           // 10^0
		{Lo: 10},                                          // 10^1
		{Lo: 100},                                         // 10^2
		{Lo: 1000},                                        // 10^3
		{Lo: 10000},                                       // 10^4
		{Lo: 100000},                                      // 10^5
		{Lo: 1000000},                                     // 10^6
		{Lo: 10000000},                                    // 10^7
		{Lo: 100000000},                                   // 10^8
		{Lo: 1000000000},                                  // 10^9
		{Lo: 10000000000},                                 // 10^10
		{Lo: 100000000000},                                // 10^11
		{Lo: 1000000000000},                               // 10^12
		{Lo: 10000000000000},                              // 10^13
		{Lo: 100000000000000},                             // 10^14
		{Lo: 1000000000000000},                            // 10^15
		{Lo: 10000000000000000},                           // 10^16
		{Lo: 100000000000000000},                          // 10^17
		{Lo: 1000000000000000000},                         // 10^18
		{Lo: 10000000000000000000},                        // 10^19
		{Lo: 7766279631452241920, Hi: 5},                  // 10^20
		{Lo: 3875820019684212736, Hi: 54},                 // 10^21
		{Lo: 1864712049423024128, Hi: 542},                // 10^22
		{Lo: 200376420520689664, Hi: 5421},                // 10^23
		{Lo: 2003764205206896640, Hi: 54210},              // 10^24
		{Lo: 1590897978359414784, Hi: 542101},             // 10^25
		{Lo: 15908979783594147840, Hi: 5421010},           // 10^26
		{Lo: 11515845246265065472, Hi: 54210108},          // 10^27
		{Lo: 4477988020393345024, Hi: 542101086},          // 10^28
		{Lo: 7886392056514347008, Hi: 5421010862},         // 10^29
		{Lo: 5076944270305263616, Hi: 54210108624},        // 10^30
		{Lo: 13875954555633532928, Hi: 542101086242},      // 10^31
		{Lo: 9632337040368467968, Hi: 5421010862427},      // 10^32
		{Lo: 4089650035136921600, Hi: 54210108624275},     // 10^33
		{Lo: 4003012203950112768, Hi: 542101086242752},    // 10^34
		{Lo: 3136633892082024448, Hi: 5421010862427522},   // 10^35
		{Lo: 12919594847110692864, Hi: 54210108624275221}, // 10^36
		{Lo: 68739955140067328, Hi: 542101086242752217},   // 10^37
		{Lo: 687399551400673280, Hi: 5421010862427522170}, // 10^38
	}
)
