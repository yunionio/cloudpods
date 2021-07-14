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

package mac

import (
	"crypto/md5"
	"fmt"
)

func HashMac(in ...string) string {
	h := md5.New()
	for _, s := range in {
		h.Write([]byte(s))
	}
	sum := h.Sum(nil)
	b := sum[0]
	b &= 0xfe
	b |= 0x02
	mac := fmt.Sprintf("%02x", b)
	for _, b := range sum[1:6] {
		mac += fmt.Sprintf(":%02x", b)
	}
	return mac
}

func HashVpcHostDistgwMac(hostId string) string {
	return HashMac(hostId)
}

func HashSubnetRouterPortMac(netId string) string {
	return HashMac(netId, "rp")
}

func HashSubnetDhcpMac(netId string) string {
	return HashMac(netId, "dhcp")
}

func HashSubnetMetadataMac(netId string) string {
	return HashMac(netId, "md")
}
