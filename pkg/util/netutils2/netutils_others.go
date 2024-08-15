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

//go:build !linux
// +build !linux

package netutils2

import (
	"net"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/iproute2"
)

func (n *SNetInterface) GetAddresses() [][]string {
	return nil
}

func (n *SNetInterface) GetRouteSpecs() []iproute2.RouteSpec {
	return []iproute2.RouteSpec{}
}

func (n *SNetInterface) ClearAddrs() error {
	return nil
}

func DefaultSrcIpDev() (srcIp net.IP, ifname string, err error) {
	err = errors.ErrNotImplemented
	return
}
