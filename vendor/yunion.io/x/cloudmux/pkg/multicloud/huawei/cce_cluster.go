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

package huawei

import (
	"fmt"
	"net/url"
)

type SCceCluster struct {
	Metadata struct {
		Uid  string
		Name string
	}
}

func (self *SRegion) ListCceClusters() ([]SCceCluster, error) {
	query := url.Values{}
	resp, err := self.list(SERVICE_CCE, "clusters", query)
	if err != nil {
		return nil, err
	}
	ret := []SCceCluster{}
	return ret, resp.Unmarshal(&ret, "items")
}

type SCceNode struct {
	Metadata struct {
		Uid  string
		Name string
	}
}

func (self *SRegion) ListCceNodes(cluster string) ([]SCceNode, error) {
	query := url.Values{}
	res := fmt.Sprintf("clusters/%s/nodes", cluster)
	resp, err := self.list(SERVICE_CCE, res, query)
	if err != nil {
		return nil, err
	}
	ret := []SCceNode{}
	return ret, resp.Unmarshal(&ret, "items")
}
