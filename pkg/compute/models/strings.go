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

import "fmt"

func (h *SHost) String() string {
	return fmt.Sprintf("%s(%s,%s)", h.Name, h.AccessIp, h.Id)
}

func (n *SNetwork) String() string {
	return fmt.Sprintf("%s(%s/%d)", n.Name, n.GuestIpStart, n.GuestIpMask)
}

func (netif *SNetInterface) String() string {
	return fmt.Sprintf("%s(%d)", netif.Mac, netif.VlanId)
}
