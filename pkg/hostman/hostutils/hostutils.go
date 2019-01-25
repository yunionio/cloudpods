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
)

type IHost interface {
	GetZone() string
	GetHostId() string
	GetMediumType() string

	IsKvmSupport() bool
	IsNestedVirtualization() bool

	PutHostOnline() error

	GetBridgeDev(bridge string) hostbridge.IBridgeDriver
	GetIsolatedDeviceManager() *isolated_device.IsolatedDeviceManager
}

func GetComputeSession(ctx context.Context) *mcclient.ClientSession {
	return auth.GetAdminSession(ctx, options.HostOptions.Region, "v2")
}

func GetImageSession(ctx context.Context, zone string) *mcclient.ClientSession {
	return auth.AdminSession(ctx, options.HostOptions.Region, zone, "internal", "v1")
}

func TaskFailed(ctx context.Context, reason string) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		modules.ComputeTasks.TaskFailed2(GetComputeSession(ctx), taskId.(string), reason)
	} else {
		log.Errorln("Reqeuest task failed missing task id, with reason(%v)", reason)
	}
}

func TaskComplete(ctx context.Context, params jsonutils.JSONObject) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		modules.ComputeTasks.TaskComplete(GetComputeSession(ctx), taskId.(string), params)
	} else {
		log.Errorln("Reqeuest task complete missing task id")
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
	return modules.Storagecachedimages.Update2(GetComputeSession(ctx),
		storagecacheId, imageId, params, query)
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
	ParamsError = fmt.Errorf("Delay task parse params error")
)

func GetWorkManager() *workmanager.SWorkManager {
	return wm
}

func DelayTask(ctx context.Context, task workmanager.DelayTaskFunc, params interface{}) {
	wm.DelayTask(ctx, task, params)
}

func DelayTaskWithoutReqctx(ctx context.Context, task workmanager.DelayTaskFunc, params interface{}) {
	wm.DelayTaskWithoutReqctx(ctx, task, params)
}

func init() {
	wm = workmanager.NewWorkManger(TaskFailed, TaskComplete)
}
