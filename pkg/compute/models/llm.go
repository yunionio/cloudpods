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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
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

func (llm *SLLM) llmFetchContainerById(ctrMngr *SContainerManager) (*SContainer, error) {
	ctr := &SContainer{}
	ctr.Id = llm.ContainerId
	if err := ctrMngr.Query().First(ctr); err != nil {
		return nil, errors.Wrap(err, "Query.First Container")
	}
	return ctr, nil
}

func (llm *SLLM) llmGetOllamaContainer(ctrMngr *SContainerManager) (*SContainer, error) {
	ctrs, err := ctrMngr.GetContainersByPod(llm.GuestId)
	if err != nil {
		return nil, err
	}
	// found ollama
	for _, ctr := range ctrs {
		if strings.Contains(ctr.Spec.Image, "ollama") {
			/* TODO: set and update ContainerId */
			llm.ContainerId = ctr.GetId()
			return &ctr, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "ollama container for guest %s not found", llm.GuestId)
}

func (llm *SLLM) GetContainer() (*SContainer, error) {
	ctrMngr := GetContainerManager()
	if llm.ContainerId == "" {
		return llm.llmGetOllamaContainer(ctrMngr)
	} else {
		return llm.llmFetchContainerById(ctrMngr)
	}
}

// func (manager *SLLMManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
// }

func (llm *SLLM) CheckModelExists(ctx context.Context, userCred mcclient.TokenCredential, ctr *SContainer, model string) (exists bool, err error) {
	listInput := &api.ContainerExecSyncInput{
		Command: []string{api.LLM_OLLAMA_EXEC_PATH, api.LLM_OLLAMA_LIST_ACTION},
		Timeout: 0,
	}

	list, err := ctr.PerformExecSync(ctx, userCred, nil, listInput)
	if nil != err {
		errors.Wrap(err, "Ollama list")
		return
	}

	listResp := new(api.ContainerExecSyncResponse)
	if err = list.Unmarshal(listResp); err != nil {
		errors.Wrap(err, "Unmarshal list response")
		return
	}

	// Check if the model exists in the output
	// ollama list output format: NAME  ID  SIZE  MODIFIED
	lines := strings.Split(listResp.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == model {
			return true, nil
		}
	}

	return false, nil
}

func (llm *SLLM) PullModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	// get container
	ctr, err := llm.GetContainer()
	if nil != err {
		return err
	}
	// init input
	pullInput := &api.ContainerExecSyncInput{
		Command: []string{api.LLM_OLLAMA_EXEC_PATH, api.LLM_OLLAMA_PULL_ACTION, llm.Model, "&"},
		Timeout: 0,
	}
	// pull model
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
				return nil
			}
		}
	}
}

// func (llm *SLLM) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
// 	llm.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
// }
