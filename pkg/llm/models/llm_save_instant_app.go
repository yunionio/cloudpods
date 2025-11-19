package models

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

func (llm *SLLM) GetDetailsProbedPackages(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	pkgInfos, err := llm.getProbedPackagesExt(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "getProbedPackagesExt")
	}
	return jsonutils.Marshal(pkgInfos), nil
}

func (llm *SLLM) PerformSaveInstantApp(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.LLMSaveInstantAppInput,
) (jsonutils.JSONObject, error) {
	if llm.Status != api.LLM_STATUS_RUNNING {
		return nil, httperrors.NewInvalidStatusError("LLM is not running")
	}

	pkgInfos, err := llm.getProbedPackagesExt(ctx, userCred, input.PackageName)
	if err != nil {
		return nil, errors.Wrap(err, "getProbedPackagesExt")
	}

	pkgInfo, ok := pkgInfos[input.PackageName]
	if !ok {
		return nil, httperrors.NewBadRequestError("App %s not found", input.PackageName)
	}

	mountDirs, err := llm.detectAppPaths(ctx, userCred, pkgInfo)
	if err != nil {
		return nil, errors.Wrap(err, "detectAppPaths")
	}

	if len(input.ImageName) == 0 {
		input.ImageName = fmt.Sprintf("%s-%s", pkgInfo.Name+":"+pkgInfo.Tag, time.Now().Format("060102"))
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
	instantAppCreateInput := api.InstantAppCreateInput{
		LLMType:   drv.GetType(),
		ModelId:   pkgInfo.ModelId,
		ModelName: pkgInfo.Name,
		Tag:       pkgInfo.Tag,
		Mounts:    mountDirs,
	}
	instantAppCreateInput.Name = input.ImageName
	log.Debugf("instantAppCreateInput: %s", jsonutils.Marshal(instantAppCreateInput))

	instantAppObj, err := db.DoCreate(GetInstantAppManager(), ctx, userCred, nil, jsonutils.Marshal(instantAppCreateInput), ownerId)
	if err != nil {
		return nil, errors.Wrap(err, "InstantAppManager.DoCreate")
	}

	instantApp := instantAppObj.(*SInstantApp)

	input.InstantAppId = instantApp.Id

	_, err = llm.StartSaveAppImageTask(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "StartSaveAppImageTask")
	}

	return jsonutils.Marshal(instantApp), nil
}

func (llm *SLLM) DoSaveAppImage(ctx context.Context, userCred mcclient.TokenCredential, session *mcclient.ClientSession, input api.LLMSaveInstantAppInput) error {
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_SAVING_APP, "DoSaveAppImage")

	instantAppObj, err := GetInstantAppManager().FetchById(input.InstantAppId)
	if err != nil {
		return errors.Wrap(err, "InstantAppManager.FetchById")
	}
	instantApp := instantAppObj.(*SInstantApp)

	drv := llm.GetLLMContainerDriver()
	prefix, saveDirs, err := drv.GetSaveDirectories(instantApp)
	if err != nil {
		return errors.Wrap(err, "GetSaveDirectories")
	}

	saveImageInput := computeapi.ContainerSaveVolumeMountToImageInput{
		GenerateName:      input.ImageName,
		Notes:             fmt.Sprintf("instance app image for %s(%s)", input.PackageName, instantApp.ModelName+":"+instantApp.Tag),
		Index:             0,
		Dirs:              saveDirs,
		UsedByPostOverlay: true,
		DirPrefix:         prefix,
	}

	lc, err := llm.GetLLMContainer()
	if err != nil {
		return errors.Wrap(err, "GetLLMContainer")
	}
	ctr, err := lc.GetSContainer(ctx)
	if err != nil {
		return errors.Wrap(err, "GetSContainer")
	}

	result, err := compute.Containers.PerformAction(session, ctr.Id, "save-volume-mount-image", jsonutils.Marshal(saveImageInput))
	if err != nil {
		return errors.Wrap(err, "compute.Containers.PerformAction")
	}
	log.Debugf("container save-volume-mount-image result: %s", result)
	saveImageOutput := hostapi.ContainerSaveVolumeMountToImageInput{}
	err = result.Unmarshal(&saveImageOutput)
	if err != nil {
		return errors.Wrap(err, "save-volume-mount-image.result.Unmarshal")
	}

	err = instantApp.saveImageId(ctx, userCred, saveImageOutput.ImageId)
	if err != nil {
		return errors.Wrap(err, "saveImageId")
	}

	return nil
}

func (llm *SLLM) StartSaveAppImageTask(ctx context.Context, userCred mcclient.TokenCredential, input api.LLMSaveInstantAppInput) (*taskman.STask, error) {
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_START_SAVE_APP, "StartSaveAppImageTask")

	params := jsonutils.Marshal(input)
	task, err := taskman.TaskManager.NewTask(ctx, "LLMStartSaveAppImageTask", llm, userCred, params.(*jsonutils.JSONDict), "", "")
	if err != nil {
		return nil, errors.Wrap(err, "taskman.TaskManager.NewTask")
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return nil, errors.Wrap(err, "task.ScheduleRun")
	}
	return task, nil
}

func (llm *SLLM) getProbedPackagesExt(ctx context.Context, userCred mcclient.TokenCredential, pkgAppIds ...string) (map[string]api.LLMInternalPkgInfo, error) {
	drv := llm.GetLLMContainerDriver()
	return drv.GetProbedPackagesExt(ctx, userCred, llm, pkgAppIds...)
}

func (llm *SLLM) detectAppPaths(ctx context.Context, userCred mcclient.TokenCredential, pkgInfo api.LLMInternalPkgInfo) ([]string, error) {
	return llm.GetLLMContainerDriver().DetectModelPaths(ctx, userCred, llm, pkgInfo)
}
