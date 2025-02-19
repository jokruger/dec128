# dec128

[![GoDoc](https://godoc.org/github.com/jokruger/dec128?status.svg)](https://godoc.org/github.com/jokruger/dec128) 
[![Go Report Card](https://goreportcard.com/badge/github.com/jokruger/dec128)](https://goreportcard.com/report/github.com/jokruger/dec128)

128-bit fixed-point decimal numbers in go.

## Key Objectives / Features
- [x] High performance
- [x] Minimal or zero memory allocation
- [x] Precision up to 19 decimal places
- [x] Fixed size memory layout (128 bits)
- [x] No panic or error arithmetics (use NaN instead)
- [x] Immutability (methods return new instances)
- [x] Basic arithmetic operations required for financial calculations (specifically for banking and accounting)
- [ ] Additional arithmetic operations for scientific calculations
- [x] Easy to use
- [x] Easy to inegrate with external systems (e.g. databases, accounting systems, JSON, etc.)
- [x] Financially correct rounding
- [x] Correct comparison of numbers encoded in different precisions (e.g. 1.0 == 1.00)
- [x] Correct handling of NaN values (e.g. NaN + 1 = NaN)
- [x] Conversion to canonical representation (e.g. 1.0000 -> 1)
- [x] Conversion to fixed string representation (e.g. 1.0000 -> "1.0000")
- [x] Conversion to human-readable string representation (e.g. 1.0000 -> "1")

## Install

Run `go get github.com/jokruger/dec128`

## Requirements

This library requires Go version `>=1.23`

## Documentation

http://godoc.org/github.com/jokruger/dec128

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

    dailyRate := annualRate.Div(dec128.FromInt64(365))
    dailyRate = dailyRate.Div(dec128.FromInt64(100))

    accruedInterest := principal.Mul(dailyRate).Mul(dec128.FromInt64(days)).RoundBank(2)

    fmt.Printf("Principal: %v\n", principal.StringFixed())
    fmt.Printf("Annual Interest Rate: %v\n", annualRate.String())
    fmt.Printf("Days: %v\n", days)
    fmt.Printf("Accrued Interest: %v\n", accruedInterest.String())

    total := principal.Add(accruedInterest).RoundBank(2)
    fmt.Printf("Total after %v days: %v\n", days, total.StringFixed())
}
```

## Why not use other libraries?

There are several other libraries that provide decimal arithmetic in Go. However, most of them are either too slow, too memory-intensive, or lack the integration features required for financial applications. This library aims to provide a high-performance, low-memory, and easy-to-use alternative to existing libraries.

## Benchmarks

The following benchmarks were run on a MacBook Pro (2019) with a 2.6 GHz 6-Core Intel Core i7 processor and 16 GB of RAM (https://github.com/jokruger/go-decimal-benchmark).

```
                                 parse (ns/op)  string (ns/op)     add (ns/op)     mul (ns/op)     div (ns/op)

dec128.Dec128                           13.986          36.404          10.518           7.637          34.129
udecimal.Decimal                        22.383          44.740          11.998          11.141          40.701
alpacadecimal.Decimal                   90.959          83.291         222.275          70.552         481.113
shopspring.Decimal                     160.160         183.984         241.129          74.726         451.901
```

## Notes on Terminology

- **Precision**: The number of decimal places in a number. For example, 1.00 has a precision of 2 and 1.0000 has a precision of 4.
- **Expontent**: Same as precision, but in the context of low-level implementation details or Dec128 encoding.
- **Canonical**: The representation of a number with the minimum number of decimal places required to represent the number.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

### Attribution

This project includes code derived from:
- A project licensed under the BSD 3-Clause License (Copyright © 2025 Quang).
- A project licensed under the MIT License (Copyright © 2019 Luke Champine).

See the `LICENSE` file for full license texts.
