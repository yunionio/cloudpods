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
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func GetComputeSession(ctx context.Context) *mcclient.ClientSession {
	return auth.GetAdminSession(ctx, options.HostOptions.Region, "v2")
}

func TaskFailed(ctx context.Context, reason string) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		modules.ComputeTasks.TaskFailed2(GetComputeSession(ctx), taskId.(string), reason)
	} else {
		log.Errorln("Reqeuest task failed missing task id, with reason(%s)", reason)
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

func UpdateServerStatus(ctx context.Context, sid, status string) (jsonutils.JSONObject, error) {
	var body = jsonutils.NewDict()
	var stats = jsonutils.NewDict()
	stats.Set("status", jsonutils.NewString(status))
	body.Set("server", stats)
	return modules.Servers.PerformAction(GetComputeSession(ctx), sid, "status", body)
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

var wm *workmanager.SWorkManager

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
