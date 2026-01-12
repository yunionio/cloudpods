package models

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

func (llm *SLLM) GetDetailsProbedModels(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mdlInfos, err := llm.getProbedInstantModelsExt(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "getProbedPackagesExt")
	}
	return jsonutils.Marshal(mdlInfos), nil
}

func (llm *SLLM) PerformSaveInstantModel(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.LLMSaveInstantModelInput,
) (jsonutils.JSONObject, error) {
	if llm.Status != api.LLM_STATUS_RUNNING {
		return nil, httperrors.NewInvalidStatusError("LLM is not running")
	}

	mdlInfos, err := llm.getProbedInstantModelsExt(ctx, userCred, input.ModelId)
	if err != nil {
		return nil, errors.Wrap(err, "getProbedPackagesExt")
	}

	mdlInfo, ok := mdlInfos[input.ModelId]
	if !ok {
		return nil, httperrors.NewBadRequestError("ModelId %s not found", input.ModelId)
	}

	mountDirs, err := llm.detectModelPaths(ctx, userCred, mdlInfo)
	if err != nil {
		return nil, errors.Wrap(err, "detectModelPaths")
	}

	if len(input.ModelFullName) == 0 {
		input.ModelFullName = fmt.Sprintf("%s-%s", mdlInfo.Name+":"+mdlInfo.Tag, time.Now().Format("060102"))
	}

	var ownerId mcclient.IIdentityProvider
	if len(input.TenantId) > 0 {
		domainId := input.ProjectDomainId
		if len(domainId) == 0 {
			domainId = userCred.GetProjectDomainId()
		} else {
			domain, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, domainId)
			if err != nil {
				return nil, errors.Wrap(err, "TenantCache.FetchDomainByIdOrName")
			}
			domainId = domain.GetId()
		}
		tenant, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, input.TenantId, domainId)
		if err != nil {
			return nil, errors.Wrap(err, "TenantCache.FetchById")
		}
		ownerId = &db.SOwnerId{
			DomainId:  domainId,
			Domain:    tenant.Domain,
			ProjectId: tenant.Id,
			Project:   tenant.Name,
		}
	} else {
		ownerId = userCred
	}

	input.ProjectId = ownerId.GetProjectId()
	input.ProjectDomainId = ownerId.GetProjectDomainId()

	modelName, modelTag, _ := llm.GetLargeLanguageModelName(input.ModelFullName)
	if len(modelName) == 0 {
		modelName = mdlInfo.Name
	}
	if len(modelTag) == 0 {
		modelTag = mdlInfo.Tag
	}

	drv := llm.GetLLMContainerDriver()
	instantModelCreateInput := api.InstantModelCreateInput{
		LlmType:   drv.GetType(),
		ModelId:   mdlInfo.ModelId,
		ModelName: modelName,
		ModelTag:  modelTag,
		Mounts:    mountDirs,
	}
	instantModelCreateInput.Name = input.ModelFullName
	boolTrue := true
	instantModelCreateInput.DoNotImport = &boolTrue
	log.Debugf("instantModelCreateInput: %s", jsonutils.Marshal(instantModelCreateInput))

	instantMdlObj, err := db.DoCreate(GetInstantModelManager(), ctx, userCred, nil, jsonutils.Marshal(instantModelCreateInput), ownerId)
	if err != nil {
		return nil, errors.Wrap(err, "GetInstantModelManager.DoCreate")
	}

	instantMdl := instantMdlObj.(*SInstantModel)

	input.InstantModelId = instantMdl.Id

	_, err = llm.StartSaveModelImageTask(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "StartSaveAppImageTask")
	}

	return jsonutils.Marshal(instantMdl), nil
}

func (llm *SLLM) DoSaveModelImage(ctx context.Context, userCred mcclient.TokenCredential, session *mcclient.ClientSession, input api.LLMSaveInstantModelInput) error {
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_SAVING_MODEL, "DoSaveModelImage")

	instantModelObj, err := GetInstantModelManager().FetchById(input.InstantModelId)
	if err != nil {
		return errors.Wrap(err, "GetInstantModelManager.FetchById")
	}
	instantModel := instantModelObj.(*SInstantModel)

	drv := llm.GetLLMContainerDriver()
	prefix, saveDirs, err := drv.GetSaveDirectories(instantModel)
	if err != nil {
		return errors.Wrap(err, "GetSaveDirectories")
	}

	saveImageInput := computeapi.ContainerSaveVolumeMountToImageInput{
		GenerateName:      input.ModelFullName,
		Notes:             fmt.Sprintf("instance model image for %s(%s)", input.ModelId, instantModel.ModelName+":"+instantModel.ModelTag),
		Index:             0,
		Dirs:              saveDirs,
		UsedByPostOverlay: true,
		DirPrefix:         prefix,
	}

	lc, err := llm.GetLLMContainer()
	if err != nil {
		return errors.Wrap(err, "GetLLMContainer")
	}

	result, err := compute.Containers.PerformAction(session, lc.CmpId, "save-volume-mount-image", jsonutils.Marshal(saveImageInput))
	if err != nil {
		return errors.Wrap(err, "compute.Containers.PerformAction")
	}
	log.Debugf("container save-volume-mount-image result: %s", result)
	saveImageOutput := hostapi.ContainerSaveVolumeMountToImageInput{}
	err = result.Unmarshal(&saveImageOutput)
	if err != nil {
		return errors.Wrap(err, "save-volume-mount-image.result.Unmarshal")
	}

	err = instantModel.saveImageId(ctx, userCred, saveImageOutput.ImageId)
	if err != nil {
		return errors.Wrap(err, "saveImageId")
	}

	return nil
}

func (llm *SLLM) StartSaveModelImageTask(ctx context.Context, userCred mcclient.TokenCredential, input api.LLMSaveInstantModelInput) (*taskman.STask, error) {
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_START_SAVE_MODEL, "StartSaveModelImageTask")

	params := jsonutils.Marshal(input)
	task, err := taskman.TaskManager.NewTask(ctx, "LLMStartSaveModelImageTask", llm, userCred, params.(*jsonutils.JSONDict), "", "")
	if err != nil {
		return nil, errors.Wrap(err, "taskman.TaskManager.NewTask")
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return nil, errors.Wrap(err, "task.ScheduleRun")
	}
	return task, nil
}

func (llm *SLLM) detectModelPaths(ctx context.Context, userCred mcclient.TokenCredential, pkgInfo api.LLMInternalInstantMdlInfo) ([]string, error) {
	return llm.GetLLMContainerDriver().DetectModelPaths(ctx, userCred, llm, pkgInfo)
}

// HttpGet performs a GET request and returns the response body
func (llm *SLLM) HttpGet(ctx context.Context, url string) ([]byte, error) {
	client := httputils.GetTimeoutClient(0)
	transport := httputils.GetTransport(true)
	client.Transport = transport

	resp, err := httputils.Request(client, ctx, httputils.GET, url, http.Header{}, nil, false)
	if err != nil {
		return nil, errors.Wrap(err, "http request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	return body, nil
}

// HttpDownloadFile downloads a file from URL and saves it to the specified path
func (llm *SLLM) HttpDownloadFile(ctx context.Context, url string, filePath string) error {
	client := httputils.GetTimeoutClient(0)
	transport := httputils.GetTransport(true)
	client.Transport = transport

	resp, err := httputils.Request(client, ctx, httputils.GET, url, http.Header{}, nil, false)
	if err != nil {
		return errors.Wrap(err, "http request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// create temporary file first, then rename to avoid partial downloads
	tmpPath := filePath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", tmpPath)
	}

	written, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		return errors.Wrap(err, "failed to write file")
	}

	log.Infof("Downloaded %d bytes to %s", written, filePath)

	// rename tmp file to final path
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return errors.Wrapf(err, "failed to rename %s to %s", tmpPath, filePath)
	}

	return nil
}
