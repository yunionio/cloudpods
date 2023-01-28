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

package yunionconf

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

type ParametersManager struct {
	modulebase.ResourceManager
}

var (
	Parameters ParametersManager
)

func init() {
	Parameters = ParametersManager{modules.NewYunionConfManager("parameter", "parameters",
		[]string{"id", "created_at", "update_at", "name", "value"},
		[]string{"namespace", "namespace_id", "created_by", "updated_by"},
	)}
	modules.Register(&Parameters)
}

func (m *ParametersManager) GetGlobalSettings(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.getParametersRpc(s, "global-settings", params)
}

func (m *ParametersManager) GetWidgetSettings(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.getParametersRpc(s, "widget-settings", params)
}

func (m *ParametersManager) getParametersRpc(s *mcclient.ClientSession, key string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	adminSession := auth.GetAdminSession(context.Background(), "")
	p := jsonutils.NewDict()
	p.Add(jsonutils.NewString("system"), "scope")
	p.Add(jsonutils.NewString(key), "name")
	parameters, err := m.ListInContext(adminSession, p, &identity.ServicesV3, "yunionagent")
	if err != nil {
		return nil, err
	}
	if parameters.Total == 0 {
		// if no such setting, create one
		empty := jsonutils.NewDict()
		empty.Add(jsonutils.NewString(key), "name")
		empty.Add(jsonutils.NewDict(), "value")
		empty.Add(jsonutils.NewString("yunionagent"), "service_id")
		_, err := m.Create(adminSession, empty)
		if err != nil && httputils.ErrorCode(err) != 409 {
			return nil, err
		}
		return m.getParametersRpc(s, key, params)
	}
	return parameters.Data[0], nil
}
