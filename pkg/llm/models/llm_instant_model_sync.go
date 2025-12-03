package models

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func (llm *SLLM) getMountedInstantModels(ctx context.Context, probedExt map[string]apis.LLMInternalInstantMdlInfo) (map[string]struct{}, error) {
	container, err := llm.GetLLMSContainer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetSContainer")
	}
	if container.Spec == nil {
		return nil, errors.Wrap(errors.ErrEmpty, "no Spec")
	}
	if len(container.Spec.VolumeMounts) == 0 {
		return nil, errors.Wrap(errors.ErrEmpty, "no VolumeMounts")
	}
	if container.Spec.VolumeMounts[0].Disk == nil {
		return nil, errors.Wrap(errors.ErrEmpty, "no Disk")
	}
	if len(container.Spec.VolumeMounts[0].Disk.PostOverlay) == 0 {
		return nil, nil
	}
	mdlNameToId := make(map[string]string)
	for mdlId, model := range probedExt {
		mdlNameToId[model.Name+":"+model.Tag] = mdlId
	}
	mdlMap := make(map[string]struct{})
	postOverlays := container.Spec.VolumeMounts[0].Disk.PostOverlay
	drv := llm.GetLLMContainerDriver()
	for i := range postOverlays {
		postOverlay := postOverlays[i]
		mdlId := drv.GetInstantModelIdByPostOverlay(postOverlay, mdlNameToId)
		if mdlId != "" {
			mdlMap[mdlId] = struct{}{}
		}
	}
	return mdlMap, nil
}

func (llm *SLLM) getProbedInstantModelsExt(ctx context.Context, userCred mcclient.TokenCredential, instantModelIds ...string) (map[string]apis.LLMInternalInstantMdlInfo, error) {
	drv := llm.GetLLMContainerDriver()
	return drv.GetProbedInstantModelsExt(ctx, userCred, llm, instantModelIds...)
}

type sInstantModelStatus struct {
	apis.LLMInternalInstantMdlInfo

	Probed  bool
	Mounted bool
}

func (llm *SLLM) getProbedMountedInstantModels(ctx context.Context, userCred mcclient.TokenCredential) (map[string]*sInstantModelStatus, error) {
	mdlMap := make(map[string]*sInstantModelStatus)
	probedExt, errExt := llm.getProbedInstantModelsExt(ctx, userCred)
	if errExt != nil {
		return nil, errors.Wrap(errExt, "getProbedInstantModelsExt")
	}

	for modelId := range probedExt {
		mdlMap[modelId] = &sInstantModelStatus{
			LLMInternalInstantMdlInfo: probedExt[modelId],
			Probed:                    true,
		}
	}

	mounted, err := llm.getMountedInstantModels(ctx, probedExt)
	if err != nil {
		return nil, errors.Wrap(err, "llm.getMountedInstantModels")
	}
	for mdlId := range mounted {
		if _, ok := mdlMap[mdlId]; ok {
			mdlMap[mdlId].Mounted = true
		} else {
			mdlMap[mdlId] = &sInstantModelStatus{
				LLMInternalInstantMdlInfo: apis.LLMInternalInstantMdlInfo{
					ModelId: mdlId,
				},
				Mounted: true,
			}
		}
	}
	return mdlMap, nil
}

func (llm *SLLM) uninstallInstantModel(ctx context.Context, userCred mcclient.TokenCredential, mdlId string) error {
	boolFalse := false
	// uninstalled
	probed := &boolFalse
	mounted := &boolFalse
	_, err := GetLLMInstantModelManager().updateInstantModel(ctx, llm.Id, mdlId, "", "", probed, mounted)
	if err != nil {
		return errors.Wrap(err, "uninstallPackage")
	}
	return nil
}

func findInstantModelWithModelInfo(allModels []SLLMInstantModel, mdl apis.ModelInfo) *SLLMInstantModel {
	for i := range allModels {
		if allModels[i].ModelId == mdl.ModelId {
			return &allModels[i]
		}
	}
	return nil
}

func findModelsToUninstall(allModels []SLLMInstantModel, input apis.LLMSyncModelTaskInput) []SLLMInstantModel {
	ret := make([]SLLMInstantModel, 0)
	for i := range input.Models {
		existingModel := findInstantModelWithModelInfo(allModels, input.Models[i])
		if existingModel != nil && existingModel.IsMounted && (input.Method == apis.QuickModelUninstall || input.Method == apis.QuickModelReinstall || (!existingModel.IsProbed && input.Method == apis.QuickModelInstall)) {
			ret = append(ret, *existingModel)
		}
	}
	return ret
}

func findModelsToUnmount(allModels []SLLMInstantModel, input apis.LLMSyncModelTaskInput) []SLLMInstantModel {
	ret := make([]SLLMInstantModel, 0)
	for i := range input.Models {
		existingModel := findInstantModelWithModelInfo(allModels, input.Models[i])
		if existingModel != nil && existingModel.IsMounted && (input.Method == apis.QuickModelUninstall || input.Method == apis.QuickModelReinstall || (!existingModel.IsProbed && input.Method == apis.QuickModelInstall)) {
			ret = append(ret, *existingModel)
		}
	}
	return ret
}

func isImageInUnmountModels(imageId string, mdls []SLLMInstantModel) (bool, error) {
	instMdl, err := GetInstantModelManager().findInstantModelByImageId(imageId)
	if err != nil {
		return false, errors.Wrap(err, "findInstantAppByImageId")
	}
	if instMdl == nil {
		return false, nil
	}
	for i := range mdls {
		if mdls[i].ModelId == instMdl.ModelId {
			return true, nil
		}
	}
	return false, nil
}

func (llm *SLLM) RefreshInstantModels(ctx context.Context, userCred mcclient.TokenCredential, refresh bool) error {
	lockman.LockObject(ctx, llm)
	defer lockman.ReleaseObject(ctx, llm)

	if !llm.LastInstantModelProbe.IsZero() && time.Since(llm.LastInstantModelProbe) < apis.LLM_PROBE_INSTANT_MODEl_INTERVAL_SECOND*time.Second && (!refresh || time.Since(llm.LastInstantModelProbe) < apis.LLM_PROBE_INSTANT_MODEl_INTERVAL_SECOND*time.Second) {
		// already probed, null operation
		return nil
	}

	mdlMap, err := llm.getProbedMountedInstantModels(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "getProbedMountedInstantModels")
	}

	models, err := llm.FetchModels(nil, nil, nil)
	if err != nil {
		return errors.Wrap(err, "FetchModels")
	}

	var errs []error
	for i := range models {
		mdl := models[i]
		var probed *bool
		var mounted *bool
		if status, ok := mdlMap[mdl.ModelId]; ok {
			// probed
			probed = &status.Probed
			mounted = &status.Mounted
			_, err = GetLLMInstantModelManager().updateInstantModel(ctx, llm.Id, mdl.ModelId, status.Name, status.Tag, probed, mounted)
			delete(mdlMap, mdl.ModelId)
		} else {
			// uninstalled
			err = llm.uninstallInstantModel(ctx, userCred, mdl.ModelId)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(mdlMap) > 0 {
		for mdlId, status := range mdlMap {
			_, err := GetLLMInstantModelManager().updateInstantModel(ctx, llm.Id, mdlId, status.Name, status.Tag, &status.Probed, &status.Mounted)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	// update timer
	_, err = db.Update(llm, func() error {
		llm.LastInstantModelProbe = time.Now()
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "update last_instant_model_probe")
	}

	return nil
}

func (llm *SLLM) PerformQuickModels(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.LLMPerformQuickModelsInput) (*apis.LLMBatchPerformOutput, error) {
	if !utils.IsInArray(llm.Status, []string{apis.LLM_STATUS_RUNNING, apis.LLM_STATUS_READY}) {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "llm:%s(%s) status:%s", llm.Name, llm.Id, llm.Status)
	}

	llmStatus := llm.Status
	if len(input.Method) == 0 {
		input.Method = apis.QuickModelInstall
	}

	var toInstallSizeGb float64
	var errs []error
	for i := range input.Models {
		// specified by ID
		if len(input.Models[i].Id) > 0 {
			instModelObj, err := GetInstantModelManager().FetchByIdOrName(ctx, userCred, input.Models[i].Id)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					errs = append(errs, httperrors.NewResourceNotFoundError2(GetInstantModelManager().Keyword(), input.Models[i].Id))
				} else {
					errs = append(errs, errors.Wrap(err, "FetchByIdOrName"))
				}
			} else {
				instApp := instModelObj.(*SInstantModel)
				input.Models[i].Id = instApp.Id
				input.Models[i].ModelId = instApp.ModelId
				input.Models[i].Tag = instApp.Tag
				if input.Method == apis.QuickModelInstall {
					toInstallSizeGb += float64(instApp.GetActualSizeMb()) * 1024 * 1024 / 1000 / 1000 / 1000
				}
			}
		} else {
			mdl, err := GetInstantModelManager().findInstantModel(input.Models[i].ModelId, input.Models[i].Tag, true)
			if err != nil {
				return nil, errors.Wrapf(err, "findInstantModel %s %s", input.Models[i].ModelId, input.Models[i].Tag)
			}
			if mdl == nil {
				errs = append(errs, httperrors.NewResourceNotFoundError2(GetInstantModelManager().Keyword(), input.Models[i].ModelId))
			} else {
				input.Models[i].Id = mdl.Id
				input.Models[i].Tag = mdl.Tag
				input.Models[i].ModelId = mdl.ModelId
			}
		}
	}
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}
	if input.Method == apis.QuickModelInstall {
		if llm.InstantModelQuotaGb > 0 && toInstallSizeGb > float64(llm.InstantModelQuotaGb)-llm.GetTotalInstantModelSizeGb() {
			return nil, errors.Wrapf(httperrors.ErrOutOfQuota, "toInstallSizeGb %f > InstantAppQuotaGb %d - total %f Gb", toInstallSizeGb, llm.InstantModelQuotaGb, llm.GetTotalInstantModelSizeGb())
		}
	}

	task, err := llm.StartLLMInstantModelsSyncTask(ctx, userCred, llmStatus, input, "")
	if err != nil {
		return nil, errors.Wrap(err, "StartLLMInstantModelsSyncTask")
	}

	if input.Method == apis.QuickModelInstall {
		// save pending quota
		llm.insertPendingInstantModelQuota(task.Id, toInstallSizeGb)
	}

	output := apis.LLMBatchPerformOutput{
		Data: []apis.LLMPerformOutput{
			{
				Id:            llm.Id,
				Name:          llm.Name,
				RequestStatus: http.StatusOK,
				TaskId:        task.Id,
			},
		},
		Task: task,
	}
	return &output, nil
}

func (llm *SLLM) StartLLMInstantModelsSyncTask(ctx context.Context, userCred mcclient.TokenCredential, llmStatus string, input apis.LLMPerformQuickModelsInput, parentTaskid string) (*taskman.STask, error) {
	if !utils.IsInArray(llmStatus, []string{apis.LLM_STATUS_RUNNING, apis.LLM_STATUS_READY}) {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "cannot sync models in status %s", llmStatus)
	}
	taskInput := apis.LLMSyncModelTaskInput{
		LLMPerformQuickModelsInput: input,
		LLMStatus:                  llmStatus,
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LLMInstantModelsSyncTask", llm, userCred, jsonutils.Marshal(taskInput).(*jsonutils.JSONDict), parentTaskid, "")
	if err != nil {
		return nil, errors.Wrap(err, "NewTask")
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return nil, errors.Wrap(err, "ScheduleRun")
	}
	return task, nil
}

func (llm *SLLM) FetchModels(isProbed, isMounted, isSystem *bool) ([]SLLMInstantModel, error) {
	q := GetLLMInstantModelManager().Query().Equals("llm_id", llm.Id)
	q = GetLLMInstantModelManager().filterModels(q, isProbed, isMounted, isSystem)

	models := make([]SLLMInstantModel, 0)
	err := db.FetchModelObjects(GetLLMInstantModelManager(), q, &models)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return models, nil
}

func (llm *SLLM) FetchModelsFullName(isProbed, isMounted *bool) ([]string, error) {
	models, err := llm.FetchModels(isProbed, isMounted, nil)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModels")
	}
	mdlFullNames := make([]string, len(models))
	for idx, mdl := range models {
		mdlFullNames[idx] = mdl.ModelName + ":" + mdl.Tag + "-" + mdl.ModelId
	}
	return mdlFullNames, nil
}

func (llm *SLLM) FetchMountedModelFullName() ([]string, error) {
	boolTrue := true
	return llm.FetchModelsFullName(nil, &boolTrue)
}

func (llm *SLLM) RequestUnmountModel(ctx context.Context, userCred mcclient.TokenCredential, input apis.LLMSyncModelTaskInput) ([]string, []*commonapi.ContainerVolumeMountDiskPostOverlay, error) {
	if input.LLMStatus == apis.LLM_STATUS_RUNNING {
		err := llm.RefreshInstantModels(ctx, userCred, true)
		if err != nil {
			return nil, nil, errors.Wrap(err, "RefreshInstantModels")
		}
	}
	allModels, err := llm.FetchModels(nil, nil, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "FetchModels")
	}
	drv := llm.GetLLMContainerDriver()

	if input.LLMStatus == apis.LLM_STATUS_RUNNING {
		uninstallModels := findModelsToUninstall(allModels, input)
		for i := range uninstallModels {
			err := drv.UninstallModel(ctx, userCred, llm, &uninstallModels[i])
			if err != nil {
				log.Errorf("fail to uninstall %s", err)
				continue
			}
		}
	}

	// next found out models that need to unmount
	unmountModels := findModelsToUnmount(allModels, input)
	if len(unmountModels) == 0 {
		return nil, nil, nil
	}

	container, err := llm.GetLLMSContainer(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "GetContainer")
	}

	var unmountOverlays []*commonapi.ContainerVolumeMountDiskPostOverlay
	existingOverlays := container.Spec.VolumeMounts[0].Disk.PostOverlay

	for i := range existingOverlays {
		eOverlay := existingOverlays[i]
		if eOverlay.Image != nil && len(eOverlay.Image.Id) > 0 {
			find, err := isImageInUnmountModels(eOverlay.Image.Id, unmountModels)
			if err != nil {
				return nil, nil, errors.Wrap(err, "isImageInUnmountModels")
			}
			if find {
				unmountOverlays = append(unmountOverlays, eOverlay)
			}

		}
	}

	var modelIds []string
	for i := range unmountModels {
		modelIds = append(modelIds, unmountModels[i].ModelId)
	}
	return modelIds, unmountOverlays, nil
}

func (llm *SLLM) RequestMountModels(ctx context.Context, userCred mcclient.TokenCredential, input apis.LLMSyncModelTaskInput) ([]string, []string, []*commonapi.ContainerVolumeMountDiskPostOverlay, error) {
	if input.LLMStatus == apis.LLM_STATUS_RUNNING {
		err := llm.RefreshInstantModels(ctx, userCred, true)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "RefreshApps")
		}
	}
	existingMdls, err := llm.FetchModels(nil, nil, nil)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "FetchApps")
	}
	log.Debugf("=======RequestMountModels input: %s", jsonutils.Marshal(input).PrettyString())
	models, overlays, err := llm.getMountingModelsPostOverlay(ctx, input, existingMdls)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "getMountingModelsPostOverlay")
	}
	drv := llm.GetLLMContainerDriver()
	var mdlIds []string
	for i := range models {
		model := models[i]
		if input.LLMStatus == apis.LLM_STATUS_RUNNING {
			err := drv.PreInstallModel(ctx, userCred, llm, &model)
			if err != nil {
				log.Errorf("preinstallPackage fail %s", err)
			}
		}
		mdlIds = append(mdlIds, model.ModelId)
	}
	targetDirs := make([]string, 0)
	for i := range overlays {
		if len(overlays[i].ContainerTargetDir) > 0 {
			targetDirs = append(targetDirs, overlays[i].ContainerTargetDir)
		}
	}
	return mdlIds, targetDirs, overlays, nil
}

func (llm *SLLM) TryContainerUnmountPaths(ctx context.Context, userCred mcclient.TokenCredential, s *mcclient.ClientSession, overlays []*commonapi.ContainerVolumeMountDiskPostOverlay, waitSecs int) error {
	start := time.Now()
	for time.Since(start) < time.Second*time.Duration(waitSecs) {
		err := llm.containerUnmountPaths(ctx, userCred, s, overlays)
		if err != nil {
			if strings.Contains(err.Error(), string(errors.ErrInvalidStatus)) {
				// wait
				time.Sleep(5 * time.Second)
			} else {
				return errors.Wrap(err, "containerMountPaths")
			}
		} else {
			// success
			return nil
		}
	}
	return errors.ErrTimeout
}

func (llm *SLLM) containerUnmountPaths(ctx context.Context, userCred mcclient.TokenCredential, s *mcclient.ClientSession, overlays []*commonapi.ContainerVolumeMountDiskPostOverlay) error {
	ctr, err := llm.GetLLMSContainer(ctx)
	if err != nil {
		return errors.Wrap(err, "GetSContainer")
	}

	if !computeapi.ContainerFinalStatus.Has(ctr.Status) {
		return errors.Wrapf(errors.ErrInvalidStatus, "cannot unmount post path in status %s", ctr.Status)
	}

	params := computeapi.ContainerVolumeMountRemovePostOverlayInput{
		Index:       0,
		PostOverlay: overlays,
		UseLazy:     true,
		ClearLayers: true,
	}
	_, err = compute.Containers.PerformAction(s, ctr.Id, "remove-volume-mount-post-overlay", jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrap(err, "PerformAction remove-volume-mount-post-overlay")
	}
	return nil
}

func (llm *SLLM) TryContainerMountPaths(ctx context.Context, userCred mcclient.TokenCredential, s *mcclient.ClientSession, overlays []*commonapi.ContainerVolumeMountDiskPostOverlay, waitSecs int) error {
	start := time.Now()
	for time.Since(start) < time.Second*time.Duration(waitSecs) {
		err := llm.containerMountPaths(ctx, userCred, s, overlays)
		if err != nil {
			if strings.Contains(err.Error(), string(errors.ErrInvalidStatus)) {
				log.Errorf("containerMountPaths error %s, retry", err)
				// retry
				time.Sleep(5 * time.Second)
			} else {
				return errors.Wrap(err, "containerMountPaths")
			}
		} else {
			// success
			return nil
		}
	}
	return errors.ErrTimeout
}

func (llm *SLLM) containerMountPaths(ctx context.Context, userCred mcclient.TokenCredential, s *mcclient.ClientSession, overlays []*commonapi.ContainerVolumeMountDiskPostOverlay) error {
	ctr, err := llm.GetLLMSContainer(ctx)
	if err != nil {
		return errors.Wrap(err, "GetLLMSContainer")
	}

	if !computeapi.ContainerFinalStatus.Has(ctr.Status) {
		return errors.Wrapf(errors.ErrInvalidStatus, "cannot mount post path in status %s", ctr.Status)
	}
	params := computeapi.ContainerVolumeMountAddPostOverlayInput{
		Index:       0,
		PostOverlay: overlays,
	}
	_, err = compute.Containers.PerformAction(s, ctr.Id, "add-volume-mount-post-overlay", jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrap(err, "PerformAction add-volume-mount-post-overlay")
	}
	return nil
}

func (llm *SLLM) MarkInstantModelsUnmounted(ctx context.Context, userCred mcclient.TokenCredential, llmStatus string, mdlIds []string) error {
	return llm.markInstantModelsMounted(ctx, userCred, llmStatus, mdlIds, false)
}

func (llm *SLLM) MarkInstantModelsMounted(ctx context.Context, userCred mcclient.TokenCredential, llmStatus string, mdlIds []string) error {
	return llm.markInstantModelsMounted(ctx, userCred, llmStatus, mdlIds, true)
}

func (llm *SLLM) markInstantModelsMounted(ctx context.Context, userCred mcclient.TokenCredential, llmStatus string, mdlIds []string, mounted bool) error {
	boolFalse := false
	boolTrue := true
	var isProbed *bool
	if !mounted {
		isProbed = &boolFalse
	} else {
		isProbed = &boolTrue
	}

	var errs []error
	for i := range mdlIds {
		_, err := GetLLMInstantModelManager().updateInstantModel(ctx, llm.Id, mdlIds[i], "", "", isProbed, &mounted)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	if llmStatus == apis.LLM_STATUS_RUNNING {
		err := llm.RefreshInstantModels(ctx, userCred, true)
		if err != nil {
			return errors.Wrap(err, "RefreshApps")
		}
	}
	mountedModelsFullName, err := llm.FetchMountedModelFullName()
	if err != nil {
		return errors.Wrap(err, "FetchMountedModelFullName")
	}
	{
		// save mounted apps to volume
		err := llm.UpdateVolumeMountedModelFullNames(mountedModelsFullName)
		if err != nil {
			return errors.Wrap(err, "UpdateVolumeMountedModelFullNames")
		}
	}
	logclient.AddActionLogWithContext(ctx, llm, logclient.ACT_UPDATE, mountedModelsFullName, userCred, true)
	return nil
}

func (llm *SLLM) UpdateVolumeMountedModelFullNames(mdlFullNames []string) error {
	volume, err := llm.GetVolume()
	if err != nil {
		return errors.Wrap(err, "GetVolume")
	}
	return volume.UpdateMountedModelFullNames(mdlFullNames)
}

func (llm *SLLM) getMountingModelsPostOverlay(ctx context.Context, input apis.LLMSyncModelTaskInput, existingMdls []SLLMInstantModel) ([]SLLMInstantModel, []*commonapi.ContainerVolumeMountDiskPostOverlay, error) {
	var models []SLLMInstantModel
	for i := range input.Models {
		if input.Method == apis.QuickModelInstall || input.Method == apis.QuickModelReinstall {
			mdl := input.Models[i]
			if input.Method == apis.QuickModelInstall {
				existingModel := findInstantModelWithModelInfo(existingMdls, mdl)
				if existingModel != nil && (existingModel.IsProbed || existingModel.IsMounted) {
					// if the model is already probed or mounted, skip mount
					continue
				}
			}
			model, err := GetLLMInstantModelManager().updateInstantModel(ctx, llm.Id, mdl.ModelId, mdl.DisplayName, mdl.Tag, nil, nil)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "updateInstantModel %s", mdl.ModelId)
			}
			models = append(models, *model)
		}
	}
	if len(models) == 0 {
		return nil, nil, nil
	}
	drv := llm.GetLLMContainerDriver()
	overlays, err := models2overlays(drv, models, true)
	if err != nil {
		return nil, nil, errors.Wrap(err, "models2overlays")
	}
	return models, overlays, nil
}

func models2overlays(drv ILLMContainerDriver, models []SLLMInstantModel, isInstall bool) ([]*commonapi.ContainerVolumeMountDiskPostOverlay, error) {
	var errs []error
	var allDirs []apis.LLMMountDirInfo
	for i := range models {
		mdlDirs, err := models[i].getMountPaths(isInstall)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		allDirs = append(allDirs, mdlDirs...)
	}
	if len(allDirs) == 0 {
		if len(errs) > 0 {
			return nil, errors.Wrap(errors.NewAggregate(errs), "getMountPaths")
		}
		return nil, nil
	}
	if len(errs) > 0 {
		log.Errorf("models2overlays getMountPaths error %s", errors.NewAggregate(errs))
	}
	var overlays []*commonapi.ContainerVolumeMountDiskPostOverlay
	for i := range allDirs {
		overlay := drv.GetDirPostOverlay(allDirs[i])
		overlays = append(overlays, overlay)
	}
	return overlays, nil
}

func (llm *SLLM) InstallInstantModels(ctx context.Context, userCred mcclient.TokenCredential, dirs []string, mdlIds []string) error {
	drv := llm.GetLLMContainerDriver()
	return drv.InstallModel(ctx, userCred, llm, dirs, mdlIds)
}

func (llm *SLLM) EnsureInstantModelsInstalled(ctx context.Context, userCred mcclient.TokenCredential, mdlIds []string) error {
	mdlMap, err := llm.getProbedInstantModelsExt(ctx, userCred, mdlIds...)
	if err != nil {
		return errors.Wrap(err, "FetchApps")
	}
	var errs []error
	for _, mdlId := range mdlIds {
		if _, ok := mdlMap[mdlId]; ok {
			// probed
			// errs = append(errs, errors.Wrap(errors.ErrInvalidStatus, pkg))
		} else {
			// not mounted and not probed
			errs = append(errs, errors.Wrapf(errors.ErrNotFound, "mdlId %s", mdlId))
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}
