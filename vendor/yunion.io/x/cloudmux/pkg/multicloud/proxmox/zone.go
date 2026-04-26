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
	"net/url"

	"yunion.io/x/pkg/errors"
)

type SCluster struct {
	Nodes   int    `json:"nodes,omitempty"`
	Quorate int    `json:"quorate,omitempty"`
	Version int    `json:"version,omitempty"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Online  int    `json:"online,omitempty"`
	Level   string `json:"level,omitempty"`
	Nodeid  int    `json:"nodeid,omitempty"`
	Local   int    `json:"local,omitempty"`
	IP      string `json:"ip,omitempty"`
}

func (self *SProxmoxClient) GetCluster() (*SCluster, error) {
	css := []SCluster{}

	err := self.get("/cluster/status", url.Values{}, &css)
	if err != nil {
		return nil, err
	}

	for i := range css {
		if css[i].Type == "cluster" {
			return &css[i], nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "GetCluster")
}
