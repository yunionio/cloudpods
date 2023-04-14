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

package service

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apimap/models/vpcagent"
	"yunion.io/x/onecloud/pkg/apimap/options"
	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	common_app "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/scheduler/service"
)

func StartService() {
	options.Init()
	opt := options.GetOptions()

	// hack: set GuestManager recordChecksum to false
	models.GuestManager.SetEnableRecordChecksum(false)

	service.StartServiceWrapper(&opt.DBOptions, &opt.CommonOptions, func(app *appsrv.Application) error {
		common_options.StartOptionManager(&opt, opt.ConfigSyncPeriodSeconds, compute_api.SERVICE_TYPE, compute_api.SERVICE_VERSION, options.OnOptionsChange)
		InitHandlers(app)
		common_app.ServeForever(app, &opt.BaseOptions)
		return nil
	})
}

func InitHandlers(app *appsrv.Application) {
	app.AddHandler2("GET", "/vpcagent", auth.Authenticate(vpcAgentHandler), nil, "get_vpcagent_topo", nil)
}

func vpcAgentHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	result, err := vpcagent.GetTopoResult(ctx, userCred, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(result))
}
