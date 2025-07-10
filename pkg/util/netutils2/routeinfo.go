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
	"net"

	"yunion.io/x/pkg/errors"
)

type SPrefixInfo struct {
	Prefix    net.IP
	PrefixLen uint8
}

func (p SPrefixInfo) String() string {
	return fmt.Sprintf("%s/%d", p.Prefix.String(), p.PrefixLen)
}

type SRouteInfo struct {
	SPrefixInfo

	Gateway net.IP
}

func (r SRouteInfo) String() string {
	return fmt.Sprintf("%s via %s", r.SPrefixInfo.String(), r.Gateway.String())
}

func ParseRouteInfo(route []string) (*SRouteInfo, error) {
	if len(route) < 2 {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "invalid route %#v", route)
	}
	_, prefixLen, err := net.ParseCIDR(route[0])
	if err != nil {
		return nil, errors.Wrapf(err, "net.ParseCIDR %s", route[0])
	}
	ones, _ := prefixLen.Mask.Size()
	return &SRouteInfo{
		SPrefixInfo: SPrefixInfo{
			Prefix:    prefixLen.IP,
			PrefixLen: uint8(ones),
		},
		Gateway: net.ParseIP(route[1]),
	}, nil
}
