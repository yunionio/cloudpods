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
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type LoadbalancerBackendDetails struct {
	apis.StatusStandaloneResourceDetails
	LoadbalancerBackendGroupResourceInfo

	SLoadbalancerBackend

	ProjectId string `json:"tenant_id"`
}

type LoadbalancerBackendListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	LoadbalancerBackendGroupFilterListInput

	// filter by backend server
	Backend string `json:"backend"`

	// filter by backend group
	// BackendGroup string `json:"backend_group"`

	BackendType []string `json:"backend_type"`
	BackendRole []string `json:"backend_role"`
	Address     []string `json:"address"`

	SendProxy []string `json:"send_proxy"`
	Ssl       []string `json:"ssl"`
}

type LoadbalancerBackendCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	//swagger: ignore
	BackendGroup   string `json:"backend_group" yunion-deprecated-by:"backend_group_id"`
	BackendGroupId string `json:"backend_group_id"`

	//swagger: ignore
	Backend   string `json:"backend" yunion-deprecated-by:"backend_id"`
	BackendId string `json:"backend_id"`

	BackendType string `json:"backend_type"`
	Weight      int    `json:"weight"`
	Port        int    `json:"port"`
	SendProxy   string `json:"send_proxy"`
	Address     string `json:"address"`
	Ssl         string `json:"ssl"`
}

type LoadbalancerBackendUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput

	Weight    *int    `json:"weight"`
	Port      *int    `json:"port"`
	SendPorxy *string `json:"send_proxy"`
	Ssl       *string `json:"ssl"`
}

func (self *LoadbalancerBackendUpdateInput) Validate() error {
	if self.Weight != nil {
		if *self.Weight < 1 || *self.Weight > 100 {
			return httperrors.NewOutOfRangeError("weight out of range 1-100")
		}
	}
	if self.Port != nil {
		if *self.Port < 1 || *self.Port > 65535 {
			return httperrors.NewOutOfRangeError("port out of range 1-65535")
		}
	}
	if self.SendPorxy != nil {
		if !utils.IsInStringArray(*self.SendPorxy, LB_SENDPROXY_CHOICES) {
			return httperrors.NewInputParameterError("invalid send_proxy %v", self.SendPorxy)
		}
	}
	if self.Ssl != nil {
		if !utils.IsInStringArray(*self.Ssl, []string{LB_BOOL_ON, LB_BOOL_OFF}) {
			return httperrors.NewInputParameterError("invalid ssl %v", self.Ssl)
		}
	}
	return nil
}
