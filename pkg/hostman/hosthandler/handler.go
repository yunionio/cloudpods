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
	"os"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

type actionFunc func(context.Context, string, jsonutils.JSONObject) (interface{}, error)

var (
	keyWords = []string{"hosts"}
)

func AddHostHandler(prefix string, app *appsrv.Application) {
	for _, keyword := range keyWords {
		for action, f := range map[string]actionFunc{
			"sync":                          hostSync,
			"probe-isolated-devices":        hostProbeIsolatedDevices,
			"shutdown-servers-on-host-down": setOnHostDown,
			"restart-host-agent":            hostRestart,
		} {
			app.AddHandler("POST",
				fmt.Sprintf("%s/%s/<sid>/%s", prefix, keyword, action),
				auth.Authenticate(hostActions(f)),
			)
		}
	}
}

func setOnHostDown(ctx context.Context, hostId string, body jsonutils.JSONObject) (interface{}, error) {
	if !body.Contains("shutdown_servers") {
		return nil, httperrors.NewMissingParameterError("shutdown_servers")
	}

	if jsonutils.QueryBoolean(body, "shutdown_servers", false) {
		hostinfo.Instance().SetOnHostDown(hostconsts.SHUTDOWN_SERVERS)
	} else {
		hostinfo.Instance().SetOnHostDown("")
	}
	return nil, nil
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

func hostRestart(ctx context.Context, hostId string, body jsonutils.JSONObject) (interface{}, error) {
	log.Infof("Received host restart request, going to restart host-agent.")
	timeutils2.AddTimeout(time.Second*3, func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
		hostinfo.Stop()
		hostutils.GetWorkManager().Stop()
		os.Exit(1)
	})
	return nil, nil
}

func hostProbeIsolatedDevices(ctx context.Context, hostId string, body jsonutils.JSONObject) (interface{}, error) {
	_, err := hostinfo.Instance().ProbeSyncIsolatedDevices(hostId, body)
	return nil, err
}
