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

package bingocloud

import (
	"time"
)

type SHost struct {
	CPUHz           string    `json:"CpuHz"`
	ModelId         string    `json:"ModelId"`
	MonitorType     string    `json:"MonitorType"`
	IpmiMgrEnabled  string    `json:"IpmiMgrEnabled"`
	JoinTime        time.Time `json:"JoinTime"`
	IsBareMetal     string    `json:"isBareMetal"`
	BmcPwd          string    `json:"BmcPwd"`
	Extra           string    `json:"Extra"`
	InstanceId      string    `json:"instanceId"`
	Manufacturer    string    `json:"Manufacturer"`
	BaseBoardSerial string    `json:"BaseBoardSerial"`
	BareMetalHWInfo string    `json:"BareMetalHWInfo"`
	BmcPort         string    `json:"BmcPort"`
	CPUCores        int       `json:"CpuCores"`
	BmcIP           string    `json:"BmcIp"`
	Cabinet         string    `json:"Cabinet"`
	Memo            string    `json:"Memo"`
	HostIP          string    `json:"HostIp"`
	Memory          int       `json:"Memory"`
	HostId          string    `json:"HostId"`
	CPUKind         string    `json:"CpuKind"`
	SystemSerial    string    `json:"SystemSerial"`
	InCloud         string    `json:"InCloud"`
	Location        string    `json:"Location"`
	BmcUser         string    `json:"BmcUser"`
	PublicIP        string    `json:"PublicIp"`
	Status          string    `json:"Status"`
	Room            string    `json:"Room"`
	SSHMgrEnabled   string    `json:"SshMgrEnabled"`
	BmState         string    `json:"bmState"`
	HostName        string    `json:"HostName"`
}

func (self *SRegion) GetHosts(nextToken string) ([]SHost, string, error) {
	params := map[string]string{}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	resp, err := self.invoke("DescribePhysicalHosts", params)
	if err != nil {
		return nil, "", err
	}
	ret := struct {
		DescribePhysicalHostsResult struct {
			PhysicalHostSet []SHost
		}
		NextToken string
	}{}
	resp.Unmarshal(&ret)
	return ret.DescribePhysicalHostsResult.PhysicalHostSet, ret.NextToken, nil
}
