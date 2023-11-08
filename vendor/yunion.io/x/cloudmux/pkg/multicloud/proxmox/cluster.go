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
	"sort"
)

type SClusterResource struct {
	Maxcpu     int    `json:"maxcpu,omitempty"`
	Uptime     int    `json:"uptime,omitempty"`
	Template   int    `json:"template,omitempty"`
	Netin      int    `json:"netin,omitempty"`
	Mem        int    `json:"mem,omitempty"`
	Node       string `json:"node"`
	VmId       int    `json:"vmid,omitempty"`
	Maxdisk    int64  `json:"maxdisk"`
	Netout     int    `json:"netout,omitempty"`
	Diskwrite  int    `json:"diskwrite,omitempty"`
	Diskread   int    `json:"diskread,omitempty"`
	Maxmem     int64  `json:"maxmem,omitempty"`
	Disk       int    `json:"disk"`
	CPU        int    `json:"cpu,omitempty"`
	Id         string `json:"id"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Name       string `json:"name,omitempty"`
	Level      string `json:"level,omitempty"`
	Storage    string `json:"storage,omitempty"`
	Plugintype string `json:"plugintype,omitempty"`
	Content    string `json:"content,omitempty"`
	Shared     int    `json:"shared,omitempty"`
}

type SStorageResource struct {
	Id      string
	Path    string
	Node    string
	Name    string
	Shared  int
	Content string
}

type SNodeResource struct {
	Id   string
	Node string
}

type SVmResource struct {
	VmId     int
	Id       string
	Name     string
	Node     string
	NodeId   string
	Status   string
	Template bool
}

func (self *SRegion) GetClusterAllResources() ([]SClusterResource, error) {
	resources := []SClusterResource{}
	err := self.get("/cluster/resources", url.Values{}, &resources)
	return resources, err
}

func (self *SRegion) GetClusterResources(resType string) ([]SClusterResource, error) {
	resources := []SClusterResource{}
	params := url.Values{}
	if len(resType) > 0 {
		params.Set("type", resType)
	}
	err := self.get("/cluster/resources", params, &resources)
	return resources, err
}

func (self *SRegion) GetClusterNodeResources() (map[string]SNodeResource, error) {
	resources := []SClusterResource{}
	nodeResources := map[string]SNodeResource{}
	params := url.Values{}
	params.Set("type", "node")
	err := self.get("/cluster/resources", params, &resources)
	if err != nil {
		return nil, err
	}

	for _, res := range resources {
		if res.Type == "node" {
			nres := SNodeResource{
				Id:   res.Id,
				Node: res.Node,
			}

			nodeResources[nres.Id] = nres
		}
	}

	return nodeResources, nil
}

func (self *SRegion) GetClusterVmResources() (map[int]SVmResource, error) {
	resources := []SClusterResource{}
	VmResources := map[int]SVmResource{}
	err := self.get("/cluster/resources", url.Values{}, &resources)
	if err != nil {
		return nil, err
	}

	for _, res := range resources {
		if res.Type == "qemu" {
			vres := SVmResource{
				VmId:     res.VmId,
				Id:       res.Id,
				Name:     res.Name,
				Node:     res.Node,
				NodeId:   fmt.Sprintf("node/%s", res.Node),
				Status:   res.Status,
				Template: Itob(res.Template),
			}

			VmResources[vres.VmId] = vres
		}
	}

	return VmResources, nil
}

func (self *SRegion) GetClusterVmMaxId() int {
	resources := []SClusterResource{}
	idxs := []int{}
	err := self.get("/cluster/resources", url.Values{}, &resources)
	if err != nil {
		return -1
	}

	for i := range resources {
		if resources[i].Type == "qemu" {
			idxs = append(idxs, resources[i].VmId)
		}
	}

	if len(idxs) < 1 {
		return 99
	}

	sort.Ints(idxs)
	return idxs[len(idxs)-1]
}
