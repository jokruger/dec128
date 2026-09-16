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

### The core, and the completeness set

Two claims run through this README and they apply to different parts of the library, so it is worth separating them
before anything else.

**The core is the financial one** -- the decimal type itself, `Add`, `Sub`, `Mul`, `Div`, comparison, rounding,
allocation, splitting, parsing, printing and the interchange codecs. It is what `dec128` is for and what its promises
are about: fixed 24-byte layout, **zero heap allocation**, and every result exact or rounded by a single documented
decision. `TestAllocationGates` holds every one of these operations to zero allocations per call.

**The completeness set is the scientific one** -- roots, rational powers, and the transcendentals `Exp`, `Ln`, `Ln1p`,
`Expm1`, `Log10`, `Log2` and `Pow`. These are here so the type is a complete numeric type rather than because a ledger
needs them, and they do not carry the core's promises:

| | allocates | exactness |
|---|---|---|
| the core, and `PowIntRound` | never | exact, or one documented rounding (`PowIntRound` is faithfully rounded) |
| `NthRootRound`, `PowRational` | yes | **correctly rounded** -- the decision is an exact integer comparison |
| `Exp`, `Ln`, `Ln1p`, `Expm1`, `Log10`, `Log2`, `Pow` | yes, unless `Pow` gets an integer exponent or the logarithm gets an exact power of its base | **faithfully rounded** -- within one unit in the last place, measured, not proven |
| `Prod` and its forms | yes | exact in the middle, **one rounding** at the end |

They allocate because their exact intermediate is wider than any fixed register, so they run on `math/big`. Both lists
are maintained in the package documentation and pinned from both sides by the test suite: everything not on the
allocating list is held to zero allocations, and everything on it is checked to really allocate. If you are counting
allocations in a hot path, stay in the core; the completeness set is for the calculation you do once.

## Key Objectives / Features

- [x] High performance
- [x] Zero dependencies
- [x] Zero memory allocation throughout the core (see above; the scientific completeness set allocates)
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
- [x] Per-call scale and rounding mode for every operation (`AddRound`, `SubRound`, `MulRound`, `DivRound`, `SqrtRound`, `PowIntRound`) -- a documented subset that reads no global configuration, so the same inputs give the same bytes in every process
- [x] Fused multiply-add (`MulAddRound`), fused multiply-divide (`MulDivRound`), fused add-and-divide (`AddQuoRound`) and a wide exact accumulator (`Accumulator`, with `AddMul`): a chain of multiply-accumulate rounds once, not once per term
- [x] Exact products with one rounding (`Prod`, `ProdRound`, `ProdSlice`, `ProdSliceRound`): chain-linking a year of daily factors drifts by several units in the last place if folded with `Mul`, and by nothing here
- [x] Integer powers with guard digits (`PowIntRound`): `(1+r)^360` correctly rounded rather than carrying a dozen roundings
- [x] Exact splitting of an amount (`Allocate`, `Split`, `AllocateResidual`, `SplitResidual`): the shares sum to the whole, digit for digit, by largest remainder or with the difference going to a named party
- [x] Rounding to a negative number of places, to a multiple and to a number of significant digits (`RoundToPlaces`, `RoundToMultiple`, `RoundToSignificant`) for cash rounding, disclosure rules and quoted rates
- [x] The n-th root (`NthRootRound`), correctly rounded: the inverse of compounding, for converting an effective rate to a periodic one
- [x] Column checks before the insert (`FitsNumeric`, `SignificantDigits`, `IntegerDigits`) and `RescaleRoundInexact` to make a value fit and say what it cost
- [x] `fmt.Formatter`, so `%.2f` on a `Dec128` means what it says
- [x] Interchange codecs: PostgreSQL `numeric` binary, IEEE 754 decimal128 (BID), Arrow/Parquet int128, and the SQL driver `Decompose`/`Compose` pair
- [x] An exact `math/big` bridge (`Rat`, `BigInt`, `FromRat`): a `Dec128` is a rational, and the conversion says so in both directions
- [x] The General Decimal Arithmetic operations the specification names: `Log10`, `Log2`, `Logb`, `CopySign`, `CmpTotal`, `RemainderNear` and `Inv`
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

### What the globals reach, and what they do not

The five `Set*` functions are process-global and are meant to be set once at start-up, if at all. It is worth being
precise about their reach, because the natural reading -- that they are "this application's scale and rounding" -- is
not what they are:

* **`SetDefaultScale` is not a maximum scale.** The maximum is `MaxScale`, a compile-time constant of 19, and nothing
  configures it. The default scale is only the scale `Div`, `Sqrt` and `Inv` compute to when no scale is given. It
  does not touch `Add`, `Sub` or `Mul`, whose scale comes from the operands -- `Mul` adds the scales -- and is reduced
  only when the exact result will not fit.
* **`SetArithmeticRounding` is not the application's rounding rule.** For `Add`, `Sub` and `Mul` it fires only when
  digits *must* go, past 19 places or past 128 bits, which is the overflow path and not the everyday one. For `Div`
  and `Sqrt` it applies to every call, because those round by nature.
* **`SetLossPolicy`** decides whether discarding a digit is allowed at all, and it is the one of the three worth
  setting for money.

So the shape a calculation wants is: **keep the intermediates wide, round once at the end, at the boundary.**

```go
// start-up: nothing, or at most the loss policy
dec128.SetLossPolicy(dec128.LossNaNOnUnderflow) // a non-zero amount never silently becomes zero

// the calculation: exact throughout, no business rounding yet
gross := qty.Mul(unitPrice)      // exact: the scales add
vat := gross.MulPercent(vatRate) // exact: two places more again
total := gross.Add(vat)          // exact

// the boundary: the business scale and the business rounding, once
out := total.RescaleRound(6, dec128.ROUND_BANK)
```

Setting `SetDefaultScale(6)` to say "this application works to six places" does something other than it looks like:
it leaves `Mul` alone and makes every `Div` throw away thirteen digits the later steps would have used, so `1/3` is
`0.333333` and every step after it inherits that. Leave it at 19 and ask for six places where six places is the
answer -- `DivRound(x, 6, mode)` for a quotient that is presented, `RescaleRound(6, mode)` at the end of a chain.
Likewise `SetArithmeticRounding(ROUND_BANK)` changes the direction taken when a result has run out of digits, not how
the application's amounts are rounded; if banker's rounding is the business rule, it belongs in the `*Round` call that
produces the presented value. Both are per-call arguments for a reason, and a library that has to produce the same
bytes in every process should not read the globals at all -- see the global-free subset below.

### A worked example: six places, banker's rounding

Take the common fintech requirement: every amount in and out of the application is held at **six decimal places and
rounded bankers' way**. That needs no configuration at all -- no `Set*` call anywhere. The scale and the mode are
arguments; what makes the result right is *where* they are applied. Here is an interest accrual with a fee and a
three-way split, pinned as `Example_sixPlacesBankersRounding` in the test suite, so the numbers below are the ones
the code produces:

```go
const (
    scale = 6
    mode  = dec128.ROUND_BANK
)

// In: put the inputs on the ledger's grid, once, at the boundary.
principal := dec128.FromString("12345.678901").RescaleRound(scale, mode)
annualRate := dec128.FromString("4.25") // per cent per year
feeRate := dec128.FromString("1.5")     // per cent of the interest

// Middle: exact. MulPercent adds two places to the scales of its operands rather than rounding, so the
// yearly interest is the exact product; the day count is the one rounding, taken at the ledger's scale.
yearly := principal.MulPercent(annualRate)                // 524.6913532925 -- 6+2+2 places, nothing discarded
interest := yearly.MulDivRoundInt64(31, 365, scale, mode) // 44.562827      -- the single rounding decision
fee := interest.MulPercentRound(feeRate, scale, mode)     // 0.668442
net := interest.Sub(fee)                                  // 43.894385      -- exact: both are already at six places

// Out: one check for the whole chain, and a split whose shares add back up to it exactly.
if err := net.ErrorDetails(); err != nil {
    return err
}
shares, ok := net.Split(3, scale) // 14.631462, 14.631462, 14.631461 -- and they sum to 43.894385 exactly
if !ok {
    return errShareTooWide
}
```

Five things are worth naming, because they are the whole method:

* **No global is set, and none is read.** `RescaleRound`, `MulPercentRound`, `MulDivRoundInt64`, `Sub` and `Split`
  are all in the global-free subset; `MulPercent` would read `SetArithmeticRounding` only if the exact product would
  not fit -- and at ledger magnitudes it fits. Drop this code into a process whose `init` set the globals to
  something else and it returns the same bytes.
* **One rounding per output, not one per operation.** `principal * 4.25% * 31/365` is a multiplication and two
  divisions, and it makes exactly one rounding decision -- inside `MulDivRoundInt64`, on the exact numerator. Written
  as three rounded steps it would make three, each one biasing the next.
* **The six is asked for, never assumed.** Every call that can put a value off the grid takes `scale` and `mode`
  explicitly, so "six places, banker's" is visible at each point where it actually applies, and a different rule for
  one field -- a tax line at two places, a rate at nine -- is a different argument rather than a different process.
* **Exact steps stay exact.** `interest.Sub(fee)` rounds nothing: both operands already stand at six places, so
  there is nothing to discard. The type only rounds where the exact answer will not fit.
* **The parts add up to the whole.** `Split` hands out the quanta by largest remainder, so the reconciliation
  `dec128.SumSliceRound(shares, scale, mode).Equal(net)` is true exactly, not within a tolerance.

The one global worth considering for a ledger is `SetLossPolicy(LossNaNOnUnderflow)`, at start-up and never again: it
is not a scale or a rounding rule but a safety net, turning "a non-zero amount silently became zero" into a NaN that
`ErrorDetails` reports.

## Determinism: the global-free subset

Three of the five process-global settings can change the value an operation returns: `SetDefaultScale`,
`SetArithmeticRounding` and `SetLossPolicy`. A library that must produce the same bytes in every process, whatever
some other package passed to those functions at init time, cannot use the operations that read them.

`dec128` therefore documents and tests a **global-free subset**. It excludes `Add`, `Sub`, `Mul`, `MulScaled` and
`MulPercent` when the exact result does not fit, `Div`, `Inv`, `Sqrt`, `PowInt64`, `Sum`, `SumSlice`, `Avg`, `Prod`,
`ProdSlice` and `EncodeIEEE`, and includes everything else -- in particular the whole `*Round` family,
`NthRootRound`, `Accumulator`, `QuoRem`, `Mod`, `RemainderNear`, the `Round*` methods, `Allocate`, `Format` and the
codecs. Each excluded operation has a twin inside the subset: `AddRound`, `SubRound`, `MulRound`, `MulScaledRound`,
`MulPercentRound`, `DivRound`, `InvRound`, `SqrtRound`, `PowIntRound`, `SumRound`, `SumSliceRound`, `AvgRound`,
`ProdRound`, `ProdSliceRound` and `EncodeIEEERound`, so nothing has to be given up to stay deterministic. `TestGlobalFreeSubset` runs every operation on the list under the whole matrix of the three settings and
requires the results to be bit-identical; its complement checks that the operations left out really do depend on
them. The package documentation carries the maintained list.

## Fused and exact operations

Every intermediate rounding is a place where two implementations can diverge and where bias accumulates, so the
operations a financial formula leans on have forms that round once.

```go
// d*b + c with the product held exactly and one rounding, instead of three
pv := principal.MulAddRound(rate, fee, 2, dec128.ROUND_BANK)

// d*b/c with the numerator and the divisor kept apart until one rounding
share := fee.MulDivRound(weight, totalWeight, 2, dec128.ROUND_HALF_AWAY_FROM_ZERO)

// the same, when the rational arrives as a pair of integers - a day count over a year basis
accrued := interest.MulDivRoundInt64(31, 365, 2, dec128.ROUND_HALF_AWAY_FROM_ZERO)

// a percentage: the product is exact and the division by a hundred is a move of the point
vat := net.MulPercent(rate)                             // exact, two places more than net and rate together
fee := amount.MulPercentRound(rate, 2, dec128.ROUND_BANK) // or straight to the presented scale, one rounding

// a sum of products that is exact until Total: NPV, weighted averages, schedule reconciliation
acc := dec128.NewAccumulator(2)
for i, cf := range cashflows {
    acc.AddMul(cf, discountFactors[i])
}
npv := acc.Total(2, dec128.ROUND_HALF_AWAY_FROM_ZERO)

// compounding with guard digits: the running product carries 57 places, not 19
factor := dec128.FromString("1.000164383561643836").PowIntRound(3650, 19, dec128.ROUND_HALF_AWAY_FROM_ZERO)
```

`MulDivRound` is the multiplicative counterpart of `MulAddRound`, and it is the one operation here that cannot be
composed at all rather than merely composed badly: scaling by an exact rational whose decimal expansion does not
terminate -- 1/3, 2/7, 31/365 -- has no decimal factor to convert to first, so the numerator and the divisor have to
stay apart until the single division. `MulRound` followed by `DivRound` rounds twice, and on a proration of
1119.32 by 25.12/204.20 the two answers differ by a quantum. Its exact numerator reaches 2^383, which is why the
intermediate is held in the 384-bit register rather than the 256-bit one a product needs.

`MulPercent` is the one of these that turns up in every schedule of fees, rates and taxes, and it is a fused
operation for the same reason as the others. `amount.Mul(rate).Div(dec128.Decimal100)` brings the product to a scale
under the process-global mode and then rounds the quotient again, two decisions where one will do, and it fails on an
intermediate the result never needed: `3e37` at 50 per cent is `NaN(Overflow)` that way round and `1.5e37` here,
because the exact product is held in a register wider than a coefficient until the single reduction. It follows the
scale rule as `Mul` does, so on money-sized operands it is exact and the rounding to the presented scale stays where
it belongs -- at the end:

```go
vat := net.MulPercent(rate).RescaleRound(2, dec128.ROUND_BANK) // 100.00 * 7.5% = 7.50000 -> 7.50
```

`MulScaled` and `MulScaledRound` are the general form, `d * f * 10^k`: a per-mille at `k = -3`, a basis point at
`k = -4`, a price quoted per hundred units at `k = 2`. `ScaleByPow10` is the same move of the point with no
multiplication, for converting a quoted rate to a fraction on its own.

`Accumulator` is the one mutable type in the package, and deliberately so: positive and negative terms are kept apart
and cancelled once, at the end, so the total does not depend on the order the terms arrived in. `Mean` divides that
exact total by the number of terms under the same single rounding, `Reset` empties it for the next batch, its zero
value is a usable empty accumulator, and nothing in it reads the globals. `Sum` and `SumSlice` are the same idea for a
plain sum, over a narrower register, and `SumRound`, `SumSliceRound` and `AvgRound` are their global-free spellings --
`AvgRound` divides the exact total by the count under one rounding, where `Avg` rounds the total and then rounds the
quotient.

## Splitting an amount

`Allocate` divides a value into shares proportional to a list of ratios, and `Split` into `n` equal ones, by the
largest-remainder method: every share gets the whole quanta its exact proportion is worth, and the quanta left over go
one each to the largest fractions, ties to the lowest index. The shares sum to the original value **exactly**, so a
reconciliation is an equality and not an epsilon check.

```go
shares, ok := dec128.FromString("100.00").Allocate([]dec128.Dec128{one, one, one}, 2)
// 33.34, 33.33, 33.33 -- and Sum(shares...) is exactly 100.00

parts, ok := dec128.FromString("0.05").Split(3, 2)  // 0.02, 0.02, 0.01
```

`AllocateResidual` and `SplitResidual` are the other convention, the one an amortization schedule and a syndicated
facility use: every share but one is its proportion rounded the agreed way, and the share at an index you name takes
whatever is left.

```go
// the last installment absorbs the rounding of the others
parts, ok := dec128.FromString("1000.00").SplitResidual(7, 2, 6, dec128.ROUND_HALF_AWAY_FROM_ZERO)
// 142.86 x6, then 142.84 -- and they sum to exactly 1000.00
```

The shares still sum to the whole exactly, but only the others are within a quantum of their proportions: the named
one absorbs all the rounding, and for an amount of a few quanta it can even come out on the other side of zero.

`AppendAllocate`, `AppendSplit` and their residual forms append to a slice the caller keeps; a split of up to 32 ways
then allocates nothing.

## Grids that are not a scale

```go
dec128.FromString("2.37").RoundToMultiple(five, dec128.ROUND_HALF_AWAY_FROM_ZERO) // 2.35, Swiss cash rounding
dec128.FromString("183.47").RoundToMultiple(ten, dec128.ROUND_UP)                 // 190, the next whole ten
dec128.FromString("1234567").RoundToPlaces(-3, dec128.ROUND_HALF_AWAY_FROM_ZERO)  // 1235000
dec128.FromString("5.25").ScaleByPow10(-2)                                        // 0.0525, no division
dec128.FromString("1.09875").RoundToSignificant(5, dec128.ROUND_BANK)             // 1.0988, an FX quote
```

## Roots, and the shape of a value

`NthRootRound` is the inverse of `PowIntRound` and the primitive of rate conversion, and `PowRational` is the general
form: `d^(p/q)` for any reduced fraction, which is the factor of a compound change spread over a fractional number of
periods. Both are **correctly rounded** -- a stronger guarantee than `PowIntRound`'s faithful rounding -- because the
decision is an exact integer comparison rather than a guarded approximation. `d^(p/q)` is the `q`-th root of `d^p`
and `d^p` is an exact rational, so the whole computation is one integer root and two integer comparisons with nothing
rounded on the way.

```go
annual := dec128.FromString("1.126825")
monthly := annual.NthRootRound(12, 10, dec128.ROUND_HALF_AWAY_FROM_ZERO) // 1.0099999977

// five months of a 6% year: the exponent 5/12 has no decimal form at all
part := dec128.FromString("1.06").PowRational(5, 12, 19, dec128.ROUND_HALF_AWAY_FROM_ZERO)
// 1.0245758393924285985, where a twelfth root raised to the fifth gives ...983
```

Taking the root and the power separately rounds twice, and the root is where the significant digits are lost, so the
error is then amplified rather than cancelled. Reduction happens first, and for a negative base it is the whole of
the answer: `(-8)^(2/6)` is `(-8)^(1/3)` and so `-2`, not `64^(1/6)` and so `2`.

Both allocate, for the reason given at the top of this README: the comparison their rounding decision needs is wider
than any register here, so they run on `math/big`. The bound on the exponent is a measurement rather than a caution:
at the worst operand the package has, a degree of 1024 costs 1--3 ms, 10950 costs 23--89 ms and 16384 costs
44--151 ms, so 16384 is where it is refused. That covers 30 years of daily rests (10950) and 40 years of them
(14600), and refuses a century of them, which is not an instrument.

## Exp, Ln and the rest of the completeness set

`Exp`, `Ln`, `Ln1p`, `Expm1`, `Log10`, `Log2` and `Pow` complete the type. They are **the only operations here that
are not exact**, and they say so: everything else computes an exact intermediate and makes one rounding decision on
it, while a transcendental is irrational for all but a handful of arguments and has no exact intermediate to decide
on. `Log10` and `Log2` are the exception within the exception: an exact power of the base is detected from the
coefficient and answered exactly, which the General Decimal Arithmetic specification requires of `log10` and which a
series alone would not give -- converging on 3 from below, a directed rounding mode would return 2.999... instead.

```go
r := dec128.FromString("0.06")
force := r.Ln1p(19, dec128.ROUND_HALF_AWAY_FROM_ZERO)      // the continuously-compounded equivalent of 6%
back := force.Expm1(19, dec128.ROUND_HALF_AWAY_FROM_ZERO)  // and back again

part := dec128.FromString("1.06").Pow(dec128.FromString("0.4166666666666666667"), 19, dec128.ROUND_BANK)
```

**The guarantee is faithful rounding: within one unit in the last place of the correctly rounded value.** That is a
measurement, not a proof, and here is the measurement. Against `testdata/transcendental_golden.csv` -- 80-digit values
from an implementation of the General Decimal Arithmetic specification, in which `exp` and `ln` are correctly rounded
-- the methods were compared at five scales in every rounding mode, 6097 results, and **every one was correctly
rounded**. A second, randomized oracle computes `Exp` and `Ln` again by a construction sharing nothing with the one
under test -- the plain Taylor series in 512-bit binary floating point, with the logarithm obtained by inverting it
with Newton's method -- and over 28000 further results the largest error was again **zero**. Comparing the general
arm of `Pow` against the exact `PowRational` over random operands, 1515 results, the largest error was **one unit in
the last place**.

That one unit cannot be engineered away with more guard digits. It shows up where the exact value is itself
representable -- 19 to the power 24/8 is exactly 6859 -- and a series converging on it from below is truncated by a
directed rounding mode; deciding those correctly means proving the value exact, which is the table-maker's dilemma.
Where an exact answer matters and the exponent is a short fraction, `PowRational` gives one, and `Pow` reaches
`PowRational` by itself: `x.Pow(half, ...)` is bit-identical to `x.SqrtRound(...)`.

`Pow` dispatches on the exponent -- an integer goes to `PowIntRound`, which allocates nothing; a fraction that reduces
to small halves goes to `PowRational`, which is correctly rounded; anything else is `exp(e·ln d)`. `Ln1p` and `Expm1`
are the usual entry points for one-plus-something-small, but note that the cancellation they exist to prevent in a
binary format **does not arise here**: a Dec128 carries 39 significant digits, so `1+x` is exact and `Ln(One.Add(x))`
loses nothing. They are the right thing to write and they stay exact at the top of the range; they do not give you
back digits that `Ln` would have lost, and the test suite pins the two to agree.

Three methods report the shape of a value for the column it has to live in, so that a value too wide for its
`NUMERIC(p, s)` is caught where it is computed and not by the database halfway through a batch:

```go
amount.FitsNumeric(19, 4)         // does it go into NUMERIC(19,4) exactly?
amount.SignificantDigits()        // the p it needs
amount.IntegerDigits()            // the p - s it needs
v, lost := amount.RescaleRoundInexact(4, dec128.ROUND_BANK) // make it fit, and say whether that cost anything
```

`IsInteger`, `IntFrac` and `Clamp` are the small value operations a schedule and a limit check need; `IntFrac` splits
a value so that the parts add back up exactly, `-1.25` giving `-1` and `-0.25`.

## Printing

`Dec128` implements `fmt.Formatter`, so the verbs an amount is printed with do what they say:

```go
fmt.Sprintf("%v", d)      // 1.5      -- String
fmt.Sprintf("%f", d)      // 1.50     -- StringFixed, the value's own scale
fmt.Sprintf("%.2f", d)    // 1.50     -- exactly two places, rounded half to even as Go's own %f rounds a float64
fmt.Sprintf("%8.2f", d)   // "    1.50"
fmt.Sprintf("%e", d)      // 1.5e+0
fmt.Sprintf("%d", d)      // 1        -- the integer part
```

Note that `%.2f` rounds the *decimal*: `2.675` becomes `2.68`, where Go's `%.2f` on the float64 `2.675` gives `2.67`,
because that float is really 2.67499999999999982236431605997495353221893310546875.

## Text output

`String` removes trailing zeros; `StringFixed` keeps the value's scale. `Value` (for `database/sql`), `MarshalJSON`
and `MarshalText` emit the fixed form, so `1.50` reaches PostgreSQL and JSON consumers as `1.50` and its scale survives
the round trip. `SetTrimOutput(true)` restores the trimmed output of versions up to v1.0.20.

## Interchange formats

Besides its own compact binary form (`EncodeBinary`, at most 18 bytes), `dec128` reads and writes four external
formats, all into caller-supplied buffers without allocating:

- **PostgreSQL `numeric` binary** (`EncodePgNumeric`, `DecodePgNumeric`): the wire format of the binary protocol, dscale
  preserved. [pgxdec128](https://github.com/jokruger/pgxdec128) builds a pgx v5 codec for OID 1700 on this pair,
  skipping the `big.Int` and text round trip pgx performs for a `sql.Scanner` target.
- **IEEE 754 decimal128, BID encoding** (`EncodeIEEE`, `DecodeIEEE`): 16 bytes, as used by MongoDB's BSON Decimal128.
  A coefficient wider than 34 digits is rounded on export with the configured mode (`ROUND_NAN` refuses); `FitsIEEE`
  tells in advance.
- **int128 with a schema scale** (`EncodeInt128`, `DecodeInt128`): Apache Arrow (little-endian) and
  Parquet (big-endian) `Decimal128`.
- **The SQL driver decomposer interface** (`Decompose`, `Compose`): the form/sign/coefficient/exponent shape that
  the pgx and go-mssqldb drivers all speak. A coefficient with digits below `MaxScale` that cannot be
  cancelled away is refused rather than rounded. This type has no infinity, so composing one is an error rather than a
  silent NaN.

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
- **You need transcendental functions at more than 19 places, or correctly rounded.** `Exp`, `Ln`, `Ln1p`, `Expm1`,
  `Log10`, `Log2` and `Pow` are here and are faithfully rounded, but they stop at `MaxScale` and they do not claim
  correct rounding. An arbitrary-precision context is `apd`'s job, not this one's.
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
| `d.Pow(x Decimal)`, `d.PowInt32(n)` | `d.PowInt(n)`, `d.PowIntRound(n, scale, mode)` | integer exponents only; no `PowWithPrecision`, `Ln`, `ExpHullAbrham`, `Atan` or fractional powers. Both keep the running product at 57 places and round once |
| — | `d.MulAddRound(b, c, scale, mode)` | fused multiply-add; shopspring has no fused form |
| — | `d.MulPercent(f)`, `d.MulPercentRound(f, scale, mode)`, `d.MulScaled(f, k)` | a percentage, a per-mille or a basis point with one rounding instead of a multiplication and a division |
| — | `dec128.NewAccumulator(scale)` | an exact wide running total with `Add`, `AddMul` and one rounding in `Total` |
| — | `d.Sqrt()`, `d.SqrtRound(scale, mode)`, `d.NthRootRound(n, scale, mode)` | shopspring has no roots |
| `decimal.Sum/Avg/Min/Max(first, rest...)` | `dec128.Sum/Avg/Min/Max(a, b...)` | identical shape |
| — | `d.Allocate(ratios, scale)`, `d.Split(n, scale)` | exact splitting; shopspring has neither |
| — | `d.AllocateResidual(ratios, scale, i, mode)`, `d.SplitResidual(n, scale, i, mode)` | the same, with the rounding difference going to a named share |
| — | `d.Clamp(lo, hi)`, `d.IsInteger()`, `d.IntFrac()` | shopspring has none of the three |

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
| `d.RoundCash(interval)` | `d.RoundToMultiple(m, mode)` | any positive multiple, not only the five shopspring intervals, and the mode is explicit |
| — | `d.RoundToPlaces(places int8, mode)` | `places` may be negative: rounding to the nearest hundred or thousand |
| — | `d.RoundToSignificant(digits, mode)` | rounding to a number of significant digits, for a quoted rate |

### Output and conversion

| `shopspring/decimal` | `dec128` | note |
|---|---|---|
| `d.String()` | `d.String()` | trailing zeros removed in both |
| `d.StringFixed(n)` | `d.RescaleRound(n, dec128.ROUND_HALF_AWAY_FROM_ZERO).StringFixed()` | shopspring's `StringFixed` rounds before padding; `dec128.StringFixed()` takes no argument and prints the value's own scale |
| `d.StringFixed(n)` | `fmt.Sprintf("%.*f", n, d)` | the same through `fmt.Formatter`, but rounding half to even as Go's `%f` does |
| — | `d.FitsNumeric(p, s)`, `d.SignificantDigits()`, `d.IntegerDigits()` | the shape of the value, for the column it is going into |
| — | `d.RescaleRoundInexact(n, mode)` | rescale, and say whether anything was discarded |
| `d.StringFixedBank(n)` | `d.RescaleRound(n, dec128.ROUND_BANK).StringFixed()` | same shape with ties-to-even |
| `d.IntPart()` | `d.Int64()` | shopspring returns a silently wrong `int64` when the value does not fit (`1e30` gives `5076944270305263616`); `dec128` returns `overflow` |
| `d.InexactFloat64()` | `d.InexactFloat64()` | returns `(float64, error)` here, `float64` alone in shopspring |
| `d.Rat() *big.Rat` | `d.Rat() (*big.Rat, error)` | exact in both; a NaN has no rational value, so it returns the NaN's reason |
| `d.BigInt() *big.Int` | `d.BigInt() (*big.Int, error)` | the truncated integer part in both, not the coefficient; this is the only exit for the integer part of a value past the `int64` range |
| `d.BigFloat() *big.Float` | `new(big.Float).SetRat(r)` on `d.Rat()` | shopspring's goes through `String()` and loses digits; via `Rat` it is exact and the caller picks the precision |
| — | `dec128.FromRat(r, scale, mode)` | the way back in, with a single rounding rather than the two a formatted round trip costs |
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
