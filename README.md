# dec128
128-bit fixed-point decimal numbers in go.

## Key Objectives
- [ ] High performance
- [ ] High precision in financial calculations
- [x] No panic or error arithmetics (use NaN instead)
- [ ] Basic arithmetic operations required for financial calculations (specifically for banking and accounting)
- [x] Easy to use
- [ ] Easy to inegrate with external systems (e.g. databases, accounting systems, JSON, etc.)
- [ ] Financially correct rounding
- [x] Correct comparison of numbers encoded in different precisions (e.g. 1.0 == 1.00)
- [x] Correct handling of NaN values (e.g. NaN + 1 = NaN)
- [x] Conversion to canonical representation (e.g. 1.0000 -> 1)
- [ ] Conversion to fixed string representation (e.g. 1.0000 -> "1.0000")
- [x] Conversion to human-readable string representation (e.g. 1.0000 -> "1")

## Notes on Terminology
- **Precision**: The number of decimal places in a number. For example, 1.00 has a precision of 2 and 1.0000 has a precision of 4.
- **Expontent**: Same as precision, but in the context of low-level implementation details or Dec128 encoding.
- **Canonical**: The representation of a number with the minimum number of decimal places required to represent the number.
