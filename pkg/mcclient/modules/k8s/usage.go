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

package k8s

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	Usages *SUsageManager
)

func init() {
	Usages = NewUsageManager()
}

type SUsageManager struct {
	*ResourceManager
}

func NewUsageManager() *SUsageManager {
	return &SUsageManager{
		ResourceManager: NewResourceManager("usage", "usages", NewColumns(), NewColumns()),
	}
}

func (m *SUsageManager) GetUsage(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := "/usages"
	if params != nil {
		query := params.QueryString()
		if len(query) > 0 {
			url = fmt.Sprintf("%s?%s", url, query)
		}
	}
	return modulebase.Get(m.GetBaseManager(), s, url, "usage")
}
