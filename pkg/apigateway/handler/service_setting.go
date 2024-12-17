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

package handler

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

var settingNames = map[string][]string{
	"identity":   []string{"no_action_logout_seconds"},
	"yunionapi":  []string{"enable_organization", "totp_issuer"},
	"image":      []string{"enable_pending_delete"},
	"compute_v2": []string{"enable_pending_delete"},
	"meter":      []string{"cost_conversion_available", "enable_prediction", "share_resource_type"},
	"common":     []string{"api_server", "enable_quota_check", "enable_watermark", "enable_cloud_shell"},
}

func (mh *MiscHandler) getServiceSettings(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	s := auth.GetAdminSession(ctx, FetchRegion(req))
	types := []string{}
	for typ := range settingNames {
		types = append(types, typ)
	}
	params := map[string]interface{}{
		"limit": 20,
		"scope": "system",
		"type":  types,
	}
	resp, err := identity.ServicesV3.List(s, jsonutils.Marshal(params))
	if err != nil {
		e := httperrors.NewInternalServerError(err.Error())
		httperrors.JsonClientError(ctx, w, e)
		return
	}
	services := []struct {
		Id   string
		Type string
	}{}
	err = jsonutils.Update(&services, resp.Data)
	if err != nil {
		e := httperrors.NewInternalServerError(err.Error())
		httperrors.JsonClientError(ctx, w, e)
		return
	}
	result := map[string]map[string]interface{}{}
	for _, service := range services {
		result[service.Type] = map[string]interface{}{}
		data, err := identity.ServicesV3.GetSpecific(s, service.Id, "config", nil)
		if err != nil {
			e := httperrors.NewInternalServerError(err.Error())
			httperrors.JsonClientError(ctx, w, e)
			return
		}
		names, _ := settingNames[service.Type]
		for _, name := range names {
			v, _ := data.Get("config", "default", name)
			result[service.Type][name] = v
		}
	}
	appsrv.SendJSON(w, jsonutils.Marshal(result))
}
