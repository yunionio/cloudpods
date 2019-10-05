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

package agent

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/workmanager"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func (agent *SBaseAgent) AddImageCacheHandler(prefix string, app *appsrv.Application) {
	hostutils.InitWorkerManager()
	app.AddHandler("POST",
		fmt.Sprintf("%s/disks/image_cache", prefix),
		auth.Authenticate(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			performImageCache(ctx, w, r, agent.CacheManager.PrefetchImageCache)
		}))
	app.AddHandler("DELETE",
		fmt.Sprintf("%s/disks/image_cache", prefix),
		auth.Authenticate(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			performImageCache(ctx, w, r, agent.CacheManager.DeleteImageCache)
		}))
}

func performImageCache(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	performTask workmanager.DelayTaskFunc,
) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)

	disk, err := body.Get("disk")
	if err != nil {
		httperrors.MissingParameterError(w, "disk")
		return
	}

	hostutils.DelayTask(ctx, performTask, disk)
	hostutils.ResponseOk(ctx, w)
}
