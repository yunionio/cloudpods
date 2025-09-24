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
	"path"
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/drivers"
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
	// notifyclient.AddNotifyDBHookResources(LLMManager.KeywordPlural(), LLMManager.AliasPlural())
}

// func (manager *SLLMManager) DeleteByGuestId(ctx context.Context, userCred mcclient.TokenCredential, gstId string) error {
// 	q := manager.Query().Equals("guest_id", gstId)
// 	llms := make([]SLLM, 0)
// 	if err := db.FetchModelObjects(manager, q, &llms); err != nil {
// 		return errors.Wrap(err, "db.FetchModelObjects")
// 	}

// 	for _, llm := range llms {
// 		// log.Infoln("get in delete by container id", llm)
// 		if err := llm.RealDelete(ctx, userCred); nil != err {
// 			return err
// 		}
// 	}
// 	return nil
// }

func (manager *SLLMManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input *api.LLMCreateInput) (*api.LLMCreateInput, error) {
	if input.Model == "" {
		return nil, httperrors.NewNotEmptyError("model is required")
	}

	// find ollama container and set it
	var ctr *computeapi.PodContainerCreateInput
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

func (llm *SLLM) GetPodDriver() IPodDriver {
	return &drivers.SPodDriver{}
}

func (llm *SLLM) GetModelName() string {
	return llm.ModelName
}

func (llm *SLLM) GetModel() string {
	return llm.ModelName + ":" + llm.ModelTag
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
	server, err := llm.GetPodDriver().RequestCreatePod(ctx, userCred, &input.ServerCreateInput)
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

func (llm *SLLM) download(ctx context.Context, userCred mcclient.TokenCredential, taskId string, webUrl string, path string) error {
	input := &computeapi.ContainerDownloadFileInput{
		WebUrl: webUrl,
		Path:   path,
	}

	_, err := llm.GetPodDriver().RequestDownloadFileIntoContainer(ctx, userCred, llm.ContainerId, taskId, input)

	return err
}

func (llm *SLLM) exec(ctx context.Context, userCred mcclient.TokenCredential, command ...string) (string, error) {
	// exec command

	input := &computeapi.ContainerExecSyncInput{
		Command: command,
	}
	resp, err := llm.GetPodDriver().RequestExecSyncContainer(ctx, userCred, llm.ContainerId, input)

	// check error and return result
	var rst string
	if nil != err {
		return "", errors.Wrapf(err, "LLM exec error")
	}
	if nil != resp {
		rst = resp.String()
	}
	log.Infoln("llm container exec result: ", rst)
	return rst, nil
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

func (llm *SLLM) RunModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (llm *SLLM) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return llm.SVirtualResourceBase.Delete(ctx, userCred)
}

func (llm *SLLM) ConfirmContainerId(ctx context.Context, userCred mcclient.TokenCredential) error {
	ctrs, err := llm.GetPodDriver().RequestGetContainersByPodId(ctx, userCred, llm.GuestId)
	if err != nil {
		return err
	}

	// find ollama container id and set
	if len(ctrs.Data) != 1 {
		return errors.Errorf("Strange llm containers")
	}
	for _, ctr := range ctrs.Data {
		specJson, _ := ctr.GetString("spec")
		spec := new(computeapi.ContainerSpec)
		if err := jsonutils.JSONFalse.Unmarshal(spec, specJson); nil != err {
			continue
		}
		if strings.Contains(spec.Image, api.LLM_OLLAMA) {
			// update ContainerId
			ctrId, _ := ctr.GetString("id")
			llm.SetContainerId(ctrId)
			return nil
		}
	}

	return errors.Errorf("Cannot find ollama container in pod %s, with containers: %s", llm.GuestId, ctrs)
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
	input := new(api.LLMAccessCacheInput)
	if err := task.GetParams().Unmarshal(input); nil != err {
		return err
	}
	if _, err := llm.GetPodDriver().RequestOllamaBlobsCache(ctx, userCred, llm.ContainerId, task.GetTaskId(), input); nil != err {
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

	// copy gguf file
	if _, err := llm.exec(ctx, userCred, "/bin/sh", "-c", fmt.Sprintf("mkdir -p %s && cp %s %s", getGgufDir(llm), llm.GgufFile, getGgufFile(llm))); nil != err {
		return errors.Wrapf(err, "failed to prepare gguf file in container")
	}
	return nil
}

func (llm *SLLM) CopyBlobs(ctx context.Context, userCred mcclient.TokenCredential, blobs []string) error {
	// cp blobs
	blobsTargetDir := getBlobsPath(llm)
	blobsSrcDir := path.Join(api.LLM_OLLAMA_CACHE_MOUNT_PATH, api.LLM_OLLAMA_CACHE_DIR)

	var commands []string
	commands = append(commands, fmt.Sprintf("mkdir -p %s", blobsTargetDir))
	for _, blob := range blobs {
		src := path.Join(blobsSrcDir, blob)
		target := path.Join(blobsTargetDir, blob)
		commands = append(commands, fmt.Sprintf("cp %s %s", src, target))
	}

	cmd := strings.Join(commands, " && ")
	if _, err := llm.exec(ctx, userCred, "/bin/sh", "-c", cmd); err != nil {
		return errors.Wrapf(err, "failed to copy blobs to container")
	}
	return nil
}

func (llm *SLLM) DeleteModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	// stop running proccess
	if _, err := llm.exec(ctx, userCred, "/bin/ollama", "stop", llm.GetModel()); nil != err {
		return errors.Wrapf(err, "Stop ollama running model failed")
	}

	// delete manifests and blobs and gguf in container
	if _, err := llm.exec(ctx, userCred, "/bin/rm", "-r", "-f", path.Dir(getManifestsPath(llm))); nil != err {
		return errors.Wrapf(err, "Delete manifests file failed")
	}

	if _, err := llm.exec(ctx, userCred, "/bin/rm", "-r", "-f", getBlobsPath(llm)); nil != err {
		return errors.Wrapf(err, "Delete blobs file failed")
	}

	if _, err := llm.exec(ctx, userCred, "/bin/rm", "-r", "-f", getGgufDir(llm)); nil != err {
		return errors.Wrapf(err, "Delete gguf file failed")
	}

	return nil
}

func (llm *SLLM) DownloadManifests(ctx context.Context, userCred mcclient.TokenCredential, taskId string) error {
	// wget manifests
	suffix := fmt.Sprintf("%s/manifests/%s", llm.ModelName, llm.ModelTag)
	url := fmt.Sprintf(api.LLM_OLLAMA_LIBRARY_BASE_URL, suffix)

	return llm.download(ctx, userCred, taskId, url, getManifestsPath(llm))
}

func (llm *SLLM) DownloadGgufFile(ctx context.Context, userCred mcclient.TokenCredential, taskId string) error {
	return llm.download(ctx, userCred, taskId, llm.GgufFile, getGgufFile(llm))
}

func (llm *SLLM) FetchBlobs(ctx context.Context, userCred mcclient.TokenCredential) ([]string, error) {
	readCommand := fmt.Sprintf("cat %s", getManifestsPath(llm))
	manifestContent, err := llm.exec(ctx, userCred, "/bin/sh", "-c", readCommand)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read manifests from container")
	}

	// find all blobs
	var results []string
	re := regexp.MustCompile(`"digest":"(sha256:[^"]*)"`)
	matches := re.FindAllStringSubmatch(manifestContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			digest := match[1]
			processedDigest := strings.Replace(digest, "sha256:", "sha256-", 1)
			results = append(results, processedDigest)
		}
	}

	return results, nil
}

func (llm *SLLM) InstallGgufModel(ctx context.Context, userCred mcclient.TokenCredential, spec *api.LLMModelFileSpec) error {
	// touch modelfile
	modelfile := createModelFile(llm, spec)
	modelfilePath := path.Join(getGgufDir(llm), api.LLM_OLLAMA_MODELFILE_NAME)
	writeCommand := fmt.Sprintf("mkdir -p %s && printf '%%s' %q > %s", getGgufDir(llm), modelfile, modelfilePath)
	if _, err := llm.exec(ctx, userCred, "/bin/sh", "-c", writeCommand); nil != err {
		return errors.Wrapf(err, "failed to write modelfile into container")
	}

	// create model
	if _, err := llm.exec(ctx, userCred, api.LLM_OLLAMA_EXEC_PATH, api.LLM_OLLAMA_CREATE_ACTION, llm.GetModel(), "-f", modelfilePath); nil != err {
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

func createPullModelTask(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, input *api.LLMPullModelInput) (task *taskman.STask, err error) {
	if input.Gguf != nil {
		llm.GgufFile = input.Gguf.GgufFile
		task, err = taskman.TaskManager.NewTask(ctx, "LLMInstallGgufTask", llm, userCred, jsonutils.Marshal(input.Gguf).(*jsonutils.JSONDict), "", "", nil)
	} else {
		task, err = taskman.TaskManager.NewTask(ctx, "LLMPullModelTask", llm, userCred, jsonutils.NewDict(), "", "", nil)
	}

	return
}

func createModelFile(llm *SLLM, spec *api.LLMModelFileSpec) string {
	mdlFile := fmt.Sprintf(api.LLM_OLLAMA_GGUF_FROM, getGgufFile(llm))

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

func getBlobsPath(llm *SLLM) string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_BLOBS_DIR)
}

func getGgufDir(llm *SLLM) string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_GGUF_DIR, llm.GetModel())
}

func getGgufFile(llm *SLLM) string {
	return path.Join(getGgufDir(llm), path.Base(llm.GgufFile))
}

func getManifestsPath(llm *SLLM) string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_MANIFESTS_BASE_PATH, llm.ModelName, llm.ModelTag)
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
