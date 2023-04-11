# EFP (Excel Formula Parser)

[![Build Status](https://github.com/xuri/efp/workflows/Go/badge.svg)](https://github.com/xuri/efp/actions?workflow=Go)
[![Code Coverage](https://codecov.io/gh/xuri/efp/branch/master/graph/badge.svg)](https://codecov.io/gh/xuri/efp)
[![Go Report Card](https://goreportcard.com/badge/github.com/xuri/efp)](https://goreportcard.com/report/github.com/xuri/efp)
[![go.dev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/xuri/efp)
[![Licenses](https://img.shields.io/badge/license-bsd-orange.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fxuri%2Fefp.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fxuri%2Fefp?ref=badge_shield)

Using EFP (Excel Formula Parser) you can get an Abstract Syntax Tree (AST) from Excel formula.

## Installation

```bash
go get github.com/xuri/efp
```

## Example

```go
package main

import "github.com/xuri/efp"

func main() {
    ps := efp.ExcelParser()
    ps.Parse("=SUM(A3+B9*2)/2")
    println(ps.PrettyPrint())
}
```

Get AST

```text
SUM <Function> <Start>
    A3 <Operand> <Range>
    + <OperatorInfix> <Math>
    B9 <Operand> <Range>
    * <OperatorInfix> <Math>
    2 <Operand> <Number>
 <Function> <Stop>
/ <OperatorInfix> <Math>
2 <Operand> <Number>
```

## Contributing

Contributions are welcome! Open a pull request to fix a bug, or open an issue to discuss a new feature or change.

## Credits

EFP (Excel Formula Parser) is a Golang port of [E. W. Bachtal's](https://ewbi.blogs.com/develops/2004/12/excel_formula_p.html) Excel formula parser.

## Licenses

This program is under the terms of the BSD 3-Clause License. See [https://opensource.org/licenses/BSD-3-Clause](https://opensource.org/licenses/BSD-3-Clause).

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fxuri%2Fefp.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fxuri%2Fefp?ref=badge_large)
