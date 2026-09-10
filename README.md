# dec128

[![Go Reference](https://pkg.go.dev/badge/github.com/jokruger/dec128.svg)](https://pkg.go.dev/github.com/jokruger/dec128)
[![lint](https://github.com/jokruger/dec128/actions/workflows/lint.yml/badge.svg)](https://github.com/jokruger/dec128/actions/workflows/lint.yml)
[![codecov](https://codecov.io/gh/jokruger/dec128/graph/badge.svg?token=TQWE8PA4AN)](https://codecov.io/gh/jokruger/dec128)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

High performance 128-bit fixed-point decimal numbers in Go, built for financial and banking arithmetic.

`dec128` is named for its coefficient: a 128-bit integer with a decimal scale, the same fixed-point model as
Apache Arrow's and Parquet's `Decimal128`. It is not IEEE 754 `decimal128`, which is a floating-point format; that is
supported as an interchange encoding only. The arithmetic follows SQL `NUMERIC` as implemented by PostgreSQL, and
the test suite checks it against PostgreSQL digit for digit.

## Key Objectives / Features

- [x] High performance
- [x] Zero dependencies
- [x] Minimal or zero memory allocation
- [x] Scale up to 19 decimal places
- [x] Fixed 24-byte layout with no indirection (128-bit coefficient, scale, sign/state)
- [x] No panic or error arithmetics (use NaN instead)
- [x] Immutability (methods return new instances)
- [x] Basic arithmetic operations required for financial calculations (specifically for banking and accounting)
- [x] Easy to use
- [x] Easy to integrate with external systems (e.g. databases, accounting systems, JSON, etc.)
- [x] Financially correct rounding: eight rounding modes, chosen per call or globally
- [x] Results that do not fit are rounded to fit, never silently wrong; NaN only when the integer part cannot be held
- [x] Correct comparison of numbers encoded in different scales (e.g. 1.0 == 1.00)
- [x] Correct handling of NaN values (e.g. NaN + 1 = NaN)
- [x] Conversion to canonical representation (e.g. 1.0000 -> 1)
- [x] Conversion to fixed string representation (e.g. 1.0000 -> "1.0000")
- [x] Conversion to human-readable string representation (e.g. 1.0000 -> "1")
- [x] Scientific notation: parsed by `FromString`, printed on request (e.g. "1.5e3" -> 1500, 12345 -> "1.2345e+4")
- [x] Per-call scale and rounding mode for multiplication, division and square root (`MulRound`, `DivRound`, `SqrtRound`)
- [x] Interchange codecs: PostgreSQL `numeric` binary, IEEE 754 decimal128 (BID), Arrow/Parquet int128
- [x] Configurable SQL `NULL` / JSON `null` handling that round-trips (`SetNullValue`)

## Install

Run `go get github.com/jokruger/dec128`

## Requirements

This library requires Go version `>=1.24` (as declared in `go.mod`).

## Development

```bash
make test       # run tests with coverage
make view       # open the HTML coverage report
make lint       # run golangci-lint (installs the pinned version on first use)
make lint-fix   # run golangci-lint with --fix
```

Linting is configured in `.golangci.yml` and enforced in CI by `.github/workflows/lint.yml`.

## Documentation

https://pkg.go.dev/github.com/jokruger/dec128

## Usage

```go
package main

import (
    "fmt"
    "github.com/jokruger/dec128"
)

func main() {
    principal := dec128.FromString("1000.00")
    annualRate := dec128.FromString("5.0")
    days := 30

    // multiply first and divide once at the end: dividing early throws away digits
    // that the later multiplications would have needed
    accrued := principal.Mul(annualRate).MulInt(days).Div(dec128.FromInt64(36500))

    if err := accrued.ErrorDetails(); err != nil {
        panic(err) // a chain reports failure once, at the end
    }

    accrued = accrued.RoundBank(2)

    fmt.Printf("Principal: %v\n", principal.StringFixed())
    fmt.Printf("Annual Interest Rate: %v\n", annualRate.String())
    fmt.Printf("Days: %v\n", days)
    fmt.Printf("Accrued Interest: %v\n", accrued.String())

    total := principal.Add(accrued)
    fmt.Printf("Total after %v days: %v\n", days, total.StringFixed())
}
```

## The contract

**Arithmetic never panics and never returns an error.** A failed operation returns a NaN carrying the reason, and NaN
propagates, so a whole calculation is one expression checked once at the end:

```go
total := principal.Mul(rate).Add(fee).RoundBank(2)
if err := total.ErrorDetails(); err != nil {
    return err
}
```

`ErrorDetails` returns a plain `error` and is the terminal check for a chain; `IsNaN` is the boolean form. Nothing
forces you to call either — unlike an ignored `err` there is no compiler or linter backstop, so make the check part of
your review habits.

Four things that surprise people:

- **`IsZero`, `IsNegative` and `IsPositive` all return `false` for NaN.** A guard written as `if !d.IsZero()` does *not*
  catch a failed calculation. Test `IsNaN` first when the difference matters.
- **`MarshalJSON` and `Value` both encode NaN as the string `"NaN"`.** A JSON consumer that accepts strings takes it
  without complaint, and PostgreSQL accepts `'NaN'` for a `numeric` column. Check `IsNaN` before a failed calculation
  reaches storage.
- **A result that does not fit is rounded, not NaN.** The scale is reduced to the largest at which the integer part fits
  and the dropped digits are rounded with the configured mode (truncation by default). NaN means the integer part itself
  does not fit. `SetArithmeticRounding(ROUND_NAN)` turns any loss of digits back into a NaN.
- **`SetDefaultScale`, `SetArithmeticRounding`, `SetTrimOutput` and `SetNullValue` are process-global.** Set them once
  during initialization; changing them while other goroutines calculate is a data race.

## Scale and rounding

Scale is preserved rather than normalized, so `1.5` and `1.50` are distinct representations that compare equal.
`Add`/`Sub` take the larger operand scale, `Mul` adds them, and a zero result keeps that scale too
(`1.50 - 1.50` is `0.00`). When the exact result needs more than 19 places or more than 128 bits, the scale is reduced
to the largest that fits and the digits below it are rounded with the mode set by `SetArithmeticRounding`
(`ROUND_TOWARD_ZERO` by default).

`Div` and `Sqrt` compute at the default scale (`SetDefaultScale`, 19 by default) and give an exact result its *ideal*
scale: `1/2` is `0.5`, `1.00/2` is `0.50`, `1/3` keeps all 19 places. `QuoRem` does not consult the default: the
quotient is always an integer.

`MulRound`, `DivRound` and `SqrtRound` take the scale and the rounding mode per call and return exactly that scale,
computed once with an exact rounding decision:

```go
rate := annual.DivRound(dec128.FromInt64(365), 12, dec128.ROUND_HALF_AWAY_FROM_ZERO) // 0.013698630137
fee := amount.MulRound(rate, 2, dec128.ROUND_BANK)
```

The `Round*` methods round to at most the given number of places and leave a shorter value alone;
`RescaleRound(places, mode)` is the "exactly n places" operation, the equivalent of PostgreSQL's `round(x, n)`
and `trunc(x, n)`:

```go
dec128.FromString("1.5").RoundBank(2)                            // 1.5
dec128.FromString("1.5").RescaleRound(2, dec128.ROUND_BANK)      // 1.50
dec128.FromString("200").DivRound(dec128.FromInt64(3), 2, dec128.ROUND_HALF_AWAY_FROM_ZERO) // 66.67
```

## Text output

`String` removes trailing zeros; `StringFixed` keeps the value's scale. `Value` (for `database/sql`), `MarshalJSON`
and `MarshalText` emit the fixed form, so `1.50` reaches PostgreSQL and JSON consumers as `1.50` and its scale survives
the round trip. `SetTrimOutput(true)` restores the trimmed output of versions up to v1.0.20.

## Interchange formats

Besides its own compact binary form (`EncodeBinary`, at most 18 bytes), `dec128` reads and writes three external
formats, all into caller-supplied buffers without allocating:

- **PostgreSQL `numeric` binary** (`EncodePgNumeric`, `DecodePgNumeric`): the wire format of the binary protocol, dscale
  preserved. With `pgx` this allows a custom codec for OID 1700 that skips the `big.Int` and text round trip pgx
  performs for a `sql.Scanner` target.
- **IEEE 754 decimal128, BID encoding** (`EncodeIEEE`, `DecodeIEEE`): 16 bytes, as used by MongoDB's BSON Decimal128.
  A coefficient wider than 34 digits is rounded on export with the configured mode (`ROUND_NAN` refuses); `FitsIEEE`
  tells in advance.
- **int128 with a schema scale** (`EncodeInt128`, `DecodeInt128`): Apache Arrow (little-endian) and
  Parquet (big-endian) `Decimal128`.

A value the type cannot hold decodes to a NaN carrying the reason rather than being rounded on the way in.

## SQL NULL

By default a `NULL` scans to `Zero`, which loses the difference between "no amount recorded" and "the amount is zero".
`SetNullValue` makes it round-trip instead:

```go
dec128.SetNullValue(dec128.Null()) // once, during initialization

var amount dec128.Dec128
_ = rows.Scan(&amount)
amount.IsNull()          // true for a NULL column
amount.Add(fee).IsNull() // true - NULL propagates the way SQL NULL does
v, _ := amount.Value()   // nil, so it writes back as NULL
```

A NULL-marked value is a NaN, so it propagates through arithmetic, but `IsNull` distinguishes it from an overflow or a
parse failure. `MarshalJSON` emits `null` for it and `UnmarshalJSON` accepts `null` symmetrically; `MarshalText` emits
empty text and `UnmarshalText` maps empty text back the same way. The default stays `Zero`, so nothing changes unless
you opt in.

## Scientific notation

`FromString` accepts both forms, so JSON, text and `sql.Scanner` input needs no extra handling:

```go
dec128.FromString("1.5e3")   // 1500
dec128.FromString("-2.5E-2") // -0.025
```

`FromSafeString` is regular form only. It skips all format checks by design, so there is nothing for the exponent marker
to be caught by, and adding a check would cost a comparison per character on the fastest parsing path. Send input that
may carry an exponent through `FromString`, which detects the marker inside the validation it already performs and so
costs nothing extra.

Printing stays in the regular form unless the scientific one is asked for. `String`, `MarshalJSON`, `MarshalText`
and `Value` are unchanged:

```go
d := dec128.FromString("12345")
d.String()    // "12345"
d.StringSci() // "1.2345e+4"
```

`StringSci` normalizes to a single leading digit, always signs the exponent and drops the trailing zeros of the
mantissa; zero prints as `0e+0` and NaN as `NaN`. Use `StringSciToBuf` with a `[MaxSciStrLen]byte` buffer to format
without allocating.

## Why not use other libraries?

Most Go decimal packages are built on `math/big`, which means a pointer, a heap allocation per value and a variable
memory footprint. That is the right trade for arbitrary precision; it is the wrong one for a ledger that moves millions
of fixed-scale amounts. `dec128` fixes the layout at 24 bytes, keeps arithmetic allocation-free, and accepts a 19-place,
128-bit budget in exchange, rounding to fit when a result exceeds it. The numbers below are the result.

## Benchmarks

Measured on Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz. Reproduce with https://github.com/jokruger/go-decimal-benchmark.

```
                                 parse (ns/op)  string (ns/op)     add (ns/op)     mul (ns/op)     div (ns/op)

float64 (baseline)                      27.772          51.262           0.349           0.335           0.356
dec128.Dec128                           11.294          20.596           5.769           3.449          35.308
udecimal.Decimal                        19.254          37.570          10.124           9.776          35.752
alpacadecimal.Decimal                   69.491          67.468         164.113          43.652         362.512
shopspring.Decimal                     123.957         155.077         173.023          47.777         344.075
```

## Notes on Terminology

- **Scale**: Number of digits after the decimal point. For example, 1.00 has scale of 2 and 1.0000 has scale of 4.
- **Exponent**: Same as scale, but in the context of low-level implementation details or Dec128 encoding.
- **Canonical**: The representation of a number with the minimum number of decimal places required to represent the number.
- **Quantum\***: The smallest step at a given scale. For example, scale 2 has quantum 0.01
- **NaN**: A value carrying a failure reason instead of a number. Produced by overflow, division by zero, a parse error and so on; inspect it with `IsNaN` or `ErrorDetails`.
- **Ideal scale**: The scale an exact result of a division or square root takes: the operands' scale difference (or 0) for a quotient, half the scale for a root. Trailing zeros beyond it are removed; inexact results keep the full scale.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

### Attribution

This project includes code derived from:

- A project licensed under the BSD 3-Clause License (Copyright © 2025 Quang).
- A project licensed under the MIT License (Copyright © 2019 Luke Champine).

See the `LICENSE` file for full license texts.
