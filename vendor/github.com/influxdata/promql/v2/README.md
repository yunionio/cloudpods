# PromQL Module

[![GoDoc](https://godoc.org/github.com/influxdata/promql?status.svg)](http://godoc.org/github.com/influxdata/promql)

The PromQL module in this package is a pruned version of the native Prometheus [promql](https://github.com/prometheus/prometheus/tree/master/promql) package, but extracted into a single module with fewer dependencies.

This module removes the promql engine and keeps anything related to lexing and parsing the PromQL language.

Each version of this module matches with the equivalent Prometheus version.

## Example

The PromQL module can be used to parse an expression.

```go
package main

import (
    "fmt"

    "github.com/influxdata/promql/v2"
)

var myExpression = `http_requests_total{job="prometheus"}[5m]`

func main() {
    expr, err := promql.ParseExpr(myExpression)
    if err != nil {
        panic(err)
    }

    fmt.Println(promql.Tree(expr))
}
```

## Contributing

Any changes to PromQL should not be made to this module as it is a pruned mirror of the original prometheus package. Changes should be submitted to the upstream [Prometheus](https://github.com/prometheus/prometheus) project and they will find their way into this repository when a new release happens.

The only changes that will be accepted into this repository are ones that fix a problem in the mirroring process, deviations from the upstream Prometheus, or operational changes to fulfill the primary purpose of this repository as a mirror of the promql package.
