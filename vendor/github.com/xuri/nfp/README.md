# NFP (Number Format Parser)

[![Build Status](https://github.com/xuri/nfp/workflows/Go/badge.svg)](https://github.com/xuri/nfp/actions?workflow=Go)
[![Code Coverage](https://codecov.io/gh/xuri/nfp/branch/main/graph/badge.svg)](https://codecov.io/gh/xuri/nfp)
[![Go Report Card](https://goreportcard.com/badge/github.com/xuri/nfp)](https://goreportcard.com/report/github.com/xuri/nfp)
[![go.dev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/xuri/nfp)
[![Licenses](https://img.shields.io/badge/license-bsd-orange.svg)](https://opensource.org/licenses/BSD-3-Clause)

Using NFP (Number Format Parser) you can get an Abstract Syntax Tree (AST) from Excel number format expression.

## Installation

```bash
go get github.com/xuri/nfp
```

## Example

```go
package main

import "github.com/xuri/nfp"

func main() {
    ps := nfp.NumberFormatParser()
    tokens := ps.Parse("_(* #,##0.00_);_(* (#,##0.00);_(* \"-\"??_);_(@_)")
    println(p.PrettyPrint())
}
```

Get AST

```text
<Positive>
      <RepeatsChar>
    # <HashPlaceHolder>
    <ThousandsSeparator>
    ## <HashPlaceHolder>
    0 <ZeroPlaceHolder>
    . <DecimalPoint>
    00 <ZeroPlaceHolder>
<Negative>
      <RepeatsChar>
    ( <Literal>
    # <HashPlaceHolder>
    , <ThousandsSeparator>
    ## <HashPlaceHolder>
    0 <ZeroPlaceHolder>
    . <DecimalPoint>
    00 <ZeroPlaceHolder>
    ) <Literal>
<Zero>
      <RepeatsChar>
    - <Literal>
    ?? <DigitalPlaceHolder>
<Text>
    @ <TextPlaceHolder>
```

## Contributing

Contributions are welcome! Open a pull request to fix a bug, or open an issue to discuss a new feature or change.

## Licenses

This program is under the terms of the BSD 3-Clause License. See [https://opensource.org/licenses/BSD-3-Clause](https://opensource.org/licenses/BSD-3-Clause).
