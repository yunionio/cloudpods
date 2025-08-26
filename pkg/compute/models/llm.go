// Copyright 2025 Yunion
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

package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/stringutils"
)

type SLLMManager struct {
	db.SVirtualResourceBaseManager
}

var LLMManager *SLLMManager

func init() {
	LLMManager = &SLLMManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLLM{},
			"llms_tbl",
			"llm",
			"llms",
		),
	}
	LLMManager.SetVirtualObject(LLMManager)
	LLMManager.SetAlias("llm", "llms")
	LLMManager.NameRequireAscii = false
	notifyclient.AddNotifyDBHookResources(LLMManager.KeywordPlural(), LLMManager.AliasPlural())
}

func (manager *SLLMManager) DeleteByContainerId(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) error {
	q := manager.Query().Equals("container_id", ctrId)
	llms := make([]SLLM, 0)
	if err := db.FetchModelObjects(manager, q, &llms); err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}

	// log.Infoln("get in delete by container id", llms)

	for _, llm := range llms {
		if err := llm.RealDelete(ctx, userCred); nil != err {
			return err
		}
	}
	return nil
}

type SLLM struct {
	db.SVirtualResourceBase

	// GuestId is also the pod id
	GuestId     string `width:"36" charset:"ascii" default:"default" list:"user" index:"true" create:"optional"`
	ContainerId string `width:"36" charset:"ascii" default:"default" list:"user" index:"true" create:"optional"`
	Model       string `width:"64" charset:"ascii" default:"qwen3:1.7b" list:"user" update:"user" create:"required"`
}

func (llm *SLLM) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// manually generate llm Id
	llm.Id = stringutils.UUID4()

	// init task
	task, err := taskman.TaskManager.NewTask(ctx, "LLMPullModelTask", llm, userCred, data.(*jsonutils.JSONDict), "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}

	// init params and set parend task id
	params := jsonutils.NewDict()
	params.Update(data)
	params.Remove("model")
	params.Set("__parent_task_id", jsonutils.NewString(task.GetId()))

	// set enviroment OLLAMA_HOST=0.0.0.0:11434
	// envs := []*apis.ContainerKeyValue{&apis.ContainerKeyValue{
	// 	Key:   api.LLM_OLLAMA_EXPORT_ENV_KEY,
	// 	Value: api.LLM_OLLAMA_EXPORT_ENV_VALUE,
	// }}
	// params.Set("envs", envs)

	// use data to create a pod
	handler := db.NewModelHandler(GuestManager)
	server, err := handler.Create(ctx, jsonutils.NewDict(), params, nil)
	if err != nil {
		return errors.Wrap(err, "CreateServer")
	}

	// set GuestID
	guestId, err := server.GetString("id")
	if err != nil {
		return errors.Wrap(err, "GetGuestId")
	}
	llm.GuestId = guestId
	return llm.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (llm *SLLM) CheckModelExists(ctx context.Context, userCred mcclient.TokenCredential, ctr *SContainer, model string) (bool, error) {
	listInput := &api.ContainerExecSyncInput{
		Command: []string{api.LLM_OLLAMA_EXEC_PATH, api.LLM_OLLAMA_LIST_ACTION},
		Timeout: 0,
	}

	list, err := ctr.PerformExecSync(ctx, userCred, nil, listInput)
	if nil != err {
		errors.Wrap(err, "Ollama list")
		return false, err
	}

	listResp := new(api.ContainerExecSyncResponse)
	if err = list.Unmarshal(listResp); err != nil {
		errors.Wrap(err, "Unmarshal list response")
		return false, err
	}

	// Check if the model exists in the output
	// ollama list output format: NAME  ID  SIZE  MODIFIED
	line := strings.TrimSpace(listResp.Stdout)

	fields := strings.Fields(line)

	for i := 4; i < len(fields); i += 4 {
		// log.Infoln("check model list, fields:", fields[i], model)
		if fields[i] == model {
			return true, nil
		}
	}

	return false, nil
}

func (llm *SLLM) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !sets.NewString(api.CONTAINER_STATUS_EXITED, api.CONTAINER_STATUS_START_FAILED).Has(llm.Status) {
		return nil, httperrors.NewInvalidStatusError("Can't start llm in status %s", llm.Status)
	}
	return nil, llm.StartStartTask(ctx, userCred, "")
}

func (llm *SLLM) StartStartTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	llm.SetStatus(ctx, userCred, api.CONTAINER_STATUS_STARTING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LLMStartTask", llm, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (llm *SLLM) RunModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (llm *SLLM) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return llm.SVirtualResourceBase.Delete(ctx, userCred)
}

func llmFetchContainerById(ctrMngr *SContainerManager, llm *SLLM) (*SContainer, error) {
	ctr := &SContainer{}
	if err := ctrMngr.Query().Equals("id", llm.ContainerId).First(ctr); err != nil {
		return nil, errors.Wrap(err, "Query.First Container")
	}
	return ctr, nil
}

func llmGetOllamaContainer(ctrMngr *SContainerManager, llm *SLLM) (*SContainer, error) {
	ctrs, err := ctrMngr.GetContainersByPod(llm.GuestId)
	if err != nil {
		return nil, err
	}
	// found ollama
	for _, ctr := range ctrs {
		if strings.Contains(ctr.Spec.Image, "ollama") {
			// update ContainerId
			llm.SetContainer(ctr.GetId())
			return &ctr, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "ollama container for guest %s not found", llm.GuestId)
}

func (llm *SLLM) GetContainer() (*SContainer, error) {
	ctrMngr := GetContainerManager()
	// log.Infoln("get container", llm.ContainerId)
	if llm.ContainerId == "default" {
		return llmGetOllamaContainer(ctrMngr, llm)
	} else {
		return llmFetchContainerById(ctrMngr, llm)
	}
}

func (llm *SLLM) SetContainer(ctrId string) error {

	if _, err := db.Update(llm, func() error {
		llm.ContainerId = ctrId
		return nil
	}); nil != err {
		return errors.Wrapf(err, "update llm container with %s", ctrId)
	}
	// log.Infoln("update container id", ret.String())
	return nil
}

// func (manager *SLLMManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
// }

func (llm *SLLM) PullModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	// get container
	ctr, err := llm.GetContainer()
	if nil != err {
		return err
	}
	// pull model
	pullInput := &api.ContainerExecSyncInput{
		Command: []string{
			"/bin/sh", "-c",
			fmt.Sprintf("nohup %s %s %s > /dev/null 2>&1 &",
				api.LLM_OLLAMA_EXEC_PATH,
				api.LLM_OLLAMA_PULL_ACTION,
				llm.Model),
		},
		Timeout: 5,
	}

	_, err = ctr.PerformExecSync(ctx, userCred, nil, pullInput)
	if nil != err {
		return err
	}

	// wait until model pull complete
	// check model exists every 10 seconds until it returns true
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			exists, err := llm.CheckModelExists(ctx, userCred, ctr, llm.Model)
			if err != nil {
				return errors.Wrap(err, "CheckModelExists")
			}
			if exists {
				log.Infoln("check model success", llm.Model)
				return nil
			}
		}
	}
}

func (llm *SLLM) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	llm.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	// set status to creating pod
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_CREATING_POD, "")
}
