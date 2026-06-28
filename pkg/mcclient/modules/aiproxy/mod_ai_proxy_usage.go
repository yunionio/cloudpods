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

package aiproxy

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type AIProxyUsageManager struct {
	modulebase.ResourceManager
}

var AIProxyUsage AIProxyUsageManager

func init() {
	AIProxyUsage = AIProxyUsageManager{
		ResourceManager: modules.NewAIProxyManager("ai_proxy_usage", "ai_proxy_usage",
			[]string{"id", "path"},
			[]string{}),
	}
	modules.Register(&AIProxyUsage)
}

func (m AIProxyUsageManager) Get(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if id != "events" {
		return m.ResourceManager.Get(s, id, params)
	}
	path := "/" + m.ContextPath(nil) + "/" + id
	if params != nil {
		if qs := params.QueryString(); qs != "" {
			path += "?" + qs
		}
	}
	return modulebase.Get(m.ResourceManager, s, path, "")
}

func (m AIProxyUsageManager) GetSpecific(s *mcclient.ClientSession, id string, spec string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if id != "events" || spec != "distinct-field" {
		return m.ResourceManager.GetSpecific(s, id, spec, params)
	}
	path := "/" + m.ContextPath(nil) + "/" + id + "/" + spec
	if params != nil {
		if qs := params.QueryString(); qs != "" {
			path += "?" + qs
		}
	}
	return modulebase.Get(m.ResourceManager, s, path, "")
}
