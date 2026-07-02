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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type AIProxyUsageManager struct {
	modulebase.ResourceManager
}

var AIProxyUsage AIProxyUsageManager

var aiProxyUsagePaths = map[string]string{
	"overview":         "/usage/overview",
	"analysis":         "/usage/analysis",
	"events":           "/usage/events",
	"api-keys-options": "/usage/api-keys/options",
}

func init() {
	AIProxyUsage = AIProxyUsageManager{
		ResourceManager: modules.NewAIProxyManager("ai_proxy_usage", "ai_proxy_usage",
			[]string{"id", "path"},
			[]string{}),
	}
	modules.Register(&AIProxyUsage)
}

func aiProxyUsageURL(id string, params jsonutils.JSONObject) (string, error) {
	path, ok := aiProxyUsagePaths[id]
	if !ok {
		return "", httperrors.NewResourceNotFoundError2("ai_proxy_usage", id)
	}
	if params != nil {
		if qs := params.QueryString(); qs != "" {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return path, nil
}

func (m *AIProxyUsageManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*printutils.ListResult, error) {
	data := make([]jsonutils.JSONObject, 0, len(aiProxyUsagePaths))
	for _, id := range []string{"overview", "analysis", "events", "api-keys-options"} {
		obj := jsonutils.NewDict()
		obj.Set("id", jsonutils.NewString(id))
		obj.Set("path", jsonutils.NewString(aiProxyUsagePaths[id]))
		data = append(data, obj)
	}
	return &printutils.ListResult{Data: data, Total: len(data)}, nil
}

func (m *AIProxyUsageManager) Get(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path, err := aiProxyUsageURL(id, params)
	if err != nil {
		return nil, err
	}
	return modulebase.Get(m.ResourceManager, s, path, "")
}
