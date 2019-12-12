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

package zstack

import "net/url"

type ImageServers []SImageServer

type SImageServer struct {
	ZStackBasic

	Hostname          string   `json:"hostname"`
	Username          string   `json:"username"`
	SSHPort           int      `json:"sshPort"`
	URL               string   `json:"url"`
	TotalCapacity     int      `json:"totalCapacity"`
	AvailableCapacity int      `json:"availableCapacity"`
	Type              string   `json:"type"`
	State             string   `json:"state"`
	Status            string   `json:"status"`
	AttachedZoneUUIDs []string `json:"attachedZoneUuids"`
	ZStackTime
}

func (v ImageServers) Len() int {
	return len(v)
}

func (v ImageServers) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v ImageServers) Less(i, j int) bool {
	if v[i].AvailableCapacity < v[j].AvailableCapacity {
		return false
	}
	return true
}

func (region *SRegion) GetImageServers(zoneId, serverId string) ([]SImageServer, error) {
	servers := []SImageServer{}
	params := url.Values{}
	params.Add("q", "state=Enabled")
	params.Add("q", "status=Connected")
	if SkipEsxi {
		params.Add("q", "type!=VCenter")
	}
	if len(zoneId) > 0 {
		params.Add("q", "zone.uuid="+zoneId)
	}
	if len(serverId) > 0 {
		params.Add("q", "uuid="+serverId)
	}
	return servers, region.client.listAll("backup-storage", params, &servers)
}
