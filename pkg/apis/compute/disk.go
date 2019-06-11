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

package compute

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type DiskCreateInput struct {
	apis.Meta

	*DiskConfig

	// prefer options
	PreferRegion string `json:"prefer_region_id"`
	PreferZone   string `json:"prefer_zone_id"`
	PreferWire   string `json:"prefer_wire_id"`
	PreferHost   string `json:"prefer_host_id"`

	Name        string `json:"name"`
	Description string `json:"description"`
	Hypervisor  string `json:"hypervisor"`
	Project     string `json:"project"`
	Domain      string `json:"domain_id"`
}

// ToServerCreateInput used by disk schedule
func (req *DiskCreateInput) ToServerCreateInput() *ServerCreateInput {
	return &ServerCreateInput{
		ServerConfigs: &ServerConfigs{
			PreferRegion: req.PreferRegion,
			PreferZone:   req.PreferZone,
			PreferWire:   req.PreferWire,
			PreferHost:   req.PreferHost,
			Hypervisor:   req.Hypervisor,
			Disks:        []*DiskConfig{req.DiskConfig},
			Project:      req.Project,
			Domain:       req.Domain,
		},
		Name: req.Name,
	}
}

func (req *ServerCreateInput) ToDiskCreateInput() *DiskCreateInput {
	return &DiskCreateInput{
		DiskConfig:   req.Disks[0],
		PreferRegion: req.PreferRegion,
		PreferHost:   req.PreferHost,
		PreferZone:   req.PreferZone,
		PreferWire:   req.PreferWire,
		Name:         req.Name,
		Project:      req.Project,
		Hypervisor:   req.Hypervisor,
	}
}
