package models

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

	var mdlInfo *api.LLMInternalInstantMdlInfo
	modelId := strings.TrimSpace(input.ModelId)
	if modelId == "" {
		return nil, httperrors.NewMissingParameterError("model_id")
	}
	mountDirs := make([]string, 0)
	if len(input.Mounts) > 0 {
		drv, err := GetLLMContainerInstantModelDriver(llm.GetLLMContainerDriver().GetType())
		if err != nil {
			return nil, errors.Wrap(err, "GetLLMContainerInstantModelDriver")
		}
		mountDirs, err = drv.ValidateMounts(input.Mounts, "", "")
		if err != nil {
			return nil, errors.Wrap(err, "validateMounts")
		}
		if len(mountDirs) == 0 {
			return nil, errors.Wrap(errors.ErrEmpty, "empty mounts")
		}
	} else {
		mdlInfos, err := llm.getProbedInstantModelsExt(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "getProbedPackagesExt")
		}

		for _, info := range mdlInfos {
			if info.ModelId == input.ModelId {
				mdlInfo = &info
				break
			}
		}
		if mdlInfo == nil {
			return nil, httperrors.NewBadRequestError("ModelId %s not found", input.ModelId)
		}

		mountDirs, err = llm.detectModelPaths(ctx, userCred, *mdlInfo)
		if err != nil {
			return nil, errors.Wrap(err, "detectModelPaths")
		}
		modelId = mdlInfo.ModelId
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

	drv := llm.GetLLMContainerDriver()
	input, instantModelCreateInput := buildSaveInstantModelCreateInput(drv.GetType(), modelId, input, mdlInfo, mountDirs, time.Now())
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

	drv, err := GetLLMContainerInstantModelDriver(llm.GetLLMContainerDriver().GetType())
	if err != nil {
		return errors.Wrap(err, "GetLLMContainerInstantModelDriver")
	}
	prefix, saveDirs, err := drv.GetSaveDirectories(instantModel)
	if err != nil {
		return errors.Wrap(err, "GetSaveDirectories")
	}

	generateName := strings.TrimSpace(input.Name)
	if generateName == "" {
		generateName = strings.TrimSpace(input.ModelFullName)
	}

	saveImageInput := computeapi.ContainerSaveVolumeMountToImageInput{
		GenerateName:      generateName,
		Notes:             fmt.Sprintf("instance model image for %s(%s)", instantModel.ModelId, instantModel.ModelName+":"+instantModel.ModelTag),
		Index:             getInstantModelSaveVolumeMountIndex(drv),
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

func getInstantModelSaveVolumeMountIndex(drv ILLMContainerInstantModelDriver) int {
	return getInstantModelPostOverlayVolumeMountIndex(drv)
}

func buildSaveInstantModelCreateInput(
	llmType api.LLMContainerType,
	modelId string,
	input api.LLMSaveInstantModelInput,
	mdlInfo *api.LLMInternalInstantMdlInfo,
	mountDirs []string,
	now time.Time,
) (api.LLMSaveInstantModelInput, api.InstantModelCreateInput) {
	modelId = strings.TrimSpace(modelId)
	input.Name = strings.TrimSpace(input.Name)
	input.ModelFullName = strings.TrimSpace(input.ModelFullName)

	if input.ModelFullName == "" {
		if mdlInfo != nil {
			input.ModelFullName = fmt.Sprintf("%s-%s", mdlInfo.Name+":"+mdlInfo.Tag, now.Format("060102"))
		} else {
			input.ModelFullName = fmt.Sprintf("%s-%s", modelId, now.Format("060102"))
		}
	}

	if input.Name == "" {
		input.Name = input.ModelFullName
	}

	modelName, modelTag, _ := parseLargeLanguageModelName(input.ModelFullName)
	if modelName == "" {
		if mdlInfo != nil {
			modelName = mdlInfo.Name
		} else {
			modelName = modelId
		}
	}
	if modelTag == "" {
		if mdlInfo != nil {
			modelTag = mdlInfo.Tag
		} else {
			modelTag = "main"
		}
	}

	instantModelCreateInput := api.InstantModelCreateInput{
		LlmType:   llmType,
		ModelId:   modelId,
		ModelName: modelName,
		ModelTag:  modelTag,
		Mounts:    mountDirs,
	}
	instantModelCreateInput.Name = input.Name
	return input, instantModelCreateInput
}

func parseLargeLanguageModelName(name string) (modelName string, modelTag string, err error) {
	if name == "" {
		return "", "", errors.Wrap(errors.ErrInvalidStatus, "model name is empty")
	}
	parts := strings.Split(name, ":")
	modelName = parts[0]
	modelTag = "latest"
	if len(parts) == 2 {
		modelTag = parts[1]
	}
	return
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
	drv, err := GetLLMContainerInstantModelDriver(llm.GetLLMContainerDriver().GetType())
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMContainerInstantModelDriver")
	}
	return drv.DetectModelPaths(ctx, userCred, llm, pkgInfo)
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
		if resp.StatusCode == http.StatusNotFound {
			return nil, httperrors.NewResourceNotFoundError("url %s not found", url)
		}
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
	return llm.HttpDownloadFileWithProgress(ctx, url, filePath, nil)
}

// HttpDownloadFileWithProgress downloads a file and reports cumulative file progress.
func (llm *SLLM) HttpDownloadFileWithProgress(ctx context.Context, url string, filePath string, callback func(downloaded, total int64)) error {
	client := httputils.GetTimeoutClient(0)
	transport := httputils.GetTransport(true)
	client.Transport = transport

	tmpPath := filePath + ".tmp"
	resumeOffset := fileSize(tmpPath)
	header := http.Header{}
	if resumeOffset > 0 {
		header.Set("Range", fmt.Sprintf("bytes=%d-", resumeOffset))
	}

	resp, err := httputils.Request(client, ctx, httputils.GET, url, header, nil, false)
	if err != nil {
		return errors.Wrap(err, "http request failed")
	}
	defer resp.Body.Close()

	appendDownload := false
	switch resp.StatusCode {
	case http.StatusOK:
		resumeOffset = 0
	case http.StatusPartialContent:
		appendDownload = resumeOffset > 0
	case http.StatusRequestedRangeNotSatisfiable:
		if resumeOffset > 0 && contentRangeTotal(resp.Header.Get("Content-Range")) == resumeOffset {
			notifyDownloadProgress(callback, resumeOffset, resumeOffset)
			if err := os.Rename(tmpPath, filePath); err != nil {
				return errors.Wrapf(err, "failed to rename %s to %s", tmpPath, filePath)
			}
			return nil
		}
		return errors.Errorf("resume range not satisfiable for %s at offset %d", url, resumeOffset)
	default:
		if resp.StatusCode == http.StatusNotFound {
			return errors.Wrapf(httperrors.ErrResourceNotFound, "url %s not found", url)
		}
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	totalSize := responseDownloadTotal(resp, resumeOffset)
	notifyDownloadProgress(callback, resumeOffset, totalSize)

	var out *os.File
	if appendDownload {
		out, err = os.OpenFile(tmpPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to open partial file %s", tmpPath)
		}
	} else {
		out, err = os.Create(tmpPath)
		if err != nil {
			return errors.Wrapf(err, "failed to create file %s", tmpPath)
		}
	}

	writer := io.Writer(out)
	if callback != nil {
		writer = &downloadProgressWriter{
			writer:     out,
			downloaded: resumeOffset,
			total:      totalSize,
			callback:   callback,
		}
	}
	written, err := io.Copy(writer, resp.Body)
	closeErr := out.Close()
	if err != nil {
		return errors.Wrap(err, "failed to write file")
	}
	if closeErr != nil {
		return errors.Wrap(closeErr, "failed to close file")
	}

	log.Infof("Downloaded %d bytes to %s", resumeOffset+written, filePath)

	// rename tmp file to final path
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return errors.Wrapf(err, "failed to rename %s to %s", tmpPath, filePath)
	}

	return nil
}

type downloadProgressWriter struct {
	writer     io.Writer
	downloaded int64
	total      int64
	callback   func(downloaded, total int64)
}

func (w *downloadProgressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n > 0 {
		w.downloaded += int64(n)
		notifyDownloadProgress(w.callback, w.downloaded, w.total)
	}
	return n, err
}

func notifyDownloadProgress(callback func(downloaded, total int64), downloaded, total int64) {
	if callback == nil {
		return
	}
	callback(downloaded, total)
}

func responseDownloadTotal(resp *http.Response, resumeOffset int64) int64 {
	if total := contentRangeTotal(resp.Header.Get("Content-Range")); total > 0 {
		return total
	}
	if resp.ContentLength > 0 {
		return resumeOffset + resp.ContentLength
	}
	return 0
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return 0
	}
	return st.Size()
}

func contentRangeTotal(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return -1
	}
	idx := strings.LastIndex(value, "/")
	if idx < 0 || idx == len(value)-1 {
		return -1
	}
	total := strings.TrimSpace(value[idx+1:])
	if total == "*" {
		return -1
	}
	var ret int64
	for _, r := range total {
		if r < '0' || r > '9' {
			return -1
		}
		ret = ret*10 + int64(r-'0')
	}
	return ret
}
