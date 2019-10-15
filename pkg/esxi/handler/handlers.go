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
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/esxi"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	AGENT_PREFIX = "disks/agent"
)

func InitHandlers(app *appsrv.Application) {
	initESXIHandler(app)
}

var defaultHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// nothing
	// todo
	httperrors.NotImplementedError(w, "")
	return
}

func IdAgentPrefix(action string) string {
	return fmt.Sprintf("%s/%s/<disk_id>", AGENT_PREFIX, action)
}

func AgentPrefix(action string) string {
	return fmt.Sprintf("%s/%s", AGENT_PREFIX, action)
}

func initESXIHandler(app *appsrv.Application) {

	app.AddHandler("POST", AgentPrefix("upload"), uploadHandler)
	app.AddHandler("POST", AgentPrefix("deploy"), deployHandler)
	app.AddHandler("POST", IdAgentPrefix("delete"), deleteHandler)
	app.AddHandler("POST", IdAgentPrefix("create"), createHandler)
	app.AddHandler("POST", IdAgentPrefix("save-prepare"), savePrepareHandler)
	app.AddHandler("POST", IdAgentPrefix("resize"), resizeHandler)
	app.AddHandler("POST", IdAgentPrefix("clone"), defaultHandler)
	app.AddHandler("POST", IdAgentPrefix("fetch"), defaultHandler)
	app.AddHandler("POST", IdAgentPrefix("post-migrate"), defaultHandler)
	app.AddHandler("POST", IdAgentPrefix("snapshot"), defaultHandler)
	app.AddHandler("POST", IdAgentPrefix("reset"), defaultHandler)
	app.AddHandler("POST", IdAgentPrefix("cleanup-snapshots"), defaultHandler)
}

func uploadHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	disk, err := body.Get("disk")
	if err != nil {
		httperrors.InputParameterError(w, "miss disk")
		return
	}
	hostutils.DelayTask(ctx, esxi.EsxiAgent.AgentStorage.SaveToGlance, disk)
	hostutils.ResponseOk(ctx, w)
}

func deployHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	disk, err := body.Get("disk")
	if err != nil {
		httperrors.InputParameterError(w, "miss disk")
		return
	}
	hostutils.DelayTask(ctx, esxi.EsxiAgent.AgentStorage.AgentDeployGuest, disk)
	hostutils.ResponseOk(ctx, w)

}

func deleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	diskId := params["<disk_id>"]
	disk := esxi.EsxiAgent.AgentStorage.GetDiskById(diskId)
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		if disk != nil {
			_, err := disk.Delete(ctx, nil)
			if err != nil {
				httperrors.GeneralServerError(w, err)
				return
			}
			hostutils.ResponseOk(ctx, w)
			return
		}
		hostutils.ResponseOk(ctx, w)
		if disk != nil {
			hostutils.DelayTask(ctx, disk.Delete, nil)
		}
	}
}

func createHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	disk, diskInfo, err := diskAndDiskInfo(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
	}
	hostutils.DelayTask(ctx, esxi.EsxiAgent.AgentStorage.CreateDiskByDiskInfo,
		storageman.SDiskCreateByDiskinfo{DiskId: disk.GetId(), Disk: disk, DiskInfo: diskInfo})
	hostutils.ResponseOk(ctx, w)
}

func savePrepareHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID).(string)
	disk, diskInfo, err := diskAndDiskInfo(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
	}
	hostutils.DelayTask(ctx, disk.PrepareSaveToGlance, storageman.PrepareSaveToGlanceParams{taskId, diskInfo})
	hostutils.ResponseOk(ctx, w)
}

func resizeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	disk, diskInfo, err := diskAndDiskInfo(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
	}
	hostutils.DelayTask(ctx, disk.Resize, diskInfo)
	hostutils.ResponseOk(ctx, w)
}

/*
func fetchHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	diskId := params["<disk_id>"]
	disk := esxi.EsxiAgent.AgentStorage.GetDiskById(diskId)
	if disk != nil {
		httperrors.GeneralServerError(w, httperrors.NewDuplicateResourceError("disk '%s'", diskId))
		return
	}
	disk := esxi.EsxiAgent.AgentStorage.CreateDisk(diskId)
	diskInfo, err := body.Get("disk")
	if err != nil {
		httperrors.InputParameterError(w, "miss disk")
	}
	url, err := diskInfo.GetString("url")
	if err != nil {
		httperrors.InputParameterError(w, "miss disk.url")
	}
	hostutils.DelayTask(ctx, disk.CreateFromUrl)
	hostutils.ResponseOk(ctx, w)
}
*/

func diskAndDiskInfo(ctx context.Context, w http.ResponseWriter, r *http.Request) (storageman.IDisk, jsonutils.JSONObject, error) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	diskId := params["<disk_id>"]
	disk := esxi.EsxiAgent.AgentStorage.GetDiskById(diskId)
	if disk == nil {
		return nil, nil, httperrors.NewNotFoundError("disk '%s'", diskId)
	}
	diskInfo, err := body.Get("disk")
	if err != nil {
		return nil, nil, httperrors.NewInputParameterError("miss disk")
	}
	return disk, diskInfo, nil
}
