// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package netutils2

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func GetHttpRequestIp(r *http.Request) string {
	ipStr := r.Header.Get("X-Forwarded-For")
	if len(ipStr) > 0 {
		ipList := strings.Split(ipStr, ",")
		if len(ipList) > 0 {
			return ipList[0]
		}
	}
	ipStr = r.Header.Get("X-Real-Ip")
	if len(ipStr) > 0 {
		return ipStr
	}
	ipStr = r.RemoteAddr
	colonPos := strings.Index(ipStr, ":")
	if colonPos > 0 {
		ipStr = ipStr[:colonPos]
	}
	return ipStr
}

func ParseIpFromUrl(u string) string {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	host := parsedUrl.Hostname()
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	return ip.String()
}
