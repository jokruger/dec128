package dec128

// Text output policy for the encoders: Value (database/sql), MarshalJSON and MarshalText.
//
// By default they emit the fixed form - the value's own scale, trailing zeros kept - so that 1.50 reaches a database
// or a JSON consumer as "1.50" and its scale survives the round trip, as it does in PostgreSQL and .NET.
// SetTrimOutput(true) switches them to the trimmed form ("1.5"), which is what versions up to v1.0.20 emitted.
// String always trims and StringFixed never does, whatever this setting.
//
// This is process-global configuration in the same spirit as SetDefaultScale: set it once during initialization,
// before any decimal is encoded.
var trimOutput = false

// SetTrimOutput selects whether Value, MarshalJSON and MarshalText remove trailing zeros from the fraction. The default
// is false: the scale is preserved. Pass true to restore the output of versions up to v1.0.20.
func SetTrimOutput(trim bool) {
	trimOutput = trim
}

// TrimOutput reports whether Value, MarshalJSON and MarshalText remove trailing zeros.
func TrimOutput() bool {
	return trimOutput
}
