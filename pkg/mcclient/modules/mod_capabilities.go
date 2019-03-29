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

package modules

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCapabilityManager struct {
	ResourceManager
}

func (this *SCapabilityManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error) {
	body, err := this._get(s, "/capabilities", "")
	if err != nil {
		return nil, err
	}
	result := ListResult{Data: []jsonutils.JSONObject{body}}
	return &result, nil
}

var (
	Capabilities SCapabilityManager
)

func init() {
	Capabilities = SCapabilityManager{
		ResourceManager: NewComputeManager("capability", "capabilities", []string{}, []string{}),
	}
	registerComputeV2(&Capabilities)
}
