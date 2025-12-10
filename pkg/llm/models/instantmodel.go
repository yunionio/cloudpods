package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	commonapis "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemodules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	imagemodules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	commonoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"

	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/options"
)

var instantModelManager *SInstantModelManager

func init() {
	GetInstantModelManager()
}

type SInstantModelManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

func GetInstantModelManager() *SInstantModelManager {
	if instantModelManager != nil {
		return instantModelManager
	}
	instantModelManager = &SInstantModelManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SInstantModel{},
			"instant_models_tbl",
			"llm_instant_model",
			"llm_instant_models",
		),
	}
	instantModelManager.SetVirtualObject(instantModelManager)
	return instantModelManager
}

type SInstantModel struct {
	db.SSharableVirtualResourceBase
	db.SEnabledResourceBase

	LlmType   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	ModelId   string `width:"128" charset:"ascii" list:"user" create:"optional"`
	ModelName string `width:"128" charset:"ascii" list:"user" create:"required"`
	ModelTag  string `width:"64" charset:"ascii" list:"user" create:"required"`

	ImageId string `width:"128" charset:"ascii" list:"user" create:"optional" update:"user"`

	Mounts []string `charset:"ascii" list:"user" create:"optional" update:"user"`

	Size int64 `nullable:"true" list:"user" create:"optional"`

	ActualSizeMb int32 `nullable:"true" list:"user" update:"user"`

	AutoCache bool `list:"user"`
}

// climc instant-app-list
func (man *SInstantModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.InstantModelListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	q, err = man.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}

	if len(input.ModelName) > 0 {
		q = q.In("model_name", input.ModelName)
	}
	if len(input.ModelTag) > 0 {
		q = q.In("model_tag", input.ModelTag)
	}
	if len(input.ModelId) > 0 {
		q = q.In("model_id", input.ModelId)
	}

	if len(input.Image) > 0 {
		s := auth.GetSession(ctx, userCred, options.Options.Region)
		params := commonoptions.BaseListOptions{}
		params.Scope = "max"
		boolFalse := false
		params.Details = &boolFalse
		limit := 2048
		params.Limit = &limit
		params.Filter = []string{fmt.Sprintf("name.contains(%s)", input.Image)}
		results, err := imagemodules.Images.List(s, jsonutils.Marshal(params))
		if err != nil {
			return nil, errors.Wrap(err, "List")
		}
		imageIds := make([]string, 0)
		for i := range results.Data {
			idstr, _ := results.Data[i].GetString("id")
			imageIds = append(imageIds, idstr)
		}
		q = q.In("image_id", imageIds)
	}

	if len(input.Mounts) > 0 {
		q = q.Contains("mounts", input.Mounts)
	}

	if input.AutoCache != nil {
		q = q.Equals("auto_cache", *input.AutoCache)
	}

	return q, nil
}

func (man *SInstantModelManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.InstantModelDetails {
	res := make([]apis.InstantModelDetails, len(objs))

	imageIds := make([]string, 0)
	mdlIds := make([]string, 0)

	virows := man.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range res {
		res[i].SharableVirtualResourceDetails = virows[i]
		instModel := objs[i].(*SInstantModel)
		if len(instModel.ImageId) > 0 {
			imageIds = append(imageIds, instModel.ImageId)
		}
		if len(instModel.ModelId) > 0 {
			mdlIds = append(mdlIds, instModel.ModelId)
		}
	}

	s := auth.GetSession(ctx, userCred, options.Options.Region)
	imageMap := make(map[string]imageapi.ImageDetails)
	if len(imageIds) > 0 {
		params := imageapi.ImageListInput{}
		params.Ids = imageIds
		params.VirtualResourceListInput.Scope = "max"
		details := false
		params.Details = &details
		limit := len(imageIds)
		params.Limit = &limit
		params.Field = []string{"id", "name"}
		imageList, err := imagemodules.Images.List(s, jsonutils.Marshal(params))
		if err != nil {
			log.Errorf("list image fail %s", err)
		} else {
			for i := range imageList.Data {
				imgDetails := imageapi.ImageDetails{}
				err := imageList.Data[i].Unmarshal(&imgDetails)
				if err != nil {
					log.Errorf("unmarshal image info %s fail %s", imageList.Data[i], err)
				} else {
					imageMap[imgDetails.Id] = imgDetails
				}
			}
		}
	}
	type imageCacheStatus struct {
		CachedCount int
		CacheCount  int
	}
	imageCacheStatusTbl := make(map[string]*imageCacheStatus)
	if len(imageIds) > 0 {
		params := commonoptions.BaseListOptions{}
		params.Scope = "max"
		params.Filter = []string{fmt.Sprintf("cachedimage_id.in(%s)", strings.Join(imageIds, ","))}
		details := false
		params.Details = &details
		limit := 1024
		params.Limit = &limit
		params.Field = []string{"storagecache_id", "cachedimage_id", "status"}
		offset := -1
		total := 0
		for offset < 0 || offset < total {
			if offset > 0 {
				params.Offset = &offset
			} else {
				offset = 0
			}
			resp, err := computemodules.Storagecachedimages.List(s, jsonutils.Marshal(params))
			if err != nil {
				log.Errorf("Storagecachedimages.List fail %s", err)
				break
			}
			for i := range resp.Data {
				sci := computeapi.StoragecachedimageDetails{}
				err := resp.Data[i].Unmarshal(&sci)
				if err != nil {
					log.Errorf("unmarshal image info %s fail %s", resp.Data[i], err)
				} else {
					if _, ok := imageCacheStatusTbl[sci.CachedimageId]; !ok {
						imageCacheStatusTbl[sci.CachedimageId] = &imageCacheStatus{}
					}
					if sci.Status == computeapi.CACHED_IMAGE_STATUS_ACTIVE {
						imageCacheStatusTbl[sci.CachedimageId].CachedCount++
					}
					imageCacheStatusTbl[sci.CachedimageId].CacheCount++
				}
			}
			offset += len(resp.Data)
			total = resp.Total
		}

	}

	llmInstModelQ := GetLLMInstantModelManager().Query().In("model_id", mdlIds).IsFalse("deleted")
	llmInstModels := make([]SLLMInstantModel, 0)
	err := db.FetchModelObjects(GetLLMInstantModelManager(), llmInstModelQ, &llmInstModels)
	if err != nil {
		log.Errorf("fetch llm instant models fail %s", err)
	}

	llmIds := make([]string, 0)
	for i := range llmInstModels {
		if !utils.IsInArray(llmInstModels[i].LlmId, llmIds) {
			llmIds = append(llmIds, llmInstModels[i].LlmId)
		}
	}

	llmMap := make(map[string]SLLM)
	if len(llmIds) > 0 {
		err = db.FetchModelObjectsByIds(GetLLMManager(), "id", llmIds, &llmMap)
		if err != nil {
			log.Errorf("FetchModelObjectsByIds LLMManager fail %s", err)
		}
	}

	modelMountedByMap := make(map[string][]apis.MountedByLLMInfo)
	for i := range llmInstModels {
		llmInstModel := llmInstModels[i]
		llm, ok := llmMap[llmInstModel.LlmId]
		if !ok {
			continue
		}
		info := apis.MountedByLLMInfo{
			LlmId:   llmInstModel.LlmId,
			LlmName: llm.Name,
		}
		if _, ok := modelMountedByMap[llmInstModel.ModelId]; !ok {
			modelMountedByMap[llmInstModel.ModelId] = make([]apis.MountedByLLMInfo, 0)
		}
		modelMountedByMap[llmInstModel.ModelId] = append(modelMountedByMap[llmInstModel.ModelId], info)
	}

	for i := range res {
		instModel := objs[i].(*SInstantModel)
		if img, ok := imageMap[instModel.ImageId]; ok {
			res[i].Image = img.Name
		}
		if status, ok := imageCacheStatusTbl[instModel.ImageId]; ok {
			res[i].CacheCount = status.CacheCount
			res[i].CachedCount = status.CachedCount
		}
		if mountedBy, ok := modelMountedByMap[instModel.ModelId]; ok {
			res[i].MountedByLLMs = mountedBy
		}
	}
	return res
}

func (man *SInstantModelManager) GetLLMContainerDriver(llmType apis.LLMContainerType) ILLMContainerDriver {
	return GetLLMContainerDriver(llmType)
}

func (man *SInstantModelManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.InstantModelCreateInput,
) (apis.InstantModelCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}

	if !apis.IsLLMContainerType(string(input.LlmType)) {
		return input, errors.Wrapf(httperrors.ErrInvalidFormat, "invalid llm_type %s", input.LlmType)
	}

	if len(input.ImageId) > 0 {
		img, err := fetchImage(ctx, userCred, input.ImageId)
		if err != nil {
			return input, errors.Wrapf(err, "fetchImage %s", input.ImageId)
		}
		if img.DiskFormat != imageapi.IMAGE_DISK_FORMAT_TGZ {
			return input, errors.Wrapf(errors.ErrInvalidFormat, "cannot use image as template of format %s", img.DiskFormat)
		}

		{
			mdl, err := man.findInstantModelByImageId(img.Id)
			if err != nil {
				return input, errors.Wrap(err, "findInstantModelByImageId")
			}
			if mdl != nil {
				return input, errors.Wrapf(httperrors.ErrConflict, "image %s has been used by other model", input.ImageId)
			}
		}

		input.ImageId = img.Id
		input.Size = img.Size
		input.Status = img.Status
		input.ActualSizeMb = img.MinDiskMB
	}
	if len(input.Mounts) > 0 {
		drv := man.GetLLMContainerDriver(input.LlmType)
		_, err = drv.ValidateMounts(input.Mounts, input.ModelName, input.ModelTag)
		if err != nil {
			return input, errors.Wrap(err, "validateMounts")
		}
	}

	input.Enabled = nil
	return input, nil
}

func (model *SInstantModel) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.InstantModelUpdateInput,
) (apis.InstantModelUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = model.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}

	if len(input.ImageId) > 0 {
		img, err := fetchImage(ctx, userCred, input.ImageId)
		if err != nil {
			return input, errors.Wrapf(err, "fetchImage %s", input.ImageId)
		}
		if img.DiskFormat != imageapi.IMAGE_DISK_FORMAT_TGZ {
			return input, errors.Wrapf(errors.ErrInvalidFormat, "cannot use image as template of format %s", img.DiskFormat)
		}

		{
			findModel, err := GetInstantModelManager().findInstantModelByImageId(img.Id)
			if err != nil {
				return input, errors.Wrap(err, "findInstantModelByImageId")
			}
			if findModel != nil && findModel.Id != model.Id {
				return input, errors.Wrapf(httperrors.ErrConflict, "image %s has been used by other model", input.ImageId)
			}
		}

		input.ImageId = img.Id
		input.Size = img.Size
		input.ActualSizeMb = img.MinDiskMB
	}
	if len(input.Mounts) > 0 {
		drv := GetInstantModelManager().GetLLMContainerDriver(apis.LLMContainerType(model.LlmType))
		input.Mounts, err = drv.ValidateMounts(input.Mounts, model.ModelName, model.ModelTag)
		if err != nil {
			return input, errors.Wrap(err, "validateMounts")
		}
		if len(input.Mounts) == 0 {
			return input, errors.Wrap(errors.ErrEmpty, "empty mounts")
		}
	}
	return input, nil
}

func (model *SInstantModel) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	model.syncImagePathMap(ctx, userCred)
	input := apis.InstantModelCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return
	}
	if input.ImageId == "" && (input.DoNotImport == nil || !*input.DoNotImport) {
		model.startImportTask(ctx, userCred, apis.InstantModelImportInput{
			LlmType:   input.LlmType,
			ModelName: input.ModelName,
			ModelTag:  input.ModelTag,
		})
	}
}

func (model *SInstantModel) PostUpdate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	model.syncImagePathMap(ctx, userCred)
}

func (model *SInstantModel) getImagePaths() map[string]string {
	drv := GetInstantModelManager().GetLLMContainerDriver(apis.LLMContainerType(model.LlmType))
	return drv.GetImageInternalPathMounts(model)
}

func (model *SInstantModel) syncImagePathMap(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(model.ImageId) == 0 {
		return nil
	}
	imgPaths := model.getImagePaths()
	if len(imgPaths) == 0 {
		return nil
	}
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	params := imageapi.ImageUpdateInput{
		Properties: map[string]string{
			"internal_path_map":    jsonutils.Marshal(imgPaths).String(),
			"used_by_post_overlay": "true",
		},
	}
	_, err := imagemodules.Images.Update(s, model.ImageId, jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}

func (model *SInstantModel) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.InstantModelSyncstatusInput) (jsonutils.JSONObject, error) {
	err := model.syncImageStatus(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "syncImageStatus")
	}
	return nil, nil
}

func (model *SInstantModel) saveImageId(ctx context.Context, userCred mcclient.TokenCredential, imageId string) error {
	_, err := db.Update(model, func() error {
		model.ImageId = imageId
		return nil
	})
	if err != nil {
		logclient.AddActionLogWithContext(ctx, model, logclient.ACT_SAVE_IMAGE, err, userCred, false)
		return errors.Wrap(err, "update image_id")
	}
	logclient.AddActionLogWithContext(ctx, model, logclient.ACT_SAVE_IMAGE, imageId, userCred, true)
	return nil
}

func (model *SInstantModel) syncImageStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	img, err := fetchImage(ctx, userCred, model.ImageId)
	if err != nil {
		if httputils.ErrorCode(err) == 404 {
			model.SetStatus(ctx, userCred, imageapi.IMAGE_STATUS_DELETED, "not found")
			return nil
		}
		return errors.Wrapf(err, "fetchImage %s", model.ImageId)
	}
	model.SetStatus(ctx, userCred, img.Status, "syncStatus")
	if img.Status == imageapi.IMAGE_STATUS_ACTIVE && (model.Size != img.Size || model.ActualSizeMb != img.MinDiskMB) {
		_, err := db.Update(model, func() error {
			model.Size = img.Size
			model.ActualSizeMb = img.MinDiskMB
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update size")
		}
	}
	{
		err := model.syncImagePathMap(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "syncImagePathMap")
		}
	}
	return nil
}

func (man *SInstantModelManager) findInstantModelByImageId(imageId string) (*SInstantModel, error) {
	q := man.Query().Equals("image_id", imageId)

	mdls := make([]SInstantModel, 0)
	err := db.FetchModelObjects(man, q, &mdls)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	if len(mdls) == 0 {
		return nil, nil
	}
	return &mdls[0], nil
}

func (man *SInstantModelManager) GetInstantModelById(id string) (*SInstantModel, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		return nil, errors.Wrap(err, "FetchById")
	}
	return obj.(*SInstantModel), nil
}

func (man *SInstantModelManager) findInstantModel(mdlId, tag string, isEnabled bool) (*SInstantModel, error) {
	q := man.Query().Equals("model_id", mdlId).Equals("status", imageapi.IMAGE_STATUS_ACTIVE)
	if isEnabled {
		q = q.IsTrue("enabled")
	}
	q = q.Desc("created_at")

	mdls := make([]SInstantModel, 0)
	err := db.FetchModelObjects(man, q, &mdls)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	if len(mdls) == 0 {
		return nil, nil
	}
	if len(tag) > 0 {
		for i := range mdls {
			if mdls[i].ModelTag == tag {
				return &mdls[i], nil
			}
		}
	}
	return &mdls[0], nil
}

func (model *SInstantModel) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input commonapis.PerformEnableInput,
) (jsonutils.JSONObject, error) {
	if len(model.ImageId) == 0 {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "empty image_id")
	}
	if len(model.Mounts) == 0 {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "empty mounts")
	}
	{
		err := model.syncImageStatus(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "syncImageStatus")
		}
	}
	if model.Status != imageapi.IMAGE_STATUS_ACTIVE {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "cannot enable model of status %s", model.Status)
	}
	// check duplicate
	{
		existing, err := GetInstantModelManager().findInstantModel(model.ModelId, model.ModelTag, true)
		if err != nil {
			return nil, errors.Wrap(err, "findInstantModel")
		}
		if existing != nil && existing.Id != model.Id {
			return nil, errors.Wrapf(errors.ErrDuplicateId, "model of modelId %s tag %s has been enabled", model.ModelId, model.ModelTag)
		}
	}
	_, err := db.Update(model, func() error {
		model.SEnabledResourceBase.SetEnabled(true)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "update")
	}
	return nil, nil
}

func (model *SInstantModel) PerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input commonapis.PerformDisableInput,
) (jsonutils.JSONObject, error) {
	_, err := db.Update(model, func() error {
		model.SEnabledResourceBase.SetEnabled(false)
		if model.AutoCache {
			model.AutoCache = false
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "update")
	}
	return nil, nil
}

func (model *SInstantModel) PerformChangeOwner(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input commonapis.PerformChangeProjectOwnerInput,
) (jsonutils.JSONObject, error) {
	// perform disk change owner
	if len(model.ImageId) > 0 {
		s := auth.GetSession(ctx, userCred, options.Options.Region)
		_, err := imagemodules.Images.PerformAction(s, model.ImageId, "change-owner", jsonutils.Marshal(input))
		if err != nil {
			return nil, errors.Wrap(err, "image change-owner")
		}
	}
	return model.SSharableVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, input)
}

func (model *SInstantModel) PerformPublic(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input commonapis.PerformPublicProjectInput,
) (jsonutils.JSONObject, error) {
	if len(model.ImageId) > 0 {
		s := auth.GetSession(ctx, userCred, options.Options.Region)
		_, err := imagemodules.Images.PerformAction(s, model.ImageId, "public", jsonutils.Marshal(input))
		if err != nil {
			return nil, errors.Wrap(err, "image public")
		}
	}
	return model.SSharableVirtualResourceBase.PerformPublic(ctx, userCred, query, input)
}

func (model *SInstantModel) PerformPrivate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input commonapis.PerformPrivateInput,
) (jsonutils.JSONObject, error) {
	if len(model.ImageId) > 0 {
		s := auth.GetSession(ctx, userCred, options.Options.Region)
		_, err := imagemodules.Images.PerformAction(s, model.ImageId, "private", jsonutils.Marshal(input))
		if err != nil {
			return nil, errors.Wrap(err, "image private")
		}
	}
	return model.SSharableVirtualResourceBase.PerformPrivate(ctx, userCred, query, input)
}

func (model *SInstantModel) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if model.Enabled.IsTrue() {
		for _, man := range []MountedModelModelManager{GetLLMSkuManager(), GetVolumeManager()} {
			used, err := man.IsPremountedModelName(model.ModelName + ":" + model.ModelTag + "-" + model.ModelId)
			if err != nil {
				return errors.Wrap(err, "IsPremountedModelName")
			}
			if used {
				return errors.Wrap(errors.ErrInvalidStatus, "cannot delete when model is used by other resources")
			}
		}
	}

	return nil
}

func (model *SInstantModel) ValidateUpdateCondition(ctx context.Context) error {
	if model.Enabled.IsTrue() {
		return errors.Wrap(errors.ErrInvalidStatus, "cannot update when enabled")
	}
	return nil
}

func (model *SInstantModel) PerformEnableAutoCache(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.InstantModelEnableAutoCacheInput,
) (jsonutils.JSONObject, error) {
	if input.AutoCache && model.Enabled.IsFalse() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot enable auto_cache for disabled app")
	}
	_, err := db.Update(model, func() error {
		model.AutoCache = input.AutoCache
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "update auto_cache")
	}
	if model.AutoCache {
		err := model.doCache(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "doCache")
		}
	}
	return nil, nil
}

func (model *SInstantModel) doCache(ctx context.Context, userCred mcclient.TokenCredential) error {
	input := computeapi.CachedImageManagerCacheImageInput{}
	input.ImageId = model.ImageId
	input.AutoCache = true
	input.HostType = []string{computeapi.HOST_TYPE_CONTAINER}
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	_, err := computemodules.Cachedimages.PerformClassAction(s, "cache-image", jsonutils.Marshal(input))
	if err != nil {
		return errors.Wrap(err, "PerformClassAction cache-image")
	}
	return nil
}

func (man *SInstantModelManager) PerformImport(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.InstantModelImportInput,
) (*SInstantModel, error) {
	// first create a temporary instant-app
	tempModel := &SInstantModel{}
	tempModel.SetModelManager(man, &SInstantModel{})
	tempModel.Name = fmt.Sprintf("tmp-instant-model-%s.%s", time.Now().Format("060102"), utils.GenRequestId(6))
	tempModel.ModelName = input.ModelName
	tempModel.ModelTag = input.ModelTag
	tempModel.LlmType = string(input.LlmType)
	tempModel.ProjectId = userCred.GetProjectId()

	err := man.TableSpec().Insert(ctx, tempModel)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	err = tempModel.startImportTask(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "startImportTask")
	}

	return tempModel, nil
}

func (model *SInstantModel) startImportTask(ctx context.Context, userCred mcclient.TokenCredential, input apis.InstantModelImportInput) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.Marshal(input), "import_input")

	task, err := taskman.TaskManager.NewTask(ctx, "LLMInstantModelImportTask", model, userCred, params, "", "")
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)

	return nil
}

func (model *SInstantModel) DoImport(ctx context.Context, userCred mcclient.TokenCredential, s *mcclient.ClientSession, input apis.InstantModelImportInput) (tmpDir string, err error) {
	// ensure LLMWorkingDirectory exists
	if err = os.MkdirAll(options.Options.LLMWorkingDirectory, 0755); err != nil {
		err = errors.Wrap(err, "MkdirAll LLMWorkingDirectory")
		return
	}
	// create temp directory for download
	tmpDir, err = os.MkdirTemp(options.Options.LLMWorkingDirectory, "instant-model-*")
	if err != nil {
		err = errors.Wrap(err, "CreateTemp")
		return
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	drv := GetInstantModelManager().GetLLMContainerDriver(input.LlmType)

	// download model from registry
	modelId, mounts, err := drv.DownloadModel(ctx, userCred, nil, tmpDir, input.ModelName, input.ModelTag)
	if err != nil {
		err = errors.Wrap(err, "DownloadModel")
		return
	}
	log.Infof("Downloaded model %s:%s with modelId: %s to %s", input.ModelName, input.ModelTag, modelId, tmpDir)

	// create tar.gz archive from downloaded files
	imagePath := fmt.Sprintf("%s/model.tgz", tmpDir)
	if err = createTarGz(tmpDir, imagePath); err != nil {
		err = errors.Wrap(err, "createTarGz")
		return
	}

	// upload the image
	imageId, err := func() (string, error) {
		imgFile, err := os.Open(imagePath)
		if err != nil {
			return "", errors.Wrap(err, "Open")
		}
		defer imgFile.Close()

		imgFileStat, err := imgFile.Stat()
		if err != nil {
			return "", errors.Wrap(err, "Stat")
		}
		imgFileSize := imgFileStat.Size()

		imgParams := imageapi.ImageCreateInput{}
		imgParams.GenerateName = fmt.Sprintf("%s-%s", input.ModelName, input.ModelTag)
		imgParams.DiskFormat = "tgz"
		imgParams.Size = &imgFileSize
		imgParams.Properties = map[string]string{
			"llm_type":   string(input.LlmType),
			"model_name": input.ModelName,
			"model_tag":  input.ModelTag,
			"model_id":   modelId,
		}

		// upload the image
		imageObj, err := imagemodules.Images.Upload(s, jsonutils.Marshal(imgParams), imgFile, imgFileSize)
		if err != nil {
			return "", errors.Wrap(err, "Upload Image")
		}
		imageId, err := imageObj.GetString("id")
		if err != nil {
			return "", errors.Wrap(err, "Get Image Id")
		}

		return imageId, nil
	}()
	if err != nil {
		err = errors.Wrap(err, "upload image")
		return
	}

	// update the instant-model
	_, err = db.Update(model, func() error {
		// model.LlmType = string(input.LlmType)
		// model.ModelName = input.ModelName
		// model.Tag = input.ModelTag
		model.ModelId = modelId
		model.ImageId = imageId
		model.Mounts = mounts
		model.Status = imageapi.IMAGE_STATUS_SAVING
		return nil
	})
	if err != nil {
		err = errors.Wrap(err, "update instant-model")
		return
	}

	// wait image to be active
	imgDetails, err := model.WaitImageStatus(ctx, userCred, []string{imageapi.IMAGE_STATUS_ACTIVE}, 1800)
	if err != nil {
		log.Errorf("WaitImageStatus failed: %s", err)
	}

	// sync image status
	err = model.syncImageStatus(ctx, userCred)
	if err != nil {
		err = errors.Wrap(err, "syncImageStatus")
		return
	}

	if imgDetails.Status == imageapi.IMAGE_STATUS_KILLED || imgDetails.Status == imageapi.IMAGE_STATUS_DEACTIVATED {
		err = errors.Wrapf(httperrors.ErrInvalidStatus, "image status: %s", imgDetails.Status)
		return
	}

	return
}

// createTarGz creates a tar.gz archive from the source directory
func createTarGz(srcDir string, dstPath string) error {
	// use -C to change directory, . to pack all contents
	// --exclude to exclude the output file itself (if in the same directory)
	dstBase := filepath.Base(dstPath)
	output, err := procutils.NewCommand("tar", "-czvf", dstPath, "-C", srcDir, "--exclude", dstBase, ".").Output()
	if err != nil {
		return errors.Wrapf(err, "tar -czvf %s -C %s: %s", dstPath, srcDir, output)
	}
	return nil
}

func (model *SInstantModel) GetImage(ctx context.Context, userCred mcclient.TokenCredential) (*imageapi.ImageDetails, error) {
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	imageObj, err := imagemodules.Images.Get(s, model.ImageId, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Get")
	}
	imgDetail := imageapi.ImageDetails{}
	err = imageObj.Unmarshal(&imgDetail)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &imgDetail, nil
}

func (model *SInstantModel) WaitImageStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*imageapi.ImageDetails, error) {
	expire := time.Now().Add(time.Second * time.Duration(timeoutSecs))
	for time.Now().Before(expire) {
		img, err := model.GetImage(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "GetImage")
		}
		if utils.IsInArray(img.Status, targetStatus) {
			return img, nil
		}
		if strings.Contains(img.Status, "fail") || img.Status == imageapi.IMAGE_STATUS_KILLED || img.Status == imageapi.IMAGE_STATUS_DEACTIVATED {
			return nil, errors.Wrap(errors.ErrInvalidStatus, img.Status)
		}
		time.Sleep(2 * time.Second)
	}
	return nil, errors.Wrapf(httperrors.ErrTimeout, "wait image status %s timeout", targetStatus)
}

func (model *SInstantModel) GetActualSizeMb() int32 {
	if model.ActualSizeMb > 0 {
		return model.ActualSizeMb
	}
	return int32(model.Size / 1024 / 1024)
}

func (model *SInstantModel) CleanupImportTmpDir(ctx context.Context, userCred mcclient.TokenCredential, tmpDir string) error {
	// sync image status
	err := model.syncImageStatus(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "syncImageStatus")
	}
	if tmpDir == "" {
		return nil
	}
	log.Infof("Cleaning up tmpDir: %s", tmpDir)
	if err := procutils.NewCommand("rm", "-rf", tmpDir).Run(); err != nil {
		return errors.Wrapf(err, "Failed to remove tmpDir %s", tmpDir)
	}
	return nil
}
