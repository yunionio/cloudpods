package hostman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/hostman/options"
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

func UpdateServerStatus(ctx context.Context, sid, status string) (jsonutils.JSONObject, error) {
	var body = jsonutils.NewDict()
	var stus = jsonutils.NewDict()
	stus.Set("status", jsonutils.NewString(status))
	body.Set("server", stus)
	return modules.Servers.PerformAction(GetComputeSession(ctx), sid, "status", body)
}
