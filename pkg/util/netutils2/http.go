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

	"yunion.io/x/pkg/util/netutils"
)

func GetHttpRequestIp(r *http.Request) string {
	return netutils.GetHttpRequestIp(r)
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
