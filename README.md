# dec128

[![Go Reference](https://pkg.go.dev/badge/github.com/jokruger/dec128.svg)](https://pkg.go.dev/github.com/jokruger/dec128)
[![lint](https://github.com/jokruger/dec128/actions/workflows/lint.yml/badge.svg)](https://github.com/jokruger/dec128/actions/workflows/lint.yml)
[![codecov](https://codecov.io/gh/jokruger/dec128/graph/badge.svg?token=TQWE8PA4AN)](https://codecov.io/gh/jokruger/dec128)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

Zero-dependency 128-bit fixed-point decimal numbers in Go, built for money, ledgers and banking arithmetic:
exact SQL `NUMERIC` semantics, no heap allocation, no panics and no returned errors.

`dec128` is named for its coefficient: a 128-bit integer with a decimal scale, the same fixed-point model as
Apache Arrow's and Parquet's `Decimal128`. It is not IEEE 754 `decimal128`, which is a floating-point format; that is
supported as an interchange encoding only. The arithmetic follows SQL `NUMERIC` as implemented by PostgreSQL, and
the test suite checks it against PostgreSQL digit for digit.

## Key Objectives / Features

- [x] High performance
- [x] Zero dependencies
- [x] Minimal or zero memory allocation
- [x] 38 significant digits (39 near the maximum), up to 19 of them after the decimal point
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

## Ecosystem

- [**pgxdec128**](https://github.com/jokruger/pgxdec128) — a [pgx](https://github.com/jackc/pgx) v5 `pgtype.Codec`
  that reads and writes PostgreSQL `numeric` and `numeric[]` columns as `Dec128` straight from the binary protocol,
  with no `pgtype.Numeric`, no `math/big.Int`, no text round trip and no allocation.
- [**kavun**](https://github.com/jokruger/kavun) — an embeddable scripting language for Go whose `decimal` type is a
  `Dec128`, giving rules and decisioning scripts exact money arithmetic instead of a float workaround.
- [**go-decimal-benchmark**](https://github.com/jokruger/go-decimal-benchmark) — the comparative harness the numbers
  in [Benchmarks](#benchmarks) come from.

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
  does not fit. That is the design, not a failure -- see [Safe operating range](#safe-operating-range) for where it
  starts. `SetLossPolicy` decides whether a dropped digit is acceptable at all.
- **A product of two small values can silently become zero.** The smallest non-zero value is `1e-19`, so
  `1e-10 * 1e-10` is `0` under the default policy. `SetLossPolicy(LossNaNOnUnderflow)` turns that into
  `NaN(Underflow)` while leaving `1/3` a value.
- **`SetDefaultScale`, `SetArithmeticRounding`, `SetLossPolicy`, `SetTrimOutput` and `SetNullValue` are
  process-global.** Set them once during initialization; changing them while other goroutines calculate is a data
  race. `SetArithmeticRounding` also writes the loss policy, so call `SetLossPolicy` second.

## Safe operating range

A `Dec128` carries **38 significant decimal digits** -- 39 while the coefficient is at or below
`340282366920938463463374607431768211455` -- shared freely between the integer and the fractional part, with **at most
19 of them after the decimal point**. That single budget is the whole size contract.

> **In one sentence:** if every operand has at most 9 decimal places and every value, intermediate products included,
> stays below `10^20`, then `Add`, `Sub` and `Mul` are exact and nothing is ever rounded.

`TestSafeZoneClaim` holds the library to that promise. Outside the zone dec128 still works; it just starts trading
fractional digits for integer digits. Two envelopes bound it.

**The top -- precision shrinks as magnitude grows.** All 19 fractional places are available while

```
|x| <= 34028236692093846346.3374607431768211455
```

whose integer part is about `3.4e19`, 3.69 times the largest `int64`. Above that one place is lost per decade, down to
none at `3.4e38`.

**The bottom -- the smallest non-zero value is `1e-19`.** `FromString("1e-20")` returns `NaN(ScaleOutOfRange)`;
arithmetic that would produce a smaller magnitude returns zero.

| decimal places | largest value | `a*b` stays exact while \|a*b\| <= |
|---|---|---|
| 0 | 3.40e38 | 3.40e38 |
| 2 (cents) | 3.40e36 | 3.40e34 |
| 4 | 3.40e34 | 3.40e30 |
| 6 | 3.40e32 | 3.40e26 |
| 8 (satoshi) | 3.40e30 | 3.40e22 |
| 9 | 3.40e29 | 3.40e20 |
| 10 | 3.40e28 | never -- every product rounds |
| 18 (wei) | 3.40e20 | never |
| 19 | 3.40e19 | never |

`Add` and `Sub` never raise the scale, so they stay exact while the running total fits the *largest value* column.
`Mul` adds the scales: a product is exact only when `s1 + s2 <= 19` **and** it fits the third column. At a uniform
working scale that means **9 places or fewer**; from 10 places up every multiplication rounds. `Div` and `Sqrt` compute
to `DefaultScale()` places and are exact only when the result terminates within that many digits -- so they are bounded
by the scale cap rather than by the coefficient, and `1/3` has 19 significant digits, not 38.

**Where this bites.** The top end is unreachable for money: `3.40e36` at two decimal places is many orders of magnitude
beyond any real balance. The bottom end is not. The same `s1 + s2 <= 19` rule that ends exact multiplication also makes
small products collapse:

```go
dec128.FromString("0.000000001").Mul(dec128.FromString("0.000000001"))   // 0.000000000000000001, exact
dec128.FromString("0.0000000001").Mul(dec128.FromString("0.0000000001")) // 0 -- 1e-20 is not representable
```

Rates, probabilities and per-unit factors around `1e-10` reach it. Chain such factors in one expression rather than
storing each intermediate, rescale into a larger unit, or set `SetLossPolicy(LossNaNOnUnderflow)` to make the collapse
loud. `MaxAtScale` and `QuantumAtScale` report the two ends of the range at any scale.

## Scale and rounding

Scale is preserved rather than normalized, so `1.5` and `1.50` are distinct representations that compare equal.
`Add`/`Sub` take the larger operand scale, `Mul` adds them, and a zero result keeps that scale too
(`1.50 - 1.50` is `0.00`). When the exact result needs more than 19 places or more than 128 bits, the scale is reduced
to the largest that fits and the digits below it are rounded with the mode set by `SetArithmeticRounding`
(`ROUND_TOWARD_ZERO` by default) -- unless `SetLossPolicy` says the loss is unacceptable.

### Loss policy

A rounding mode picks a *direction*; whether arithmetic may discard a digit at all is the separate `LossPolicy`,
because losing digits is not one condition but three. An integer part that will not fit at scale 0 is an **overflow**
and is always `NaN(Overflow)`. A result that keeps its significant digits and drops only the tail is **inexact**, which
`1/3` and `Sqrt(2)` are at every scale. A result that rounds to zero from non-zero operands has lost every significant
digit it had: that is an **underflow**, and it is the one case where a plausible-looking value hides a total loss of
information.

| policy | `1/3` | `1e-10 * 1e-10` | `1e20 * 1e20` |
|---|---|---|---|
| `LossRound` (default) | `0.3333333333333333333` | `0` | `NaN(Overflow)` |
| `LossNaNOnUnderflow` | `0.3333333333333333333` | `NaN(Underflow)` | `NaN(Overflow)` |
| `LossNaNOnInexact` | `NaN(Inexact)` | `NaN(Inexact)` | `NaN(Overflow)` |

`LossNaNOnUnderflow` is the recommended setting for money: a non-zero amount never silently becomes zero, and an
inexact division still returns a value. `SetArithmeticRounding(ROUND_NAN)` is the deprecated spelling of
`SetLossPolicy(LossNaNOnInexact)`; it still behaves exactly as it did in v1.1, and a program that sets both must call
`SetLossPolicy` second.

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
  preserved. [pgxdec128](https://github.com/jokruger/pgxdec128) builds a pgx v5 codec for OID 1700 on this pair,
  skipping the `big.Int` and text round trip pgx performs for a `sql.Scanner` target.
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

The Go default, `shopspring/decimal`, is built on `math/big`: a pointer, a heap allocation on every operation and a
variable memory footprint. That is the right trade for arbitrary precision; it is the wrong one for a ledger that moves
millions of fixed-scale amounts. `dec128` fixes the layout at 24 bytes, keeps arithmetic allocation-free, and accepts a
19-place, 128-bit budget in exchange, rounding to fit when a result exceeds it. The numbers below are the result.

| | representation | size | range | allocs per `Mul` | failure model |
|---|---|---|---|---|---|
| **`dec128`** | `uint128` coefficient + scale + sign/state | 24 B | scale 0-19, coefficient < 2^128 | **0**, always | NaN value carrying the reason |
| `shopspring/decimal` | `*big.Int` coefficient + `int32` exponent | 16 B | `int32` exponent, unbounded coefficient | **2**, always | mixed: `error`, panic, silent |
| `cockroachdb/apd/v3` | `apd.BigInt` (128-bit inline array, then heap) + `int32` exponent | 32 B | context precision, exponent +/-100000 | **0** while the coefficient fits 2^128, else 2 | `(Condition, error)` from a `Context`, with traps |
| `alpacahq/alpacadecimal` | `int64` at fixed scale 12, `*shopspring/decimal` fallback | 16 B | fast path abs(v) <= 9223372, then shopspring | **0** on the fast path, 3 on fallback | inherited from `shopspring/decimal` |
| `quagmt/udecimal` | `u128` coefficient + `*big.Int` fallback | 32 B | scale 0-19, coefficient unbounded via fallback | **0** while the coefficient fits 2^128, else 5 | returned `error`; the `Must*` variants panic |

Sizes are `unsafe.Sizeof` on `darwin/amd64`; allocation counts are `testing.AllocsPerRun` on `Mul` with operands on
each library's fast path, and again with a product wider than 2^128. Checked against `shopspring/decimal` v1.4.0,
`cockroachdb/apd/v3` v3.2.3, `alpacahq/alpacadecimal` v0.0.9 and `quagmt/udecimal` v1.10.1 on 2026-09-10.

Note what the table does *not* say: `dec128` is not the only allocation-free option. `apd/v3` inlines a 128-bit array
and takes its destination by pointer, so a steady-state loop over ordinary money values allocates nothing either, and
`udecimal` is allocation-free inside the same 128-bit budget. What separates them is the shape of the API and the price
of generality: `apd` gives you the full General Decimal Arithmetic machinery -- contexts, traps, conditions,
`Ln`/`Exp`/`Pow`, exponents to +/-100000 -- and asks you to thread a `*Context` and a `(Condition, error)` pair through
every operation. `dec128` gives you SQL `NUMERIC` in 128 bits with value semantics, no context, no error plumbing, and
interchange codecs for the wire formats a ledger actually meets.

So: pick `apd` when you need arbitrary precision, transcendentals or the GDA condition model; `shopspring/decimal` when
you want the ecosystem default and precision matters more than throughput or allocation; and `dec128` when the values
are money, the scale is known, and you want the fastest correct answer without an `error` on every line.

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

## When not to use dec128

The 128-bit budget is a deliberate trade, not an oversight. Reach for `math/big`, `cockroachdb/apd` or
`shopspring/decimal` instead when:

- **You need more than 19 decimal places, or values below 1e-19.** `MaxScale` is 19 and is not configurable; it is
  what a 128-bit coefficient buys. Two operands at 10 places already produce a rounded product.
- **Your magnitudes are unbounded.** The largest value at scale 0 is 2^128 - 1, about 3.4 x 10^38. Cryptocurrency
  wei amounts at full precision, factorials and unbounded exponentiation do not fit.
- **You need arbitrary precision or the full IEEE 754-2008 condition model.** `apd` implements the General Decimal
  Arithmetic specification with contexts, traps and conditions; `dec128` implements SQL `NUMERIC` in 128 bits.
- **You need transcendental functions.** `Sqrt` and integer `PowInt` are the extent of it: no `Ln`, `Exp`,
  `Log10` or fractional powers.
- **You want the compiler to make you handle failure.** Arithmetic returns a NaN, not an `error`, so nothing
  forces a check. That is the point of the design, and it is the wrong design for a codebase that relies on
  `errcheck` to catch mistakes.

## Migrating from shopspring/decimal

Verified against `shopspring/decimal` v1.4.0. Every mapping below was checked by running both expressions over
`{1.454, -1.454, 1.455, -1.455, 2.5, -2.5, 1.5, -1.5, 0.005, -0.005}` and comparing the strings.

> [!WARNING]
> **`RoundUp` and `RoundDown` mean opposite things in the two libraries.** `shopspring/decimal` reads them as
> *away from zero* and *toward zero*; `dec128` reads them as the IEEE 754-2019 attributes *roundTowardPositive*
> (ceiling) and *roundTowardNegative* (floor). They agree on positive values and disagree on every negative one, so a
> mechanical rename compiles, passes tests written with positive amounts, and silently changes refunds and credits.
> `shopspring.RoundUp(2)` on `-1.454` gives `-1.46`; `dec128.RoundUp(2)` gives `-1.45`.

### Constructors

| `shopspring/decimal` | `dec128` | note |
|---|---|---|
| `decimal.NewFromString(s)` | `dec128.FromString(s)` | no `error` return; check `ErrorDetails()` at the end of the chain |
| `decimal.RequireFromString(s)` | `dec128.FromString(s)` | returns a NaN instead of panicking |
| `decimal.NewFromInt(i int64)` | `dec128.FromInt64(i)` | also `FromInt(int)` |
| `decimal.NewFromFloat(f)` | `dec128.FromFloat64(f)` | still a lossy conversion; prefer `FromString` |
| `decimal.New(value int64, exp int32)` | `dec128.New(coef uint128.Uint128, scale uint8, neg bool)` | unsigned coefficient plus an explicit sign, and a positive scale where shopspring takes a negative exponent |

### Arithmetic

| `shopspring/decimal` | `dec128` | note |
|---|---|---|
| `d.Add/Sub/Mul` | `d.Add/Sub/Mul` | identical names; `Mul` panics on `int32` exponent overflow in shopspring, returns a NaN here |
| `d.Div(x)` | `d.Div(x)` | scale comes from `SetDefaultScale` (19) rather than `decimal.DivisionPrecision` (16), so `2/3` gains three places |
| `d.DivRound(x, precision int32)` | `d.DivRound(x, scale uint8, mode)` | rounding mode is explicit, never global |
| `d.QuoRem(x, precision int32)` | `d.QuoRem(x)` | no precision argument: the quotient is always an integer |
| `d.Mod(x)` | `d.Mod(x)` | identical |
| `d.Pow(x Decimal)`, `d.PowInt32(n)` | `d.PowInt(n)` | integer exponents only; no `PowWithPrecision`, `Ln`, `ExpHullAbrham`, `Atan` or fractional powers |
| — | `d.Sqrt()`, `d.SqrtRound(scale, mode)` | shopspring has no square root |
| `decimal.Sum/Avg/Min/Max(first, rest...)` | `dec128.Sum/Avg/Min/Max(a, b...)` | identical shape |

### Comparison

| `shopspring/decimal` | `dec128` | note |
|---|---|---|
| `d.Cmp(x)` | `d.Compare(x)` | same `-1/0/1` result |
| `d.Equal`, `d.GreaterThan`, `d.LessThan`, `d.GreaterThanOrEqual`, `d.LessThanOrEqual` | same names | identical |
| `d.Sign()`, `d.IsZero()`, `d.IsNegative()`, `d.IsPositive()` | same names | but all three `Is*` return `false` for a NaN here |

### Rounding

| `shopspring/decimal` | `dec128` | note |
|---|---|---|
| `d.Round(n)` | `d.RoundHalfAwayFromZero(n)` | shopspring's `Round` is ties-away-from-zero; `dec128.Round(n, mode)` exists but takes the mode explicitly |
| `d.RoundBank(n)` | `d.RoundBank(n)` | identical |
| `d.Truncate(n)` | `d.Trunc(n)` | identical |
| `d.RoundUp(n)` | `d.RoundAwayFromZero(n)` | **not** `RoundUp` -- see the warning above |
| `d.RoundDown(n)` | `d.RoundTowardZero(n)` | **not** `RoundDown` -- see the warning above |
| `d.RoundCeil(n)` | `d.RoundUp(n)` | both are toward +infinity |
| `d.RoundFloor(n)` | `d.RoundDown(n)` | both are toward -infinity |
| `d.RoundCash(interval)` | — | no Swiss-rounding equivalent |

### Output and conversion

| `shopspring/decimal` | `dec128` | note |
|---|---|---|
| `d.String()` | `d.String()` | trailing zeros removed in both |
| `d.StringFixed(n)` | `d.RescaleRound(n, dec128.ROUND_HALF_AWAY_FROM_ZERO).StringFixed()` | shopspring's `StringFixed` rounds before padding; `dec128.StringFixed()` takes no argument and prints the value's own scale |
| `d.StringFixedBank(n)` | `d.RescaleRound(n, dec128.ROUND_BANK).StringFixed()` | same shape with ties-to-even |
| `d.IntPart()` | `d.Int64()` | shopspring returns a silently wrong `int64` when the value does not fit (`1e30` gives `5076944270305263616`); `dec128` returns `overflow` |
| `d.InexactFloat64()` | `d.InexactFloat64()` | returns `(float64, error)` here, `float64` alone in shopspring |
| `d.Coefficient() *big.Int`, `d.Exponent() int32` | `d.Coefficient() uint128.Uint128`, `d.Scale() uint8` | the coefficient is unsigned here, so combine it with `Sign()`; and the exponent flips sign, since `1.50` has shopspring exponent `-2` and `dec128` scale `2` (`Exponent()` is a synonym for `Scale()`, not the negated form) |
| `decimal.NullDecimal` | `dec128.SetNullValue(dec128.Null())` | a policy on the type itself rather than a separate wrapper -- see [SQL NULL](#sql-null) |

### Globals

| `shopspring/decimal` | `dec128` | note |
|---|---|---|
| `decimal.DivisionPrecision` (16) | `dec128.SetDefaultScale(n)` (19) | process-global in both; set it once at startup |
| `decimal.MarshalJSONWithoutQuotes` | — | `MarshalJSON` always quotes; `UnmarshalJSON` accepts a quoted string or a bare JSON number |
| — | `dec128.SetArithmeticRounding(mode)` | the direction in which arithmetic discards digits when a result does not fit |
| — | `dec128.SetLossPolicy(policy)` | whether arithmetic may discard a digit at all; `LossNaNOnUnderflow` is recommended for money |
| — | `dec128.SetTrimOutput(true)` | makes `Value`/`MarshalJSON`/`MarshalText` trim trailing zeros, as shopspring does |
| `decimal.PowPrecisionNegativeExponent`, `decimal.ExpMaxIterations` | — | no transcendental functions to configure |

### What changes

1. **Errors become values.** `NewFromString` returns an `error`, `Div` and `QuoRem` panic on a zero divisor, `Mul`
   panics on `int32` exponent overflow, and `IntPart` is silently wrong when the value does not fit. `dec128` answers
   all four with a NaN that propagates until you call `ErrorDetails` or `IsNaN`. Nothing forces the check, so put it at
   the end of every chain that crosses a boundary. See [The contract](#the-contract).
2. **The range is finite.** Values that overflow 128 bits or 19 places are rounded down to a scale that fits, using the
   mode set by `SetArithmeticRounding` (truncation by default), and become `NaN(Overflow)` only when the integer part
   itself does not fit. See [Safe operating range](#safe-operating-range) for where that starts, and `SetLossPolicy`
   for making the loss an error instead.
3. **Rounding modes are explicit, and two names are traps.** `shopspring/decimal` fixes ties-away-from-zero in `Round`
   and reads a package global for division; `dec128` takes the mode per call in `Round`, `RescaleRound`, `MulRound`,
   `DivRound` and `SqrtRound`. Re-read the `RoundUp`/`RoundDown` warning above before renaming anything.
4. **`==` is not a numeric comparison.** `Dec128` is a comparable struct with no pointer, so `==` compiles and silently
   compares representations: `1.5 == 1.50` is false. Use `Equal` or `Compare`, and `Canonical` before using a value as
   a map key. (`shopspring.Decimal` holds a pointer, so `==` is wrong there too -- it just fails more obviously.)
5. **Scale survives serialization by default.** `Value`, `MarshalJSON` and `MarshalText` emit the fixed form, so `1.50`
   reaches a database or a JSON consumer as `1.50`. `SetTrimOutput(true)` restores the trimmed output that
   `shopspring/decimal` produces.

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

See the `NOTICE` file for the full third-party license texts, and `LICENSE` for this project's.
