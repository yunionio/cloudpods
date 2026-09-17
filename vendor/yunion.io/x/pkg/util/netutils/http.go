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

package netutils

import (
	"net"
	"net/http"
	"strings"
)

// parseIPLiteral returns s as a plain address if it is one, and "" otherwise.
func parseIPLiteral(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}
	if ip := net.ParseIP(s); ip != nil {
		return ip.String()
	}
	return ""
}

// GetHttpRequestIp returns the address of the client that made the request.
//
// The forwarded headers are consulted first, since a proxy in front of the
// service rewrites them. When no proxy is in place they are whatever the
// client sent, so each candidate is parsed and only an actual address is
// returned; a value that is not an address is skipped rather than passed on
// as though it were one.
//
// The connection address is used as the last resort, split with
// net.SplitHostPort so that an IPv6 peer is returned without its brackets and
// without its port.
func GetHttpRequestIp(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); len(forwarded) > 0 {
		for _, part := range strings.Split(forwarded, ",") {
			if ip := parseIPLiteral(part); len(ip) > 0 {
				return ip
			}
		}
	}
	if ip := parseIPLiteral(r.Header.Get("X-Real-Ip")); len(ip) > 0 {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
