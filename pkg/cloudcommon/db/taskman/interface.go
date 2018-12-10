package taskman

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type ITask interface {
	ScheduleRun(data jsonutils.JSONObject)
	GetParams() *jsonutils.JSONDict
	GetUserCred() mcclient.TokenCredential
	GetTaskId() string
	SetStage(stageName string, data *jsonutils.JSONDict) error

	GetTaskRequestHeader() http.Header

	SetStageComplete(ctx context.Context, data *jsonutils.JSONDict)
	SetStageFailed(ctx context.Context, reason string)
}
