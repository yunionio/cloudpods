package taskman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type ITask interface {
	ScheduleRun(data jsonutils.JSONObject)
	GetParams() *jsonutils.JSONDict
	GetUserCred() mcclient.TokenCredential
	GetTaskId() string
	SetStage(stageName string, data *jsonutils.JSONDict)

	SetStageComplete(ctx context.Context, data *jsonutils.JSONDict)
	SetStageFailed(ctx context.Context, reason string)
}
