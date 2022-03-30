# http2curl

:triangular_ruler: Convert Golang's http.Request to CURL command line

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/moul.io/http2curl)
[![License](https://img.shields.io/badge/license-Apache--2.0%20%2F%20MIT-%2397ca00.svg)](https://github.com/moul/http2curl/blob/master/COPYRIGHT)
[![GitHub release](https://img.shields.io/github/release/moul/http2curl.svg)](https://github.com/moul/http2curl/releases)
[![Docker Metrics](https://images.microbadger.com/badges/image/moul/http2curl.svg)](https://microbadger.com/images/moul/http2curl)
[![Made by Manfred Touron](https://img.shields.io/badge/made%20by-Manfred%20Touron-blue.svg?style=flat)](https://manfred.life/)

[![Go](https://github.com/moul/http2curl/workflows/Go/badge.svg)](https://github.com/moul/http2curl/actions?query=workflow%3AGo)
[![Release](https://github.com/moul/http2curl/workflows/Release/badge.svg)](https://github.com/moul/http2curl/actions?query=workflow%3ARelease)
[![PR](https://github.com/moul/http2curl/workflows/PR/badge.svg)](https://github.com/moul/http2curl/actions?query=workflow%3APR)
[![GolangCI](https://golangci.com/badges/github.com/moul/http2curl.svg)](https://golangci.com/r/github.com/moul/http2curl)
[![codecov](https://codecov.io/gh/moul/http2curl/branch/master/graph/badge.svg)](https://codecov.io/gh/moul/http2curl)
[![Go Report Card](https://goreportcard.com/badge/moul.io/http2curl)](https://goreportcard.com/report/moul.io/http2curl)
[![CodeFactor](https://www.codefactor.io/repository/github/moul/http2curl/badge)](https://www.codefactor.io/repository/github/moul/http2curl)

To do the reverse operation, check out [mholt/curl-to-go](https://github.com/mholt/curl-to-go).

## Example

```go
import (
    "http"
    "moul.io/http2curl"
)

data := bytes.NewBufferString(`{"hello":"world","answer":42}`)
req, _ := http.NewRequest("PUT", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", data)
req.Header.Set("Content-Type", "application/json")

command, _ := http2curl.GetCurlCommand(req)
fmt.Println(command)
// Output: curl -X PUT -d "{\"hello\":\"world\",\"answer\":42}" -H "Content-Type: application/json" http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu
```

## Install

```bash
go get moul.io/http2curl
```

## Usages

- https://github.com/parnurzeal/gorequest
- https://github.com/scaleway/scaleway-cli
- https://github.com/nmonterroso/cowsay-slackapp
- https://github.com/moul/as-a-service
- https://github.com/gavv/httpexpect
- https://github.com/smallnest/goreq

## License

© 2019-2021 [Manfred Touron](https://manfred.life)

Licensed under the [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0) ([`LICENSE-APACHE`](LICENSE-APACHE)) or the [MIT license](https://opensource.org/licenses/MIT) ([`LICENSE-MIT`](LICENSE-MIT)), at your option. See the [`COPYRIGHT`](COPYRIGHT) file for more details.

`SPDX-License-Identifier: (Apache-2.0 OR MIT)`
