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

package azure

import (
	"strings"

	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeNodePool struct {
	multicloud.SResourceBase
	AzureTags

	cluster *SKubeCluster

	Name              string `json:"name"`
	Count             int    `json:"count"`
	VMSize            string `json:"vmSize"`
	OsDiskSizeGB      int    `json:"osDiskSizeGB"`
	OsDiskType        string `json:"osDiskType"`
	KubeletDiskType   string `json:"kubeletDiskType"`
	MaxPods           int    `json:"maxPods"`
	Type              string `json:"type"`
	EnableAutoScaling bool   `json:"enableAutoScaling"`
	ProvisioningState string `json:"provisioningState"`
	PowerState        struct {
		Code string `json:"code"`
	} `json:"powerState"`
	OrchestratorVersion string `json:"orchestratorVersion"`
	Mode                string `json:"mode"`
	OsType              string `json:"osType"`
	OsSKU               string `json:"osSKU"`
	NodeImageVersion    string `json:"nodeImageVersion"`
	EnableFIPS          bool   `json:"enableFIPS"`
}

func (self *SKubeNodePool) GetName() string {
	return self.Name
}

func (self *SKubeNodePool) GetId() string {
	return self.Name
}

func (self *SKubeNodePool) GetGlobalId() string {
	return self.Name
}

func (self *SKubeNodePool) GetStatus() string {
	return strings.ToLower(self.PowerState.Code)
}
