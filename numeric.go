package dec128

import (
	"github.com/jokruger/dec128/state"
)

// Reporting the shape of a value, for the column it has to be stored in.
//
// A ledger keeps its amounts in SQL NUMERIC(p, s) columns, and a value that needs more than the column holds is
// refused by the database at write time, one row into a batch, with an error that names the column rather than the
// calculation that produced it. These three report the shape in advance, so the check can happen where the value is
// computed.

// SignificantDigits returns the number of decimal digits in the coefficient: 0 for zero, 1 for 0.05 and for 5, and 3
// for 1.50, because a trailing zero is a digit of the representation and a NUMERIC column has to hold it. It is the
// p that the value as written needs in NUMERIC(p, s). NaN yields 0.
func (d Dec128) SignificantDigits() int {
	if d.state >= state.Error {
		return 0
	}
	return d.coef.Len10()
}

// IntegerDigits returns the number of digits before the decimal point, which is 0 for every value below one in
// magnitude: 0.99 needs none, which is why PostgreSQL's NUMERIC(2,2) holds it, 1.5 needs one and -123.45 needs three.
// It is the p - s that the value needs in NUMERIC(p, s). Zero and NaN yield 0.
func (d Dec128) IntegerDigits() int {
	return max(0, d.SignificantDigits()-int(d.scale))
}

// FitsNumeric reports whether d can be stored in a SQL NUMERIC(precision, scale) column exactly, without the database
// rounding it or refusing the row: its integer part needs at most precision-scale digits and its fractional part at
// most scale places.
//
// Trailing zeros are not places the column has to hold, so 1.50 fits NUMERIC(3,1) even though it is written with two
// places. A precision of zero, a scale above the precision, and a NaN are all false. A column wider than a Dec128 is
// allowed and is the easy case: every value fits NUMERIC(45, 19).
//
// Use RescaleRound to make a value fit by rounding it, and RescaleRoundInexact to learn whether that rounding lost
// anything. It reads no process-global configuration.
func (d Dec128) FitsNumeric(precision, scale uint8) bool {
	if d.state >= state.Error || precision == 0 || scale > precision {
		return false
	}
	c := d.Canonical()
	return c.scale <= scale && c.IntegerDigits() <= int(precision)-int(scale)
}
