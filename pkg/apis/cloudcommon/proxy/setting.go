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

package proxy

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"
)

type ProxySetting struct {
	HttpProxy  string
	HttpsProxy string
	NoProxy    string
}

func (v *ProxySetting) Sanitize() error {
	v.HttpProxy = strings.TrimSpace(v.HttpProxy)
	if u, err := parseProxy(v.HttpProxy); err != nil {
		return errors.Wrap(err, "invalid https_proxy url")
	} else if u != nil {
		v.HttpProxy = u.String()
	}

	v.HttpsProxy = strings.TrimSpace(v.HttpsProxy)
	if u, err := parseProxy(v.HttpsProxy); err != nil {
		return errors.Wrap(err, "invalid http_proxy url")
	} else if u != nil {
		v.HttpsProxy = u.String()
	}

	if noProxy, err := parseNoProxy(v.NoProxy); err == nil {
		v.NoProxy = strings.Join(noProxy, ",")
	} else {
		return errors.Wrap(err, "invalid no_proxy")
	}
	return nil
}

func parseProxy(proxy string) (*url.URL, error) {
	if proxy == "" {
		return nil, nil
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil ||
		(proxyURL.Scheme != "http" &&
			proxyURL.Scheme != "https" &&
			proxyURL.Scheme != "socks5") {
		// proxy was bogus. Try prepending "http://" to it and
		// see if that parses correctly. If not, we fall
		// through and complain about the original one.
		if proxyURL, err := url.Parse("http://" + proxy); err == nil {
			return proxyURL, nil
		}
	}
	if err != nil {
		return nil, errors.Wrapf(err, "invalid proxy address %q", proxy)
	}
	return proxyURL, nil
}

func parseNoProxy(noProxy string) ([]string, error) {
	var r []string
	for _, p := range strings.Split(noProxy, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if len(p) == 0 {
			continue
		}

		if p == "*" {
			r = append(r, p)
			// let it go
			//return r, nil
		}

		// IPv4/CIDR, IPv6/CIDR
		if _, _, err := net.ParseCIDR(p); err == nil {
			r = append(r, p)
			continue
		}

		// IPv4:port, [IPv6]:port
		phost, _, err := net.SplitHostPort(p)
		if err == nil {
			if len(phost) == 0 {
				// There is no host part, likely the entry is malformed; ignore.
				return nil, fmt.Errorf("host part must not be empty: %s", p)
			}
			if phost[0] == '[' && phost[len(phost)-1] == ']' {
				phost = phost[1 : len(phost)-1]
			}
		} else {
			phost = p
		}
		// IPv4, IPv6
		if pip := net.ParseIP(phost); pip != nil {
			r = append(r, p)
			continue
		}

		if len(phost) == 0 {
			// There is no host part, likely the entry is malformed; ignore.
			continue
		}

		r = append(r, p)
	}
	return r, nil
}
