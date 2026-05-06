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
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/esxi"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
)

const (
	PROXMOX_PREFIX = "disks/proxmox"
)

var defaultProxmoxHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	httperrors.NotImplementedError(ctx, w, "")
	return
}

func IdProxmoxPrefix(action string) string {
	return fmt.Sprintf("%s/%s/<disk_id>", PROXMOX_PREFIX, action)
}

func ProxmoxPrefix(action string) string {
	return fmt.Sprintf("%s/%s", PROXMOX_PREFIX, action)
}

func initProxmoxHandler(app *appsrv.Application) {
	app.AddHandler("POST", ProxmoxPrefix("deploy"), auth.Authenticate(deployProxmoxHandler))
	app.AddHandler("POST", IdProxmoxPrefix("delete"), auth.Authenticate(deleteProxmoxHandler))
	app.AddHandler("POST", IdProxmoxPrefix("resize"), auth.Authenticate(resizeProxmoxHandler))
}

func deployProxmoxHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Debugf("enter deployProxmoxHandler")
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	disk, err := body.Get("disk")
	if err != nil {
		httperrors.MissingParameterError(ctx, w, "miss disk")
		return
	}
	hostutils.DelayTask(ctx, esxi.EsxiAgent.ProxmoxStorage.AgentDeployGuest, disk)
	hostutils.ResponseOk(ctx, w)
}

func deleteProxmoxHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	diskId := params["<disk_id>"]
	disk, err := esxi.EsxiAgent.ProxmoxStorage.GetDiskById(diskId)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, errors.Wrapf(err, "GetDiskById(%s)", diskId))
		return
	}
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId == nil {
		if disk != nil {
			_, err := disk.Delete(ctx, nil)
			if err != nil {
				httperrors.GeneralServerError(ctx, w, err)
				return
			}
			hostutils.ResponseOk(ctx, w)
			return
		}
	}
	hostutils.ResponseOk(ctx, w)
	hostutils.DelayTask(ctx, disk.Delete, nil)
}

func resizeProxmoxHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	disk, diskInfo, err := proxmoxDiskAndDiskInfo(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	resizeDiskInfo := &storageman.SDiskResizeInput{
		DiskInfo: diskInfo,
	}
	resizeFunc := func(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
		input, ok := params.(*storageman.SDiskResizeInput)
		if !ok {
			return nil, hostutils.ParamsError
		}
		return disk.Resize(ctx, input)
	}
	hostutils.DelayTask(ctx, resizeFunc, resizeDiskInfo)
	hostutils.ResponseOk(ctx, w)
}

func proxmoxDiskAndDiskInfo(ctx context.Context, w http.ResponseWriter, r *http.Request) (storageman.IDisk, jsonutils.JSONObject, error) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	diskId := params["<disk_id>"]
	disk, err := esxi.EsxiAgent.ProxmoxStorage.GetDiskById(diskId)
	if err != nil {
		return nil, nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetDiskById(%s)", diskId))
	}
	diskInfo, err := body.Get("disk")
	if err != nil {
		return nil, nil, httperrors.NewMissingParameterError("miss disk")
	}
	return disk, diskInfo, nil
}
