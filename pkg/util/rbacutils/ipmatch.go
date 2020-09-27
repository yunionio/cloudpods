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

package rbacutils

import (
	"strings"

	"yunion.io/x/pkg/util/netutils"
)

const (
	IP_PREFIX_SEP = ","
)

func getPrefixes(prefstr string) []netutils.IPV4Prefix {
	if len(prefstr) == 0 {
		return nil
	}
	prefs := strings.Split(prefstr, IP_PREFIX_SEP)
	ret := make([]netutils.IPV4Prefix, 0)
	for _, pref := range prefs {
		p, err := netutils.NewIPV4Prefix(pref)
		if err != nil {
			continue
		}
		ret = append(ret, p)
	}
	return ret
}

func MatchIPStrings(prefstr string, ipstr string) bool {
	prefs := getPrefixes(prefstr)
	return matchIP(prefs, ipstr)
}

func matchIP(prefs []netutils.IPV4Prefix, ipstr string) bool {
	if len(prefs) == 0 {
		return true
	}
	ip, err := netutils.NewIPV4Addr(ipstr)
	if err != nil {
		return false
	}
	for _, pref := range prefs {
		if pref.Contains(ip) {
			return true
		}
	}
	return false
}
