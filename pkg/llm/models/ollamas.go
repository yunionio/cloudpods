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
	"encoding/base64"
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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

type SOllamaManager struct {
	db.SVirtualResourceBaseManager
}

var OllamaManager *SOllamaManager

func init() {
	OllamaManager = &SOllamaManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SOllama{},
			"llms_tbl",
			"llm",
			"llms",
		),
	}
	OllamaManager.SetVirtualObject(OllamaManager)
	OllamaManager.SetAlias("llm", "llms")
	OllamaManager.NameRequireAscii = false
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

func (manager *SOllamaManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query *api.OllamaListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	if query.GuestId != "" {
		q = q.Equals("guest_id", query.GuestId)
	}
	return q, err
}

func (manager *SOllamaManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input *api.OllamaCreateInput) (*api.OllamaCreateInput, error) {
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

type SOllama struct {
	SLLMBase

	ContainerId string `width:"36" charset:"ascii" list:"user" index:"true" create:"optional"`
	ModelName   string `width:"64" charset:"ascii" default:"qwen3" list:"user" update:"user" create:"optional"`
	ModelTag    string `width:"64" charset:"ascii" default:"latest" list:"user" update:"user" create:"optional"`
	GgufFile    string `width:"256" charset:"ascii" list:"user" update:"user" create:"optional"`
}

func (o *SOllama) GetLLMBase() *SLLMBase {
	return &o.SLLMBase
}

func (ollama *SOllama) GetModelName() string {
	return ollama.ModelName
}

func (ollama *SOllama) GetModel() string {
	return ollama.ModelName + ":" + ollama.ModelTag
}

func (ollama *SOllama) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// unmarshal input
	input := &api.OllamaCreateInput{}
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "Unmarshal ServerCreateInput")
	}

	// get model name and model tag
	model := input.Model
	ollama.ModelName, ollama.ModelTag = parseModel(model)

	// // init task, decide import model from cache or gguf-file
	// llm.Id = stringutils.UUID4()
	// task, err := createPullModelTask(ctx, userCred, llm, &input.LLMPullModelInput)
	// if err != nil {
	// 	return errors.Wrap(err, "NewTask")
	// }
	// input.ParentTaskId = task.GetId()

	// // use data to create a pod
	// server, err := llm.GetPodDriver().RequestCreatePod(ctx, userCred, &input.ServerCreateInput)
	// if err != nil {
	// 	return errors.Wrap(err, "CreateServer")
	// }

	// // set GuestID
	// guestId, err := server.GetString("id")
	// if err != nil {
	// 	return errors.Wrap(err, "GetGuestId")
	// }
	// llm.GuestId = guestId
	return ollama.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (ollama *SOllama) download(ctx context.Context, userCred mcclient.TokenCredential, taskId string, webUrl string, path string) error {
	input := &computeapi.ContainerDownloadFileInput{
		WebUrl: webUrl,
		Path:   path,
	}

	_, err := ollama.GetPodDriver().RequestDownloadFileIntoContainer(ctx, userCred, ollama.ContainerId, taskId, input)

	return err
}

func (ollama *SOllama) exec(ctx context.Context, userCred mcclient.TokenCredential, command ...string) (string, error) {
	// exec command

	input := &computeapi.ContainerExecSyncInput{
		Command: command,
	}
	resp, err := ollama.GetPodDriver().RequestExecSyncContainer(ctx, userCred, ollama.ContainerId, input)

	// check error and return result
	var rst string
	if nil != err || resp == nil {
		return "", errors.Wrapf(err, "LLM exec error")
	}
	rst, _ = resp.GetString("stdout")
	log.Infoln("llm container exec result: ", resp)
	return rst, nil
}

func (ollama *SOllama) PerformChangeModel(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.OllamaPullModelInput) (jsonutils.JSONObject, error) {
	// check model is the same with current
	if input.Model == ollama.GetModel() && input.Gguf == nil && ollama.GgufFile == "" {
		return nil, errors.Errorf("LLM run with model %s already", input.Model)
	}

	// delete model
	if err := ollama.DeleteModel(ctx, userCred); nil != err {
		return nil, err
	}

	// set modelName and modelTag
	ollama.UpdateModel(input.Model)

	// pull new model
	task, err := createPullModelTask(ctx, userCred, ollama, input)
	if err != nil {
		return nil, errors.Wrap(err, "NewTask")
	}

	return jsonutils.NewDict(), task.ScheduleRun(nil)
}

func (ollama *SOllama) RunModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

// func (ollama *SOllama) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
// 	return ollama.SVirtualResourceBase.Delete(ctx, userCred)
// }

func (ollama *SOllama) ConfirmContainerId(ctx context.Context, userCred mcclient.TokenCredential) error {
	if ollama.ContainerId != "" {
		return nil
	}

	ctrs, err := ollama.GetPodDriver().RequestGetContainersByPodId(ctx, userCred, ollama.GuestId)
	if err != nil {
		return err
	}

	// find ollama container id and set
	if len(ctrs) != 1 {
		return errors.Errorf("Strange llm containers")
	}
	for _, ctr := range ctrs {
		specJson, _ := ctr.Get("spec")
		spec := new(computeapi.ContainerSpec)
		if err := specJson.Unmarshal(spec); nil != err {
			continue
		}
		if strings.Contains(spec.Image, api.LLM_OLLAMA) {
			// update ContainerId
			ctrId, _ := ctr.GetString("id")
			ollama.SetContainerId(ctrId)
			return nil
		}
	}

	return errors.Errorf("Cannot find ollama container in pod %s, with containers: %s", ollama.GuestId, ctrs)
}

func (ollama *SOllama) SetContainerId(ctrId string) error {
	if _, err := db.Update(ollama, func() error {
		ollama.ContainerId = ctrId
		return nil
	}); nil != err {
		return errors.Wrapf(err, "update llm container with %s", ctrId)
	}
	// log.Infoln("update container id", ret.String())
	return nil
}

func (ollama *SOllama) SetGgufFile(ggufFile string) error {
	if _, err := db.Update(ollama, func() error {
		ollama.GgufFile = ggufFile
		return nil
	}); nil != err {
		return errors.Wrapf(err, "update llm GGUF file with %s", ggufFile)
	}
	// log.Infoln("update container id", ret.String())
	return nil
}

// func (manager *SLLMManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
// }

func (ollama *SOllama) AccessBlobsCache(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask) error {
	// update status
	ollama.SetStatus(ctx, userCred, api.LLM_STATUS_DOWNLOADING_BLOBS, "")
	// access blobs
	input := new(api.OllamaAccessCacheInput)
	if err := task.GetParams().Unmarshal(input); nil != err {
		return err
	}
	if _, err := ollama.GetPodDriver().RequestOllamaBlobsCache(ctx, userCred, ollama.ContainerId, task.GetTaskId(), input); nil != err {
		return err
	}
	return nil
}

func (ollama *SOllama) AccessGgufFile(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask) error {
	// input := &api.LLMAccessGgufFileInput{
	// 	TargetDir: llm.GetGgufPath(),
	// 	HostPath:  llm.GgufFile,
	// }
	// // access gguf file
	// if _, err := accessGgufFile(ctx, llm, task, input); nil != err {
	// 	return err
	// }

	// copy gguf file
	if _, err := ollama.exec(ctx, userCred, "/bin/sh", "-c", fmt.Sprintf("mkdir -p %s && cp %s %s", getGgufDir(ollama), ollama.GgufFile, getGgufFile(ollama))); nil != err {
		return errors.Wrapf(err, "failed to prepare gguf file in container")
	}
	return nil
}

func (ollama *SOllama) CopyBlobs(ctx context.Context, userCred mcclient.TokenCredential, blobs []string) error {
	// cp blobs
	blobsTargetDir := getBlobsPath()
	blobsSrcDir := path.Join(api.LLM_OLLAMA_CACHE_MOUNT_PATH, api.LLM_OLLAMA_CACHE_DIR)

	var commands []string
	commands = append(commands, fmt.Sprintf("mkdir -p %s", blobsTargetDir))
	for _, blob := range blobs {
		src := path.Join(blobsSrcDir, blob)
		target := path.Join(blobsTargetDir, blob)
		commands = append(commands, fmt.Sprintf("cp %s %s", src, target))
	}

	cmd := strings.Join(commands, " && ")
	if _, err := ollama.exec(ctx, userCred, "/bin/sh", "-c", cmd); err != nil {
		return errors.Wrapf(err, "failed to copy blobs to container")
	}
	return nil
}

func (ollama *SOllama) DeleteModel(ctx context.Context, userCred mcclient.TokenCredential) error {
	// stop running proccess
	if _, err := ollama.exec(ctx, userCred, "/bin/ollama", "stop", ollama.GetModel()); nil != err {
		return errors.Wrapf(err, "Stop ollama running model failed")
	}

	// delete manifests and blobs and gguf in container
	if _, err := ollama.exec(ctx, userCred, "/bin/rm", "-r", "-f", path.Dir(getManifestsPath(ollama))); nil != err {
		return errors.Wrapf(err, "Delete manifests file failed")
	}

	if _, err := ollama.exec(ctx, userCred, "/bin/rm", "-r", "-f", getBlobsPath()); nil != err {
		return errors.Wrapf(err, "Delete blobs file failed")
	}

	if _, err := ollama.exec(ctx, userCred, "/bin/rm", "-r", "-f", getGgufDir(ollama)); nil != err {
		return errors.Wrapf(err, "Delete gguf file failed")
	}

	return nil
}

func (ollama *SOllama) DownloadManifests(ctx context.Context, userCred mcclient.TokenCredential, taskId string) error {
	// wget manifests
	suffix := fmt.Sprintf("%s/manifests/%s", ollama.ModelName, ollama.ModelTag)
	url := fmt.Sprintf(api.LLM_OLLAMA_LIBRARY_BASE_URL, suffix)

	return ollama.download(ctx, userCred, taskId, url, getManifestsPath(ollama))
}

func (ollama *SOllama) DownloadGgufFile(ctx context.Context, userCred mcclient.TokenCredential, taskId string) error {
	return ollama.download(ctx, userCred, taskId, ollama.GgufFile, getGgufFile(ollama))
}

func (ollama *SOllama) FetchBlobs(ctx context.Context, userCred mcclient.TokenCredential) ([]string, error) {
	readCommand := fmt.Sprintf("cat %s", getManifestsPath(ollama))
	manifestContent, err := ollama.exec(ctx, userCred, "/bin/sh", "-c", readCommand)
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

func (ollama *SOllama) InstallGgufModel(ctx context.Context, userCred mcclient.TokenCredential, spec *api.OllamaModelFileSpec) error {
	// touch modelfile
	modelfile := createModelFile(ollama, spec)
	modelfilePath := path.Join(getGgufDir(ollama), api.LLM_OLLAMA_MODELFILE_NAME)

	encoded := base64.StdEncoding.EncodeToString([]byte(modelfile))

	writeCommand := fmt.Sprintf("mkdir -p %s && echo %s | base64 -d > %s", getGgufDir(ollama), encoded, modelfilePath)
	if _, err := ollama.exec(ctx, userCred, "/bin/sh", "-c", writeCommand); nil != err {
		return errors.Wrapf(err, "failed to write modelfile into container")
	}

	// create model
	// if _, err := ollama.exec(ctx, userCred, api.LLM_OLLAMA_EXEC_PATH, api.LLM_OLLAMA_CREATE_ACTION, ollama.GetModel(), "-f", modelfilePath); nil != err {
	// 	return errors.Wrapf(err, "failed to create model with model file")
	// }
	ollama.exec(ctx, userCred, api.LLM_OLLAMA_EXEC_PATH, api.LLM_OLLAMA_CREATE_ACTION, ollama.GetModel(), "-f", modelfilePath) // ignore error to avoid api timeout

	return nil
}

func (ollama *SOllama) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := new(api.OllamaCreateInput)
	if err := data.Unmarshal(&input); nil != err {
		return
	}
	task, err := createPullModelTask(ctx, userCred, ollama, &input.OllamaPullModelInput)
	if nil != err {
		return
	}
	ollama.StartCreatePodTask(ctx, userCred, &input.ServerCreateInput, task.GetId())
	ollama.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
}

func (ollama *SOllama) UpdateModel(model string) error {
	if _, err := db.Update(ollama, func() error {
		ollama.ModelName, ollama.ModelTag = parseModel(model)
		return nil
	}); nil != err {
		return errors.Wrapf(err, "update llm model %s", model)
	}

	return nil
}

func createPullModelTask(ctx context.Context, userCred mcclient.TokenCredential, ollama *SOllama, input *api.OllamaPullModelInput) (task *taskman.STask, err error) {
	if input.Gguf != nil {
		ollama.SetGgufFile(input.Gguf.GgufFile)
		task, err = taskman.TaskManager.NewTask(ctx, "LLMInstallGgufTask", ollama, userCred, jsonutils.Marshal(input.Gguf).(*jsonutils.JSONDict), "", "", nil)
	} else {
		task, err = taskman.TaskManager.NewTask(ctx, "LLMPullModelTask", ollama, userCred, jsonutils.NewDict(), "", "", nil)
	}

	return
}

func createModelFile(ollama *SOllama, spec *api.OllamaModelFileSpec) string {
	mdlFile := fmt.Sprintf(api.LLM_OLLAMA_GGUF_FROM, getGgufFile(ollama))

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

func getBlobsPath() string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_BLOBS_DIR)
}

func getGgufDir(ollama *SOllama) string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_GGUF_DIR, ollama.GetModel())
}

func getGgufFile(ollama *SOllama) string {
	return path.Join(getGgufDir(ollama), path.Base(ollama.GgufFile))
}

func getManifestsPath(ollama *SOllama) string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_MANIFESTS_BASE_PATH, ollama.ModelName, ollama.ModelTag)
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
