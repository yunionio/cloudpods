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

package hosthandler

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/host_health"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type actionFunc func(context.Context, string, jsonutils.JSONObject) (interface{}, error)

var (
	keyWords = []string{"hosts"}
)

func AddHostHandler(prefix string, app *appsrv.Application) {
	for _, keyword := range keyWords {
		app.AddHandler("POST", fmt.Sprintf("%s/%s/shutdown-servers-on-host-down", prefix, keyword),
			auth.Authenticate(setOnHostDown))

		for action, f := range map[string]actionFunc{
			"sync":                   hostSync,
			"probe-isolated-devices": hostProbeIsolatedDevices,
		} {
			app.AddHandler("POST",
				fmt.Sprintf("%s/%s/<sid>/%s", prefix, keyword, action),
				auth.Authenticate(hostActions(f)),
			)
		}
	}
}

func setOnHostDown(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if err := host_health.SetOnHostDown(hostconsts.SHUTDOWN_SERVERS); err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	hostutils.ResponseOk(ctx, w)
}

func hostActions(f actionFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		params, _, body := appsrv.FetchEnv(ctx, w, r)
		if body == nil {
			body = jsonutils.NewDict()
		}
		var sid = params["<sid>"]
		res, err := f(ctx, sid, body)
		if err != nil {
			hostutils.Response(ctx, w, err)
		} else if res != nil {
			hostutils.Response(ctx, w, res)
		} else {
			hostutils.ResponseOk(ctx, w)
		}
	}
}

func hostSync(ctx context.Context, hostId string, body jsonutils.JSONObject) (interface{}, error) {
	return hostinfo.Instance().UpdateSyncInfo(hostId, body)
}

func hostProbeIsolatedDevices(ctx context.Context, hostId string, body jsonutils.JSONObject) (interface{}, error) {
	return hostinfo.Instance().ProbeSyncIsolatedDevices(hostId, body)
}
