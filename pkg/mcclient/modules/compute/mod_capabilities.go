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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SCapabilityManager struct {
	modulebase.ResourceManager
}

func (this *SCapabilityManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*printutils.ListResult, error) {
	url := "/capabilities"
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			url += "?" + qs
		}
	}
	body, err := modulebase.Get(this.ResourceManager, s, url, "")
	if err != nil {
		return nil, err
	}
	result := printutils.ListResult{Data: []jsonutils.JSONObject{body}}
	return &result, nil
}

var (
	Capabilities SCapabilityManager
)

func init() {
	Capabilities = SCapabilityManager{
		ResourceManager: modules.NewComputeManager("capability", "capabilities", []string{}, []string{}),
	}
	modules.RegisterCompute(&Capabilities)
}
