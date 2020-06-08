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

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/host_health"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var (
	keyWords = []string{"hosts"}
)

func AddHostHandler(prefix string, app *appsrv.Application) {
	for _, keyword := range keyWords {
		app.AddHandler("POST", fmt.Sprintf("%s/%s/shutdown-servers-on-host-down", prefix, keyword),
			auth.Authenticate(setOnHostDown))
	}
}

func setOnHostDown(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if err := host_health.SetOnHostDown(host_health.SHUTDOWN_SERVERS); err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	hostutils.ResponseOk(ctx, w)
}
