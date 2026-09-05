# dec128

[![Go Reference](https://pkg.go.dev/badge/github.com/jokruger/dec128.svg)](https://pkg.go.dev/github.com/jokruger/dec128)
[![lint](https://github.com/jokruger/dec128/actions/workflows/lint.yml/badge.svg)](https://github.com/jokruger/dec128/actions/workflows/lint.yml)
[![codecov](https://codecov.io/gh/jokruger/dec128/graph/badge.svg?token=TQWE8PA4AN)](https://codecov.io/gh/jokruger/dec128)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

High performance 128-bit fixed-point decimal numbers in Go, built for financial and
banking arithmetic.

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
- [x] Financially correct rounding
- [x] Correct comparison of numbers encoded in different scales (e.g. 1.0 == 1.00)
- [x] Correct handling of NaN values (e.g. NaN + 1 = NaN)
- [x] Conversion to canonical representation (e.g. 1.0000 -> 1)
- [x] Conversion to fixed string representation (e.g. 1.0000 -> "1.0000")
- [x] Conversion to human-readable string representation (e.g. 1.0000 -> "1")
- [x] Scientific notation: parsed by `FromString`, printed on request (e.g. "1.5e3" -> 1500, 12345 -> "1.2345e+4")
- [x] Per-call scale for division and square root (`DivAtScale`, `SqrtAtScale`), no reliance on global state
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

Linting is configured in `.golangci.yml` and enforced in CI by
`.github/workflows/lint.yml`.

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

**Arithmetic never panics and never returns an error.** A failed operation returns a
NaN carrying the reason, and NaN propagates, so a whole calculation is one expression
checked once at the end:

```go
total := principal.Mul(rate).Add(fee).RoundBank(2)
if err := total.ErrorDetails(); err != nil {
    return err
}
```

`ErrorDetails` returns a plain `error` and is the terminal check for a chain; `IsNaN`
is the boolean form. Nothing forces you to call either — unlike an ignored `err` there
is no compiler or linter backstop, so make the check part of your review habits.

Four things that surprise people:

- **`IsZero`, `IsNegative` and `IsPositive` all return `false` for NaN.** A guard
  written as `if !d.IsZero()` does *not* catch a failed calculation. Test `IsNaN`
  first when the difference matters.
- **`MarshalJSON` encodes NaN as the string `"NaN"`.** A consumer that accepts strings
  takes it without complaint. `Value` is safer by construction — a numeric column
  rejects it.
- **Scale is capped at `MaxScale` (19) and the cap is hard.** An operation whose exact
  result needs more digits returns NaN rather than rounding silently. Round explicitly
  when a calculation is finished.
- **`SetDefaultScale` and `SetNullValue` are process-global.** Set them once during
  initialisation; changing them while other goroutines calculate is a data race.

## Scale and division

Scale is preserved rather than normalised, so `1.5` and `1.50` are distinct
representations that compare equal. `Add`/`Sub` take the larger operand scale, `Mul`
adds them, and `Div` uses the package default (`SetDefaultScale`, 6 by default) as a
**floor** — an operand with a larger scale keeps its own. `QuoRem` does not consult it
at all: the quotient is always an integer.

Because the cap is hard, **dividing early costs precision and dividing at a scale near
`MaxScale` leaves no room to multiply afterwards**:

```go
// 5.0/365/100 truncated to 6 dp before the multiplications: 0.7% low
principal.Mul(annual.Div(d365).Div(d100)).MulInt(days)  // 4.08

// one division, last: correct
principal.Mul(annual).MulInt(days).Div(dec128.FromInt64(36500))  // 4.109589 -> 4.11
```

`DivAtScale` and `SqrtAtScale` take the scale per call and ignore the global entirely,
which is the right tool for a rate that needs more digits than your default:

```go
rate := annual.DivAtScale(dec128.FromInt64(365), 12)  // 0.013698630136
```

## SQL NULL

By default a `NULL` scans to `Zero`, which loses the difference between "no amount
recorded" and "the amount is zero". `SetNullValue` makes it round-trip instead:

```go
dec128.SetNullValue(dec128.Null())   // once, during initialisation

var amount dec128.Dec128
_ = rows.Scan(&amount)
amount.IsNull()          // true for a NULL column
amount.Add(fee).IsNull() // true - NULL propagates the way SQL NULL does
v, _ := amount.Value()   // nil, so it writes back as NULL
```

A NULL-marked value is a NaN, so it propagates through arithmetic, but `IsNull`
distinguishes it from an overflow or a parse failure. `MarshalJSON` emits `null` for
it and `UnmarshalJSON` accepts `null` symmetrically. The default stays `Zero`, so
nothing changes unless you opt in.

## Scientific notation

`FromString` accepts both forms, so JSON, text and `sql.Scanner` input needs no extra
handling:

```go
dec128.FromString("1.5e3")   // 1500
dec128.FromString("-2.5E-2") // -0.025
```

`FromSafeString` is regular form only. It skips all format checks by design, so there
is nothing for the exponent marker to be caught by, and adding a check would cost a
comparison per character on the fastest parsing path. Send input that may carry an
exponent through `FromString`, which detects the marker inside the validation it
already performs and so costs nothing extra.

Printing stays in the regular form unless the scientific one is asked for. `String`,
`MarshalJSON`, `MarshalText` and `Value` are unchanged:

```go
d := dec128.FromString("12345")
d.String()    // "12345"
d.StringSci() // "1.2345e+4"
```

`StringSci` normalises to a single leading digit, always signs the exponent and drops
the trailing zeros of the mantissa; zero prints as `0e+0` and NaN as `NaN`. Use
`StringSciToBuf` with a `[MaxSciStrLen]byte` buffer to format without allocating.

## Why not use other libraries?

Most Go decimal packages are built on `math/big`, which means a pointer, a heap
allocation per value and a variable memory footprint. That is the right trade for
arbitrary precision; it is the wrong one for a ledger that moves millions of fixed-scale
amounts. `dec128` fixes the layout at 24 bytes, keeps arithmetic allocation-free, and
accepts a hard 19-digit scale cap in exchange. The numbers below are the result.

## Benchmarks

Measured on a MacBook Pro (2019), 2.6 GHz 6-core Intel Core i7, 16 GB RAM. Reproduce
with https://github.com/jokruger/go-decimal-benchmark.

These figures predate the current default scale of 6 and are conservative for division
and square root, both of which do materially less work at a smaller scale — locally
`Div` measures ~15 ns and `Sqrt` ~78 ns at the default, against ~36 ns and ~330 ns at
scale 19. Re-run the suite before quoting the table.

```
                                 parse (ns/op)  string (ns/op)     add (ns/op)     mul (ns/op)     div (ns/op)

dec128.Dec128                           14.024          33.683           9.975           6.569          35.116
udecimal.Decimal                        23.302          41.854          12.226          11.346          40.877
alpacadecimal.Decimal                   89.528          78.884         206.393          60.364         451.828
shopspring.Decimal                     152.263         169.300         218.909          65.241         428.002
```

## Notes on Terminology

- **Scale**: Number of digits after the decimal point. For example, 1.00 has scale of 2 and 1.0000 has scale of 4.
- **Exponent**: Same as scale, but in the context of low-level implementation details or Dec128 encoding.
- **Canonical**: The representation of a number with the minimum number of decimal places required to represent the number.
- **Quantum\***: The smallest step at a given scale. For example, scale 2 has quantum 0.01
- **NaN**: A value carrying a failure reason instead of a number. Produced by overflow, division by zero, a parse error and so on; inspect it with `IsNaN` or `ErrorDetails`.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

### Attribution

This project includes code derived from:

- A project licensed under the BSD 3-Clause License (Copyright © 2025 Quang).
- A project licensed under the MIT License (Copyright © 2019 Luke Champine).

See the `LICENSE` file for full license texts.
