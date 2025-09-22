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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
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

func (manager *SLLMManager) DeleteByGuestId(ctx context.Context, userCred mcclient.TokenCredential, gstId string) error {
	q := manager.Query().Equals("guest_id", gstId)
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

func (manager *SLLMManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input *api.LLMCreateInput) (*api.LLMCreateInput, error) {
	if input.Model == "" {
		return nil, httperrors.NewNotEmptyError("model is required")
	}

	// find ollama container and set it
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
	ollamaTrue := true
	ctr.OllamaContainer = &ollamaTrue

	// mount gguf file if found
	if nil != input.Gguf && input.Gguf.Source != api.LLM_OLLAMA_GGUF_SOURCE_WEB {
		hostPath := &apis.ContainerVolumeMountHostPath{
			Path:       input.Gguf.GgufFile,
			AutoCreate: false,
			Type:       apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
		}
		volumeMount := &apis.ContainerVolumeMount{
			HostPath:  hostPath,
			ReadOnly:  true,
			MountPath: input.Gguf.GgufFile,
			Type:      apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
		}
		ctr.VolumeMounts = append(ctr.VolumeMounts, volumeMount)
	}

	// set autostart
	input.AutoStart = true

	return input, nil
}

type SLLM struct {
	db.SVirtualResourceBase

	// GuestId is also the pod id
	GuestId     string `width:"36" charset:"ascii" list:"user" index:"true" create:"optional"`
	ContainerId string `width:"36" charset:"ascii" list:"user" index:"true" create:"optional"`
	ModelName   string `width:"64" charset:"ascii" default:"qwen3" list:"user" update:"user" create:"optional"`
	ModelTag    string `width:"64" charset:"ascii" default:"latest" list:"user" update:"user" create:"optional"`
	GgufFile    string `width:"256" charset:"ascii" list:"user" update:"user" create:"optional"`
}

func (llm *SLLM) GetModelName() string {
	return llm.ModelName
}

func (llm *SLLM) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// unmarshal input
	input := &api.LLMCreateInput{}
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "Unmarshal ServerCreateInput")
	}

	// get model name and model tag
	model := input.Model
	llm.ModelName, llm.ModelTag = parseModel(model)

	// init task, decide import model from cache or gguf-file
	llm.Id = stringutils.UUID4()
	task, err := createPullModelTask(ctx, userCred, llm, &input.LLMPullModelInput)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	input.ParentTaskId = task.GetId()

	// use data to create a pod
	handler := db.NewModelHandler(GuestManager)
	server, err := handler.Create(ctx, jsonutils.NewDict(), jsonutils.Marshal(input.ServerCreateInput), nil)
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

func (llm *SLLM) exec(ctx context.Context, userCred mcclient.TokenCredential, stdin io.Reader, isSync bool, command ...string) (string, error) {
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

	// exec command
	var ret string
	if isSync {
		var resp jsonutils.JSONObject
		input := &api.ContainerExecSyncInput{
			Command: command,
		}
		resp, err = ctr.GetPodDriver().RequestExecSyncContainer(ctx, userCred, ctr, input)
		if nil != resp {
			ret = resp.String()
		}
	} else {
		input := &api.ContainerExecInput{
			Command: command,
		}
		ret, err = ctr.GetPodDriver().RequestExecStreamContainer(ctx, userCred, ctr, input, stdin)
	}

	// check error and return result
	if nil != err {
		return "", errors.Wrapf(err, "LLM exec error")
	}
	return ret, nil
}

func (llm *SLLM) PerformChangeModel(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LLMPullModelInput) (jsonutils.JSONObject, error) {
	// check model is the same with current
	if input.Model == llm.GetModel() && input.Gguf == nil && llm.GgufFile == "" {
		return nil, errors.Errorf("LLM run with model %s already", input.Model)
	}

	// delete model
	if err := llm.DeleteModel(ctx, userCred); nil != err {
		return nil, err
	}

	// set modelName and modelTag
	llm.UpdateModel(input.Model)

	// pull new model
	task, err := createPullModelTask(ctx, userCred, llm, input)
	if err != nil {
		return nil, errors.Wrap(err, "NewTask")
	}

	return jsonutils.NewDict(), task.ScheduleRun(nil)
}

// func (llm *SLLM) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
// 	if !sets.NewString(api.CONTAINER_STATUS_EXITED, api.CONTAINER_STATUS_START_FAILED).Has(llm.Status) {
// 		return nil, httperrors.NewInvalidStatusError("Can't start llm in status %s", llm.Status)
// 	}
// 	return nil, llm.StartStartTask(ctx, userCred, "")
// }

// func (llm *SLLM) StartStartTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
// 	llm.SetStatus(ctx, userCred, api.CONTAINER_STATUS_STARTING, "")
// 	task, err := taskman.TaskManager.NewTask(ctx, "LLMStartTask", llm, userCred, nil, parentTaskId, "", nil)
// 	if err != nil {
// 		return errors.Wrap(err, "NewTask")
// 	}
// 	return task.ScheduleRun(nil)
// }

func (llm *SLLM) RunModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (llm *SLLM) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return llm.SVirtualResourceBase.Delete(ctx, userCred)
}

func (llm *SLLM) GetContainer() (*SContainer, error) {
	ctrMngr := GetContainerManager()
	// log.Infoln("get container", llm.ContainerId)
	if llm.ContainerId == "" {
		return llmGetOllamaContainer(ctrMngr, llm)
	} else {
		return llmFetchContainerById(ctrMngr, llm)
	}
}

func (llm *SLLM) SetContainerId(ctrId string) error {

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

func (llm *SLLM) AccessBlobsCache(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask) error {
	// update status
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_DOWNLOADING_BLOBS, "")
	// access blobs
	if _, err := accessModelCache(ctx, llm, task); nil != err {
		return err
	}
	return nil
}

func (llm *SLLM) AccessGgufFile(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask) error {
	// input := &api.LLMAccessGgufFileInput{
	// 	TargetDir: llm.GetGgufPath(),
	// 	HostPath:  llm.GgufFile,
	// }
	// // access gguf file
	// if _, err := accessGgufFile(ctx, llm, task, input); nil != err {
	// 	return err
	// }

	// mkdir
	if _, err := llm.exec(ctx, userCred, nil, true, "/bin/mkdir", "-p", llm.GetGgufDir()); nil != err {
		return errors.Wrapf(err, "failed to mkdir")
	}

	// copy gguf file
	if _, err := llm.exec(ctx, userCred, nil, true, "/bin/cp", llm.GgufFile, llm.GetGgufFile()); nil != err {
		return errors.Wrapf(err, "failed to write gguf file into container")
	}
	return nil
}

func (llm *SLLM) CopyBlobs(ctx context.Context, userCred mcclient.TokenCredential, blobs []string) error {
	// mkdir blobs
	blobsTargetDir := llm.GetBlobsPath()
	_, err := llm.exec(ctx, userCred, nil, true, "/bin/mkdir", "-p", blobsTargetDir)
	if nil != err {
		return err
	}
	// cp blobs
	blobsSrcDir := path.Join(api.LLM_OLLAMA_CACHE_MOUNT_PATH, api.LLM_OLLAMA_CACHE_DIR)
	for _, blob := range blobs {
		src := path.Join(blobsSrcDir, blob)
		target := path.Join(blobsTargetDir, blob)
		_, err = llm.exec(ctx, userCred, nil, true, "/bin/cp", src, target)
		if nil != err {
			return err
		}
	}
	return nil
}

func (llm *SLLM) DeleteModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	// stop running proccess
	if _, err := llm.exec(ctx, userCred, nil, true, "/bin/ollama", "stop", llm.GetModel()); nil != err {
		return errors.Wrapf(err, "Stop ollama running model failed")
	}

	// delete manifests and blobs and gguf in container
	if _, err := llm.exec(ctx, userCred, nil, true, "/bin/rm", "-r", "-f", path.Dir(llm.GetManifestsPath())); nil != err {
		return errors.Wrapf(err, "Delete manifests file failed")
	}

	if _, err := llm.exec(ctx, userCred, nil, true, "/bin/rm", "-r", "-f", llm.GetBlobsPath()); nil != err {
		return errors.Wrapf(err, "Delete blobs file failed")
	}

	if _, err := llm.exec(ctx, userCred, nil, true, "/bin/rm", "-r", "-f", llm.GetGgufDir()); nil != err {
		return errors.Wrapf(err, "Delete gguf file failed")
	}

	return nil
}

func (llm *SLLM) DownloadManifests(ctx context.Context, userCred mcclient.TokenCredential) ([]string, error) {
	// wget manifests
	suffix := fmt.Sprintf("%s/manifests/%s", llm.ModelName, llm.ModelTag)
	url := fmt.Sprintf(api.LLM_OLLAMA_LIBRARY_BASE_URL, suffix)
	resp, err := webGet(url)
	if nil != err {
		return nil, err
	}
	defer resp.Close()

	// write manifests into container
	manifestBytes, err := io.ReadAll(resp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body")
	}
	filePath := llm.GetManifestsPath()
	dirPath := path.Dir(filePath)
	writeCommand := fmt.Sprintf("mkdir -p %s && cat > %s", dirPath, filePath)
	if _, err := llm.exec(ctx, userCred, bytes.NewReader(manifestBytes), false, "/bin/sh", "-c", writeCommand); nil != err {
		return nil, errors.Wrapf(err, "failed to write manifests into container")
	}

	// find all blobs
	var results []string
	re := regexp.MustCompile(`"digest":"(sha256:[^"]*)"`)
	matches := re.FindAllStringSubmatch(string(manifestBytes), -1)
	for _, match := range matches {
		if len(match) > 1 {
			digest := match[1]
			processedDigest := strings.Replace(digest, "sha256:", "sha256-", 1)
			results = append(results, processedDigest)
		}
	}

	return results, nil
}

func (llm *SLLM) DownloadGgufFile(ctx context.Context, userCred mcclient.TokenCredential) error {
	// wget gguf file
	resp, err := webGet(llm.GgufFile)
	if nil != err {
		return err
	}
	defer resp.Close()

	// write gguf file into container
	writeCommand := fmt.Sprintf("mkdir -p %s && cat > %s", llm.GetGgufDir(), llm.GetGgufFile())
	if _, err := llm.exec(ctx, userCred, resp, false, "/bin/sh", "-c", writeCommand); nil != err {
		return errors.Wrapf(err, "failed to write gguf file into container")
	}

	return nil
}

func (llm *SLLM) GetModel() string {
	return llm.ModelName + ":" + llm.ModelTag
}

func (llm *SLLM) GetBlobsPath() string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_BLOBS_DIR)
}

func (llm *SLLM) GetGgufDir() string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_GGUF_DIR, llm.GetModel())
}

func (llm *SLLM) GetGgufFile() string {
	return path.Join(llm.GetGgufDir(), path.Base(llm.GgufFile))
}

func (llm *SLLM) GetManifestsPath() string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_MANIFESTS_BASE_PATH, llm.ModelName, llm.ModelTag)
}

func (llm *SLLM) InstallGgufModel(ctx context.Context, userCred mcclient.TokenCredential, spec *api.LLMModelFileSpec) error {
	// touch modelfile
	modelfile := []byte(createModelFile(llm, spec))
	modelfilePath := path.Join(llm.GetGgufDir(), api.LLM_OLLAMA_MODELFILE_NAME)
	writeCommand := fmt.Sprintf("cat > %s", modelfilePath)
	if _, err := llm.exec(ctx, userCred, bytes.NewReader(modelfile), false, "/bin/sh", "-c", writeCommand); nil != err {
		return errors.Wrapf(err, "failed to write modelfile into container")
	}

	// create model
	if _, err := llm.exec(ctx, userCred, nil, false, api.LLM_OLLAMA_EXEC_PATH, api.LLM_OLLAMA_CREATE_ACTION, llm.GetModel(), "-f", modelfilePath); nil != err {
		return errors.Wrapf(err, "failed to create model with model file")
	}

	return nil
}

func (llm *SLLM) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	llm.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	// set status to creating pod
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_CREATING_POD, "")
}

func (llm *SLLM) UpdateModel(model string) error {
	if _, err := db.Update(llm, func() error {
		llm.ModelName, llm.ModelTag = parseModel(model)
		return nil
	}); nil != err {
		return errors.Wrapf(err, "update llm model %s", model)
	}

	return nil
}

func accessModelCache(ctx context.Context, llm *SLLM, task taskman.ITask) (jsonutils.JSONObject, error) {
	container, err := llm.GetContainer()
	if nil != err {
		return nil, err
	}
	pod := container.GetPod()
	host, _ := pod.GetHost()
	url := fmt.Sprintf("%s/pods/%s/containers/%s/llms/%s/access-cache", host.ManagerUri, pod.GetId(), container.GetId(), llm.GetId())
	header := task.GetTaskRequestHeader()
	_, ret, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, task.GetParams(), false)
	return ret, err
}

// func accessGgufFile(ctx context.Context, llm *SLLM, task taskman.ITask, input *api.LLMAccessGgufFileInput) (jsonutils.JSONObject, error) {
// 	container, err := llm.GetContainer()
// 	if nil != err {
// 		return nil, err
// 	}
// 	pod := container.GetPod()
// 	host, _ := pod.GetHost()
// 	url := fmt.Sprintf("%s/pods/%s/containers/%s/llms/%s/access-gguf-file", host.ManagerUri, pod.GetId(), container.GetId(), llm.GetId())
// 	header := task.GetTaskRequestHeader()
// 	_, ret, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, jsonutils.Marshal(input), false)
// 	return ret, err
// }

func createPullModelTask(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, input *api.LLMPullModelInput) (task *taskman.STask, err error) {
	if input != nil {
		llm.GgufFile = input.Gguf.GgufFile
		task, err = taskman.TaskManager.NewTask(ctx, "LLMInstallGgufTask", llm, userCred, jsonutils.Marshal(input.Gguf).(*jsonutils.JSONDict), "", "", nil)
	} else {
		task, err = taskman.TaskManager.NewTask(ctx, "LLMPullModelTask", llm, userCred, jsonutils.NewDict(), "", "", nil)
	}

	return
}

func createModelFile(llm *SLLM, spec *api.LLMModelFileSpec) string {
	mdlFile := fmt.Sprintf(api.LLM_OLLAMA_GGUF_FROM, llm.GetGgufFile())

	// Parameter
	if nil != spec.Parameter {
		params := spec.Parameter.GetParameters()
		for name, param := range params {
			mdlFile += fmt.Sprintf(api.LLM_OLLAMA_GGUF_PARAMETER, name, param)
		}
	}

	// Template
	if nil != spec.Template {
		mdlFile += fmt.Sprintf(api.LLM_OLLAMA_GGUF_TEMPLATE, *spec.Template)
	}

	// System
	if nil != spec.System {
		mdlFile += fmt.Sprintf(api.LLM_OLLAMA_GGUF_SYSTEM, *spec.System)
	}

	// License
	if nil != spec.License {
		mdlFile += fmt.Sprintf(api.LLM_OLLAMA_GGUF_LICENSE, *spec.License)
	}

	// Message
	if nil != spec.Message {
		for _, msgPtr := range spec.Message {
			if msgPtr == nil || msgPtr.ValidateRole() != nil {
				continue
			}
			message := *msgPtr
			mdlFile += fmt.Sprintf(api.LLM_OLLAMA_GGUF_MESSAGE, message.Role, message.Content)
		}
	}

	return mdlFile
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
			llm.SetContainerId(ctr.GetId())
			return &ctr, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "ollama container for guest %s not found", llm.GuestId)
}

func parseModel(model string) (string, string) {
	parts := strings.Split(model, ":")
	modelName := parts[0]
	modelTag := "latest"
	if len(parts) > 1 {
		modelTag = parts[1]
	}
	return modelName, modelTag
}

func webGet(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create http request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "http get request failed")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to get web file, status code: %d, url: %s", resp.StatusCode, url)
	}

	return resp.Body, nil
}
