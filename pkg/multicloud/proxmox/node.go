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

package proxmox

import (
	"fmt"
	"net/url"
)

type SNode struct {
	CPU            string `json:"CPU"`
	Level          string `json:"level"`
	Node           string `json:"node"`
	SslFingerprint string `json:"ssl_fingerprint"`
	Status         string `json:"status"`
	Uptime         int    `json:"uptime"`
	Maxcpu         int    `json:"maxcpu"`
	Maxmem         int    `json:"maxmem"`
	Mem            int    `json:"mem"`
	Type           string `json:"type"`
	Id             string `json:"id"`
}

func (self *SRegion) GetNodes() ([]SNode, error) {
	nodes := []SNode{}
	err := self.get("nodes", url.Values{}, &nodes)
	return nodes, err
}

func (self *SRegion) GetNode(id string) (*SNode, error) {
	ret := &SNode{}
	res := fmt.Sprintf("/nodes/%s", id)
	return ret, self.get(res, url.Values{}, ret)
}
