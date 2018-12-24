package hostman

import (
	"context"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var session *mcclient.ClientSession

func GetComputeSession() *mcclient.ClientSession {
	return session
}

func init() {
	session = auth.GetAdminSession(options.HostOptions.Region, "v2")
}

func TaskFailed(ctx context.Context, reason string) error {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		modules.ComputeTasks.TaskFailed(ctx, taskId.(string), reason)
		return nil
	} else {
		log.Errorln("Reqeuest task failed missing task id, with reason(%s)", reason)
		return fmt.Errorf("Reqeuest task failed missing task id")
	}
}
