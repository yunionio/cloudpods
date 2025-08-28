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
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	apis "yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/pod/remotecommand"
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
	// notifyclient.AddNotifyDBHookResources(LLMManager.KeywordPlural(), LLMManager.AliasPlural())
}

func (manager *SLLMManager) DeleteByContainerId(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) error {
	q := manager.Query().Equals("container_id", ctrId)
	llms := make([]SLLM, 0)
	if err := db.FetchModelObjects(manager, q, &llms); err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}

	for _, llm := range llms {
		// log.Infoln("get in delete by container id", llm)
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
	task, err := taskman.TaskManager.NewTask(ctx, "LLMPullModelTask", llm, userCred, jsonutils.NewDict(), "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}

	// init params
	params, err := llm.InitPodCreateData(data, task.GetId())
	if err != nil {
		return errors.Wrap(err, "Customize pod create")
	}

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

func (llm *SLLM) exec(userCred mcclient.TokenCredential, command ...string) (string, error) {
	// get container
	ctr, err := llm.GetContainer()
	if nil != err {
		return "", err
	}

	// check status
	if sets.NewString(
		api.CONTAINER_STATUS_EXITED,
		api.CONTAINER_STATUS_CREATED,
	).Has(ctr.GetStatus()) {
		return "", httperrors.NewInvalidStatusError("Can't exec container in status %s", ctr.Status)
	}

	// get Guest and Host
	guest := ctr.GetPod()
	host, _ := guest.GetHost()

	// exec command
	input := &api.ContainerExecInput{
		Command: command,
		Tty:     false,
		SetIO:   true,
		Stdin:   false,
		Stdout:  true,
	}
	urlLoc := fmt.Sprintf("%s/pods/%s/containers/%s/exec?%s", host.ManagerUri, guest.GetId(), ctr.GetId(), jsonutils.Marshal(input).QueryString())
	url, err := url.Parse(urlLoc)
	if err != nil {
		return "", errors.Wrapf(err, "parse url: %s", urlLoc)
	}
	ret, err := execStream(url, mcclient.GetTokenHeaders(userCred))
	if nil != err {
		return "", errors.Wrapf(err, "LLM exec error")
	}
	log.Infoln("get exec stream result", ret)

	// check error and return result
	return ret, nil
}

func (llm *SLLM) InitPodCreateData(data jsonutils.JSONObject, taskId string) (jsonutils.JSONObject, error) {
	// init input and set parend task id
	input := &api.ServerCreateInput{}
	if err := data.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "Unmarshal ServerCreateInput")
	}
	input.ParentTaskId = taskId

	// set auto_start
	input.AutoStart = true

	// get pod container
	if len(input.Pod.Containers) == 0 {
		return nil, errors.Errorf("Miss container in llm create")
	}
	var ctr *api.PodContainerCreateInput
	finded := false
	for _, c := range input.Pod.Containers {
		if strings.Contains(c.Image, api.LLM_OLLAMA) {
			ctr = c
			finded = true
			break
		}
	}
	if !finded {
		return nil, errors.Errorf("Image must be ollama")
	}

	// mount volume
	hostPath := &apis.ContainerVolumeMountHostPath{
		Path:       api.LLM_OLLAMA_CACHE_HOST_PATH,
		AutoCreate: true,
	}
	volumeMount := &apis.ContainerVolumeMount{
		HostPath:  hostPath,
		ReadOnly:  false,
		MountPath: api.LLM_OLLAMA_CACHE_MOUNT_PATH,
		Type:      apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
	}
	ctr.VolumeMounts = append(ctr.VolumeMounts, volumeMount)

	// set enviroment OLLAMA_HOST=0.0.0.0:11434
	env := &apis.ContainerKeyValue{
		Key:   api.LLM_OLLAMA_EXPORT_ENV_KEY,
		Value: api.LLM_OLLAMA_EXPORT_ENV_VALUE,
	}
	ctr.Envs = append(ctr.Envs, env)

	log.Infoln("In llm create: pod create input", jsonutils.Marshal(input).String())
	return jsonutils.Marshal(input), nil
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
	// pull model
	_, err := llm.exec(userCred, api.LLM_OLLAMA_EXEC_PATH, api.LLM_OLLAMA_PULL_ACTION, llm.Model)
	return err
}

func (llm *SLLM) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	llm.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	// set status to creating pod
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_CREATING_POD, "")
}

func cleanFinalANSI(input string) string {
	ansiRegexp := regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	input = strings.ReplaceAll(input, "\r", "")
	return ansiRegexp.ReplaceAllString(input, "")
}

func processTerminalStream(input string) string {
	input = strings.ReplaceAll(input, "\x1b[1G", "\r")
	lines := strings.Split(input, "\n")
	var resultLines []string

	for _, line := range lines {
		parts := strings.Split(line, "\r")
		finalPart := parts[len(parts)-1]
		cleanedLine := cleanFinalANSI(finalPart)
		if cleanedLine != "" {
			resultLines = append(resultLines, cleanedLine)
		}
	}

	return strings.Join(resultLines, "\n")
}

func execStream(url *url.URL, headers http.Header) (string, error) {
	// define out stream and err stream
	var outBuf, errBuf strings.Builder
	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()

	// define goroutine to copy result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(&outBuf, outReader)
	}()

	go func() {
		defer wg.Done()
		io.Copy(&errBuf, errReader)
	}()

	// exec stream
	exec, err := remotecommand.NewSPDYExecutor("POST", url)
	if err != nil {
		return "", errors.Wrap(err, "NewSPDYExecutor")
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             nil,
		Stdout:            outWriter,
		Stderr:            errWriter,
		Tty:               false,
		TerminalSizeQueue: nil,
		Header:            headers,
	})

	// copy result
	outWriter.Close()
	errWriter.Close()
	wg.Wait()

	outResult := processTerminalStream(outBuf.String())
	errResult := processTerminalStream(errBuf.String())
	if err != nil {
		return "", errors.Wrapf(err, "exec stream error (stdout [%s], stderr [%s])", outResult, errResult)
	}
	return outResult, nil
}

func llmFetchContainerById(ctrMngr *SContainerManager, llm *SLLM) (*SContainer, error) {
	ctr, err := ctrMngr.FetchById(llm.ContainerId)
	if nil != err {
		return nil, errors.Wrap(err, "llmFetchContainerById")
	}
	return ctr.(*SContainer), nil
}

func llmGetOllamaContainer(ctrMngr *SContainerManager, llm *SLLM) (*SContainer, error) {
	ctrs, err := ctrMngr.GetContainersByPod(llm.GuestId)
	if err != nil {
		return nil, err
	}
	// found ollama
	for _, ctr := range ctrs {
		if strings.Contains(ctr.Spec.Image, api.LLM_OLLAMA) {
			// update ContainerId
			llm.SetContainer(ctr.GetId())
			return &ctr, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "ollama container for guest %s not found", llm.GuestId)
}
