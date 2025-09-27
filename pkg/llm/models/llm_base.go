package models

import (
	"context"

	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/drivers"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
)

type ILLMBaseProvider interface {
	GetLLMBase() *SLLMBase
}

type SLLMBase struct {
	db.SVirtualResourceBase

	// GuestId is also the pod id
	GuestId string `width:"36" charset:"ascii" list:"user" index:"true" create:"optional"`
}

func (base *SLLMBase) GetPodDriver() IPodDriver {
	return &drivers.SPodDriver{}
}

func (base *SLLMBase) SetGuestId(gstId string) error {
	if _, err := db.Update(base, func() error {
		base.GuestId = gstId
		return nil
	}); nil != err {
		return errors.Wrapf(err, "update dify guest with %s", gstId)
	}

	return nil
}

func (base *SLLMBase) CreatePodByPolling(ctx context.Context, userCred mcclient.TokenCredential, input *computeapi.ServerCreateInput) error {
	gstId, err := base.GetPodDriver().RequestCreatePodWithPolling(ctx, userCred, input)
	if err != nil {
		return err
	}
	return base.SetGuestId(gstId)
}

func (base *SLLMBase) StartCreatePodTask(ctx context.Context, userCred mcclient.TokenCredential, input *computeapi.ServerCreateInput, parentTaskId string) error {
	// set status to creating pod
	base.SetStatus(ctx, userCred, api.LLM_STATUS_CREATING_POD, "")
	task, err := taskman.TaskManager.NewTask(ctx, "BaseCreatePodTask", base, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New BaseCreatePodTask")
	}
	return task.ScheduleRun(nil)
}
