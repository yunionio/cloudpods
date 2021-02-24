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

package models

import (
	"strings"

	"yunion.io/x/pkg/util/netutils"
)

type Subnets []*netutils.IPV4Prefix

func (nets Subnets) StrList() []string {
	r := make([]string, 0, len(nets))
	for _, p := range nets {
		r = append(r, p.String())
	}
	return r
}

func (nets Subnets) String() string {
	r := nets.StrList()
	return strings.Join(r, ",")
}

func (nets Subnets) ContainsAny(nets1 Subnets) bool {
	contains, _ := nets.ContainsAnyEx(nets1)
	return contains
}

func (nets Subnets) ContainsAnyEx(nets1 Subnets) (bool, *netutils.IPV4Prefix) {
	for _, p0 := range nets {
		for _, p1 := range nets1 {
			if p0.Equals(p1) {
				return true, p0
			}
		}
	}
	return false, nil
}
