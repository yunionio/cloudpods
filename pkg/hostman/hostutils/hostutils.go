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

package hostutils

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/workmanager"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

type IHost interface {
	GetZoneName() string
	GetHostId() string
	GetMediumType() string
	GetMasterIp() string

	IsKvmSupport() bool
	IsNestedVirtualization() bool

	PutHostOnline() error
	StartDHCPServer()

	GetBridgeDev(bridge string) hostbridge.IBridgeDriver
	GetIsolatedDeviceManager() *isolated_device.IsolatedDeviceManager
}

func GetComputeSession(ctx context.Context) *mcclient.ClientSession {
	return auth.GetAdminSessionWithPublic(ctx, options.HostOptions.Region, "v2")
}

func GetK8sSession(ctx context.Context) *mcclient.ClientSession {
	return auth.GetAdminSessionWithPublic(ctx, options.HostOptions.Region, "")
}

func GetImageSession(ctx context.Context, zone string) *mcclient.ClientSession {
	return auth.AdminSessionWithPublic(ctx, options.HostOptions.Region, zone, "v1")
}

func TaskFailed(ctx context.Context, reason string) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		modules.ComputeTasks.TaskFailed2(GetComputeSession(ctx), taskId.(string), reason)
	} else {
		log.Errorf("Reqeuest task failed missing task id, with reason(%s)", reason)
	}
}

func TaskFailed2(ctx context.Context, reason string, params *jsonutils.JSONDict) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		modules.ComputeTasks.TaskFailed3(GetComputeSession(ctx), taskId.(string), reason, params)
	} else {
		log.Errorf("Reqeuest task failed missing task id, with reason(%s)", reason)
	}
}

func TaskComplete(ctx context.Context, params jsonutils.JSONObject) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		modules.ComputeTasks.TaskComplete(GetComputeSession(ctx), taskId.(string), params)
	} else {
		log.Errorln("Reqeuest task complete missing task id")
	}
}

func K8sTaskFailed(ctx context.Context, reason string) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		k8s.KubeTasks.TaskFailed(GetK8sSession(ctx), taskId.(string), reason)
	} else {
		log.Errorf("Reqeuest k8s task failed missing task id, with reason(%s)", reason)
	}
}

func K8sTaskComplete(ctx context.Context, params jsonutils.JSONObject) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		k8s.KubeTasks.TaskComplete(GetK8sSession(ctx), taskId.(string), params)
	} else {
		log.Errorln("Reqeuest k8s task complete missing task id")
	}
}

func GetWireOfIp(ctx context.Context, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	res, err := modules.Networks.List(GetComputeSession(ctx), params)
	if err != nil {
		return nil, err
	}

	if len(res.Data) == 1 {
		wireId, _ := res.Data[0].GetString("wire_id")
		return GetWireInfo(ctx, wireId)
	} else {
		return nil, fmt.Errorf("Fail to get network info: no networks")
	}
}

func GetWireInfo(ctx context.Context, wireId string) (jsonutils.JSONObject, error) {
	return modules.Wires.Get(GetComputeSession(ctx), wireId, nil)
}

func RemoteStoragecacheCacheImage(ctx context.Context, storagecacheId, imageId, status, spath string) (jsonutils.JSONObject, error) {
	var query = jsonutils.NewDict()
	query.Set("auto_create", jsonutils.JSONTrue)
	var params = jsonutils.NewDict()
	params.Set("status", jsonutils.NewString(status))
	params.Set("path", jsonutils.NewString(spath))
	return modules.Storagecachedimages.Update(GetComputeSession(ctx),
		storagecacheId, imageId, query, params)
}

func UpdateServerStatus(ctx context.Context, sid, status string) (jsonutils.JSONObject, error) {
	var stats = jsonutils.NewDict()
	stats.Set("status", jsonutils.NewString(status))
	return modules.Servers.PerformAction(GetComputeSession(ctx), sid, "status", stats)
}

func ResponseOk(ctx context.Context, w http.ResponseWriter) {
	Response(ctx, w, map[string]string{"result": "ok"})
}

func Response(ctx context.Context, w http.ResponseWriter, res interface{}) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		w.Header().Set("X-Request-Id", taskId.(string))
	}
	switch res.(type) {
	case string:
		appsrv.Send(w, res.(string))
	case jsonutils.JSONObject:
		appsrv.SendJSON(w, res.(jsonutils.JSONObject))
	case error:
		httperrors.GeneralServerError(w, res.(error))
	default:
		appsrv.SendStruct(w, res)
	}
}

var (
	wm          *workmanager.SWorkManager
	k8sWm       *workmanager.SWorkManager
	ParamsError = fmt.Errorf("Delay task parse params error")
)

func GetWorkManager() *workmanager.SWorkManager {
	return wm
}

func DelayTask(ctx context.Context, task workmanager.DelayTaskFunc, params interface{}) {
	wm.DelayTask(ctx, task, params)
}

func DelayKubeTask(ctx context.Context, task workmanager.DelayTaskFunc, params interface{}) {
	k8sWm.DelayTask(ctx, task, params)
}

func DelayTaskWithoutReqctx(ctx context.Context, task workmanager.DelayTaskFunc, params interface{}) {
	wm.DelayTaskWithoutReqctx(ctx, task, params)
}

func DelayTaskWithWorker(
	ctx context.Context, task workmanager.DelayTaskFunc,
	params interface{}, worker *appsrv.SWorkerManager,
) {
	wm.DelayTaskWithWorker(ctx, task, params, worker)
}

func InitWorkerManager() {
	wm = workmanager.NewWorkManger(TaskFailed, TaskComplete, options.HostOptions.DefaultRequestWorkerCount)
}

func InitK8sWorkerManager() {
	k8sWm = workmanager.NewWorkManger(K8sTaskFailed, K8sTaskComplete, options.HostOptions.DefaultRequestWorkerCount)
}

func Init() {
	InitWorkerManager()
	InitK8sWorkerManager()
}
