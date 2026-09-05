package dec128

import (
	"github.com/jokruger/dec128/state"
)

// SQL NULL and JSON null handling.
//
// By default a NULL scans to Zero, which is what this package has always done. That
// loses the distinction between "no amount recorded" and "the amount is zero", so the
// value produced for NULL is configurable:
//
//	dec128.SetNullValue(dec128.NaN(state.Null))
//
// With that set, a NULL scans to a NaN carrying state.Null, IsNull reports it, and
// Value and MarshalJSON turn it back into NULL and null respectively, so the value
// round-trips. Because it is a NaN it also propagates through arithmetic the way SQL
// NULL does.
//
// This is process-global configuration in the same spirit as SetDefaultScale: set it
// once during initialisation, before any decimal is used.
var nullDec = Zero

// SetNullValue sets the Dec128 that a SQL NULL or a JSON null decodes to.
// The default is Zero. Pass NaN(state.Null) to have NULL round-trip instead.
func SetNullValue(d Dec128) {
	nullDec = d
}

// NullValue returns the Dec128 that a SQL NULL or a JSON null decodes to.
func NullValue() Dec128 {
	return nullDec
}

// Null returns a Dec128 marked as NULL. It is a NaN, so it propagates through
// arithmetic, and Value and MarshalJSON encode it back as NULL and null.
func Null() Dec128 {
	return Dec128{state: state.Null}
}

// IsNull returns true if the Dec128 came from, or was constructed as, a NULL.
// It is false for every other value, including other NaN reasons such as overflow.
func (d Dec128) IsNull() bool {
	return d.state == state.Null
}
