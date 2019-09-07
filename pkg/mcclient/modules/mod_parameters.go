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
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type ParametersManager struct {
	modulebase.ResourceManager
}

var (
	Parameters ParametersManager
)

func init() {
	Parameters = ParametersManager{NewYunionConfManager("parameter", "parameters",
		[]string{"id", "created_at", "update_at", "name", "value"},
		[]string{"namespace", "namespace_id", "created_by", "updated_by"},
	)}
	register(&Parameters)
}

func (this *ParametersManager) GetGlobalSettings(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	adminSession := auth.GetAdminSession(context.Background(), "", "")
	p := jsonutils.NewDict()
	p.Add(jsonutils.NewString("system"), "scope")
	p.Add(jsonutils.NewString("global-settings"), "name")
	parameters, err := this.ListInContext(adminSession, p, &ServicesV3, "yunionagent")
	if err != nil {
		return nil, err
	}

	if parameters.Total == 0 {
		return nil, httperrors.NewNotFoundError("global-settings not found")
	}

	return parameters.Data[0], nil
}
