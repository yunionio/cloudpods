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
	Id     string
	Path   string
	Node   string
	Name   string
	Shared int
}

type SNodeResource struct {
	Id   string
	Node string
}

type SVmResource struct {
	VmId   int
	Id     string
	Name   string
	Node   string
	Status string
}

func (self *SRegion) GetClusterAllResources() ([]SClusterResource, error) {
	resources := []SClusterResource{}
	err := self.get("/cluster/resources", url.Values{}, &resources)
	return resources, err
}

func (self *SRegion) GetClusterStoragesResources() ([]SStorageResource, error) {
	resources := []SClusterResource{}
	storageResources := []SStorageResource{}
	err := self.get("/cluster/resources", url.Values{}, &resources)

	if err != nil {
		return nil, err
	}

	for _, rc := range resources {
		if rc.Type == "storage" {
			src := SStorageResource{
				Id:     rc.Id,
				Path:   fmt.Sprintf("/nodes/%s/storage/%s", rc.Node, rc.Storage),
				Node:   rc.Node,
				Name:   rc.Storage,
				Shared: rc.Shared,
			}

			storageResources = append(storageResources, src)
		}
	}

	return storageResources, nil
}

func (self *SRegion) GetClusterNodeResources() ([]SNodeResource, error) {
	resources := []SClusterResource{}
	nodeResources := []SNodeResource{}
	err := self.get("/cluster/resources", url.Values{}, &resources)

	if err != nil {
		return nil, err
	}

	for _, rc := range resources {
		if rc.Type == "node" {
			nrc := SNodeResource{
				Id:   rc.Id,
				Node: rc.Node,
			}

			nodeResources = append(nodeResources, nrc)
		}
	}

	return nodeResources, nil
}

func (self *SRegion) GetClusterVmResources() ([]SVmResource, error) {
	resources := []SClusterResource{}
	VmResources := []SVmResource{}
	err := self.get("/cluster/resources", url.Values{}, &resources)

	if err != nil {
		return nil, err
	}

	for _, rc := range resources {
		if rc.Type == "qemu" {
			vrc := SVmResource{
				VmId:   rc.VmId,
				Id:     rc.Id,
				Name:   rc.Name,
				Node:   rc.Node,
				Status: rc.Status,
			}

			VmResources = append(VmResources, vrc)
		}
	}

	return VmResources, nil
}
