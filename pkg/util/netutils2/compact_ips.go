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
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"
)

func ExpandCompactIps(ipstr string) ([]string, error) {
	ips := make([]string, 0)
	ipSegs := strings.Split(strings.TrimSpace(ipstr), ";")
	for _, ipSeg := range ipSegs {
		parts := strings.Split(ipSeg, ".")
		if len(parts) <= 3 {
			return nil, errors.Wrap(errors.ErrInvalidFormat, ipSeg)
		}
		hosts := strings.Split(parts[3], ",")
		for _, host := range hosts {
			if strings.Index(host, "-") > 0 {
				subhosts := strings.Split(host, "-")
				if len(subhosts) != 2 {
					return nil, errors.Wrap(errors.ErrInvalidFormat, ipSeg)
				}
				start, err := strconv.Atoi(subhosts[0])
				if err != nil {
					return nil, errors.Wrap(errors.ErrInvalidFormat, ipSeg)
				}
				end, err := strconv.Atoi(subhosts[1])
				if err != nil {
					return nil, errors.Wrap(errors.ErrInvalidFormat, ipSeg)
				}
				for i := start; i <= end; i++ {
					ips = append(ips, fmt.Sprintf("%s.%s.%s.%d", parts[0], parts[1], parts[2], i))
				}
			} else {
				ips = append(ips, fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], host))
			}
		}
	}
	return ips, nil
}
