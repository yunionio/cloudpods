package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	commonapis "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemodules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	imagemodules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	commonoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/logclient"

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
			"llm_instant_models_tbl",
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
	ModelName string `width:"128" charset:"ascii" list:"user" create:"required"`
	ModelId   string `width:"128" charset:"ascii" list:"user" create:"optional"`
	Tag       string `width:"64" charset:"ascii" list:"user" create:"required"`

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
	if len(input.Tag) > 0 {
		q = q.In("tag", input.Tag)
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

// func (man *SInstantAppManager) FetchCustomizeColumns(
// 	ctx context.Context,
// 	userCred mcclient.TokenCredential,
// 	query jsonutils.JSONObject,
// 	objs []interface{},
// 	fields stringutils2.SSortedStrings,
// 	isList bool,
// ) []apis.InstantAppDetails {
// 	res := make([]apis.InstantAppDetails, len(objs))

// 	imageIds := make([]string, 0)
// 	mdlNames := make([]string, 0)

// 	virows := man.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
// 	for i := range res {
// 		res[i].SharableVirtualResourceDetails = virows[i]
// 		instApp := objs[i].(*SInstantApp)
// 		if len(instApp.ImageId) > 0 {
// 			imageIds = append(imageIds, instApp.ImageId)
// 		}
// 		if len(instApp.ModelName) > 0 {
// 			mdlNames = append(mdlNames, instApp.ModelName)
// 		}
// 	}

// 	s := auth.GetSession(ctx, userCred, options.Options.Region)
// 	imageMap := make(map[string]imageapi.ImageDetails)
// 	if len(imageIds) > 0 {
// 		params := imageapi.ImageListInput{}
// 		params.Ids = imageIds
// 		params.VirtualResourceListInput.Scope = "max"
// 		details := false
// 		params.Details = &details
// 		limit := len(imageIds)
// 		params.Limit = &limit
// 		params.Field = []string{"id", "name"}
// 		imageList, err := imagemodules.Images.List(s, jsonutils.Marshal(params))
// 		if err != nil {
// 			log.Errorf("list image fail %s", err)
// 		} else {
// 			for i := range imageList.Data {
// 				imgDetails := imageapi.ImageDetails{}
// 				err := imageList.Data[i].Unmarshal(&imgDetails)
// 				if err != nil {
// 					log.Errorf("unmarshal image info %s fail %s", imageList.Data[i], err)
// 				} else {
// 					imageMap[imgDetails.Id] = imgDetails
// 				}
// 			}
// 		}
// 	}
// 	type imageCacheStatus struct {
// 		CachedCount int
// 		CacheCount  int
// 	}
// 	imageCacheStatusTbl := make(map[string]*imageCacheStatus)
// 	if len(imageIds) > 0 {
// 		params := commonoptions.BaseListOptions{}
// 		params.Scope = "max"
// 		params.Filter = []string{fmt.Sprintf("cachedimage_id.in(%s)", strings.Join(imageIds, ","))}
// 		details := false
// 		params.Details = &details
// 		limit := 1024
// 		params.Limit = &limit
// 		params.Field = []string{"storagecache_id", "cachedimage_id", "status"}
// 		offset := -1
// 		total := 0
// 		for offset < 0 || offset < total {
// 			if offset > 0 {
// 				params.Offset = &offset
// 			} else {
// 				offset = 0
// 			}
// 			resp, err := computemodules.Storagecachedimages.List(s, jsonutils.Marshal(params))
// 			if err != nil {
// 				log.Errorf("Storagecachedimages.List fail %s", err)
// 				break
// 			}
// 			for i := range resp.Data {
// 				sci := computeapi.StoragecachedimageDetails{}
// 				err := resp.Data[i].Unmarshal(&sci)
// 				if err != nil {
// 					log.Errorf("unmarshal image info %s fail %s", resp.Data[i], err)
// 				} else {
// 					if _, ok := imageCacheStatusTbl[sci.CachedimageId]; !ok {
// 						imageCacheStatusTbl[sci.CachedimageId] = &imageCacheStatus{}
// 					}
// 					if sci.Status == computeapi.CACHED_IMAGE_STATUS_ACTIVE {
// 						imageCacheStatusTbl[sci.CachedimageId].CachedCount++
// 					}
// 					imageCacheStatusTbl[sci.CachedimageId].CacheCount++
// 				}
// 			}
// 			offset += len(resp.Data)
// 			total = resp.Total
// 		}

// 	}
// 	for i := range res {
// 		instApp := objs[i].(*SInstantApp)
// 		if img, ok := imageMap[instApp.ImageId]; ok {
// 			res[i].Image = img.Name
// 		}
// 		if status, ok := imageCacheStatusTbl[instApp.ImageId]; ok {
// 			res[i].CacheCount = status.CacheCount
// 			res[i].CachedCount = status.CachedCount
// 		}
// 	}
// 	return res
// }

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

	if !apis.IsLLMContainerType(string(input.LLMType)) {
		return input, errors.Wrapf(httperrors.ErrInvalidFormat, "invalid llm_type %s", input.LLMType)
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
			app, err := man.findInstantAppByImageId(img.Id)
			if err != nil {
				return input, errors.Wrap(err, "findInstantAppByImageId")
			}
			if app != nil {
				return input, errors.Wrapf(httperrors.ErrConflict, "image %s has been used by other app", input.ImageId)
			}
		}

		input.ImageId = img.Id
		input.Size = img.Size
		input.Status = img.Status
		input.ActualSizeMb = img.MinDiskMB
	}
	if len(input.Mounts) > 0 {
		drv := man.GetLLMContainerDriver(input.LLMType)
		_, err = drv.ValidateMounts(input.Mounts, input.ModelName, input.Tag)
		if err != nil {
			return input, errors.Wrap(err, "validateMounts")
		}
	}

	input.Enabled = nil
	return input, nil
}

// func (app *SInstantApp) ValidateUpdateData(
// 	ctx context.Context,
// 	userCred mcclient.TokenCredential,
// 	query jsonutils.JSONObject,
// 	input apis.InstantAppUpdateInput,
// ) (apis.InstantAppUpdateInput, error) {
// 	var err error
// 	input.SharableVirtualResourceBaseUpdateInput, err = app.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
// 	if err != nil {
// 		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
// 	}

// 	if len(input.ImageId) > 0 {
// 		img, err := fetchImage(ctx, userCred, input.ImageId)
// 		if err != nil {
// 			return input, errors.Wrapf(err, "fetchImage %s", input.ImageId)
// 		}
// 		if img.DiskFormat != imageapi.IMAGE_DISK_FORMAT_TGZ {
// 			return input, errors.Wrapf(errors.ErrInvalidFormat, "cannot use image as template of format %s", img.DiskFormat)
// 		}

// 		{
// 			findApp, err := GetInstantAppManager().findInstantAppByImageId(img.Id)
// 			if err != nil {
// 				return input, errors.Wrap(err, "findInstantAppByImageId")
// 			}
// 			if findApp != nil && findApp.Id != app.Id {
// 				return input, errors.Wrapf(httperrors.ErrConflict, "image %s has been used by other app", input.ImageId)
// 			}
// 		}

// 		input.ImageId = img.Id
// 		input.Size = img.Size
// 		input.ActualSizeMb = img.MinDiskMB
// 	}
// 	if len(input.Mounts) > 0 {
// 		drv := GetInstantAppManager().GetLLMContainerDriver(apis.LLMContainerType(app.LlmType))
// 		input.Mounts, err = drv.ValidateMounts(input.Mounts, app.Package)
// 		if err != nil {
// 			return input, errors.Wrap(err, "validateMounts")
// 		}
// 		if len(input.Mounts) == 0 {
// 			return input, errors.Wrap(errors.ErrEmpty, "empty mounts")
// 		}
// 	}
// 	return input, nil
// }

func (model *SInstantModel) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	model.syncImagePathMap(ctx, userCred)
}

// func (app *SInstantApp) PostUpdate(
// 	ctx context.Context,
// 	userCred mcclient.TokenCredential,
// 	query jsonutils.JSONObject,
// 	data jsonutils.JSONObject,
// ) {
// 	app.syncImagePathMap(ctx, userCred)
// }

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

func (man *SInstantModelManager) findInstantAppByImageId(imageId string) (*SInstantModel, error) {
	q := man.Query().Equals("image_id", imageId)

	apps := make([]SInstantModel, 0)
	err := db.FetchModelObjects(man, q, &apps)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	if len(apps) == 0 {
		return nil, nil
	}
	return &apps[0], nil
}

func (man *SInstantModelManager) GetInstantAppById(id string) (*SInstantModel, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		return nil, errors.Wrap(err, "FetchById")
	}
	return obj.(*SInstantModel), nil
}

func (man *SInstantModelManager) findInstantApp(mdlId, tag string, isEnabled bool) (*SInstantModel, error) {
	q := man.Query().Equals("model_id", mdlId).Equals("status", imageapi.IMAGE_STATUS_ACTIVE)
	if isEnabled {
		q = q.IsTrue("enabled")
	}
	q = q.Desc("created_at")

	apps := make([]SInstantModel, 0)
	err := db.FetchModelObjects(man, q, &apps)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	if len(apps) == 0 {
		return nil, nil
	}
	if len(tag) > 0 {
		for i := range apps {
			if apps[i].Tag == tag {
				return &apps[i], nil
			}
		}
	}
	return &apps[0], nil
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
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "cannot enable app of status %s", model.Status)
	}
	// check duplicate
	{
		existing, err := GetInstantModelManager().findInstantApp(model.ModelId, model.Tag, true)
		if err != nil {
			return nil, errors.Wrap(err, "findInstantApp")
		}
		if existing != nil && existing.Id != model.Id {
			return nil, errors.Wrapf(errors.ErrDuplicateId, "app of modelId %s tag %s has been enabled", model.ModelId, model.Tag)
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

// func (app *SInstantApp) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
// 	if app.Enabled.IsTrue() {
// 		return errors.Wrap(errors.ErrInvalidStatus, "cannot delete when enabled")
// 	}

// 	for _, man := range []MountedAppModelManager{GetDesktopModelManager(), GetVolumeManager()} {
// 		used, err := man.IsPremountedPackageName(app.Package)
// 		if err != nil {
// 			return errors.Wrap(err, "IsPremountedPackageName")
// 		}
// 		if used {
// 			return errors.Wrap(errors.ErrInvalidStatus, "cannot delete when package is used by other resources")
// 		}
// 	}

// 	return nil
// }

// func (app *SInstantApp) ValidateUpdateCondition(ctx context.Context) error {
// 	if app.Enabled.IsTrue() {
// 		return errors.Wrap(errors.ErrInvalidStatus, "cannot update when enabled")
// 	}
// 	return nil
// }

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

// func (manager *SInstantAppManager) PerformImport(
// 	ctx context.Context,
// 	userCred mcclient.TokenCredential,
// 	query jsonutils.JSONObject,
// 	input apis.InstantAppImportInput,
// ) (*SInstantApp, error) {
// 	if input.Invalid() {
// 		return nil, httperrors.NewInputParameterError("invalid input: %s", jsonutils.Marshal(input).String())
// 	}
// 	// first create a temporary instant-app
// 	tempApp := &SInstantApp{}
// 	tempApp.SetModelManager(manager, &SInstantApp{})
// 	tempApp.Name = fmt.Sprintf("tmp-instant-app-%s.%s", timeutils.CompactTime(time.Now()), utils.GenRequestId(6))
// 	tempApp.Package = "temp"
// 	tempApp.Version = "0.0.1"
// 	tempApp.ProjectId = userCred.GetProjectId()

// 	err := manager.TableSpec().Insert(ctx, tempApp)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "Insert")
// 	}

// 	err = tempApp.startImportTask(ctx, userCred, input)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "startImportTask")
// 	}

// 	return tempApp, nil
// }

// func (instantapp *SInstantApp) startImportTask(ctx context.Context, userCred mcclient.TokenCredential, input apis.InstantAppImportInput) error {
// 	params := jsonutils.NewDict()
// 	params.Add(jsonutils.Marshal(input), "import_input")

// 	task, err := taskman.TaskManager.NewTask(ctx, "InstantAppImportTask", instantapp, userCred, params, "", "")
// 	if err != nil {
// 		return errors.Wrap(err, "NewTask")
// 	}
// 	task.ScheduleRun(nil)

// 	return nil
// }

// func (instantapp *SInstantApp) DoImport(ctx context.Context, userCred mcclient.TokenCredential, input apis.InstantAppImportInput) error {
// 	// first download the image
// 	cfg := objectstore.NewObjectStoreClientConfig(input.Endpoint, input.AccessKey, input.SecretKey)
// 	if len(input.SignVer) > 0 {
// 		cfg.SignVersion(objectstore.S3SignVersion(input.SignVer))
// 	}
// 	minioClient, err := objectstore.NewObjectStoreClient(cfg)
// 	if err != nil {
// 		return errors.Wrap(err, "new minio client")
// 	}

// 	bucket, err := minioClient.GetIBucketByName(input.Bucket)
// 	if err != nil {
// 		return errors.Wrap(err, "GetIBucketByName")
// 	}

// 	tmpDir, err := os.MkdirTemp(options.Options.AdbWorkingDirectory, "instant-app-*")
// 	if err != nil {
// 		return errors.Wrap(err, "CreateTemp")
// 	}
// 	defer func() {
// 		os.RemoveAll(tmpDir)
// 	}()

// 	tmpFileName := filepath.Join(tmpDir, "instant-app.tar")

// 	// download the object
// 	err = func() error {
// 		tmpFile, err := os.Create(tmpFileName)
// 		if err != nil {
// 			return errors.Wrap(err, "Create")
// 		}
// 		defer tmpFile.Close()

// 		_, err = cloudprovider.DownloadObjectParallelWithProgress(ctx, bucket, input.Key, nil, tmpFile, 0, 1024*1024*10, false, 3, func(progress float64, progressMbps float64, totalSizeMb int64) {
// 			log.Infof("DownloadObjectParallelWithProgress progress: %f, progressMbps: %f, totalSizeMb: %d", progress, progressMbps, totalSizeMb)
// 		})
// 		if err != nil {
// 			return errors.Wrap(err, "DownloadObjectParallel")
// 		}
// 		return nil
// 	}()
// 	if err != nil {
// 		return errors.Wrap(err, "download object")
// 	}

// 	// untar the object
// 	err = procutils.NewCommand("tar", "xf", tmpFileName, "-C", tmpDir, "--strip-components=1").Run()
// 	if err != nil {
// 		return errors.Wrapf(err, "untar %s", tmpFileName)
// 	}

// 	scriptPath := filepath.Join(tmpDir, "scripts")
// 	imagePath := filepath.Join(tmpDir, "image")

// 	params, err := decodeParams(scriptPath)
// 	if err != nil {
// 		return errors.Wrap(err, "decodeParams")
// 	}

// 	s := auth.GetSession(ctx, userCred, options.Options.Region)

// 	// upload the image
// 	imageId, err := func() (string, error) {
// 		imgFile, err := os.Open(imagePath)
// 		if err != nil {
// 			return "", errors.Wrap(err, "Open")
// 		}
// 		defer imgFile.Close()

// 		imgFileStat, err := imgFile.Stat()
// 		if err != nil {
// 			return "", errors.Wrap(err, "Stat")
// 		}
// 		imgFileSize := imgFileStat.Size()

// 		imgParams := imageapi.ImageCreateInput{}
// 		imgParams.GenerateName = params.ImageName
// 		imgParams.DiskFormat = "tgz"
// 		imgParams.Size = &imgFileSize
// 		imgParams.Properties = map[string]string{
// 			"os_arch": "aarch64",
// 		}

// 		// upload the image
// 		imageObj, err := imagemodules.Images.Upload(s, jsonutils.Marshal(imgParams), imgFile, imgFileSize)
// 		if err != nil {
// 			return "", errors.Wrap(err, "Create")
// 		}
// 		imageId, err := imageObj.GetString("id")
// 		if err != nil {
// 			return "", errors.Wrap(err, "GetId")
// 		}

// 		return imageId, nil
// 	}()
// 	if err != nil {
// 		return errors.Wrap(err, "upload image")
// 	}

// 	newName, err := db.GenerateAlterName(instantapp, params.InstantAppName)
// 	if err != nil {
// 		return errors.Wrap(err, "GenerateAlterName")
// 	}
// 	// update the instant-app
// 	_, err = db.Update(instantapp, func() error {
// 		instantapp.Name = newName
// 		instantapp.Package = params.Package
// 		instantapp.Version = params.Version
// 		instantapp.ImageId = imageId
// 		instantapp.Mounts = []string{params.AppMount, params.DataMount}
// 		if len(params.MediaMount) > 0 {
// 			instantapp.Mounts = append(instantapp.Mounts, params.MediaMount)
// 		}
// 		instantapp.Status = imageapi.IMAGE_STATUS_SAVING
// 		return nil
// 	})
// 	if err != nil {
// 		return errors.Wrap(err, "update instant-app")
// 	}

// 	// wait image to be active
// 	imgDetails, err := instantapp.WaitImageStatus(ctx, userCred, []string{imageapi.IMAGE_STATUS_ACTIVE}, 1800)
// 	if err != nil {
// 		log.Errorf("WaitImageStatus failed: %s", err)
// 	}

// 	// sync image status
// 	err = instantapp.syncImageStatus(ctx, userCred)
// 	if err != nil {
// 		return errors.Wrap(err, "syncImageStatus")
// 	}

// 	if imgDetails.Status == imageapi.IMAGE_STATUS_KILLED || imgDetails.Status == imageapi.IMAGE_STATUS_DEACTIVATED {
// 		return errors.Wrapf(httperrors.ErrInvalidStatus, "image status: %s", imgDetails.Status)
// 	}

// 	return nil
// }

// type sInstantAppImportParams struct {
// 	ImageName      string
// 	InstantAppName string
// 	Package        string
// 	Version        string
// 	AppMount       string
// 	DataMount      string
// 	MediaMount     string
// }

// func decodeParams(fileDir string) (*sInstantAppImportParams, error) {
// 	content, err := os.ReadFile(fileDir)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "ReadFile")
// 	}
// 	return decodeParamsString(string(content))
// }

// const scriptFormat = `/root/climc image-upload --format tgz --os-arch aarch64 (?P<image_name>.*) \./image
// /root/climc instant-app-create (?P<instant_app_name>.*) (?P<package>.*) (?P<version>.*) \\
// --mounts "(?P<app_mount>.*)" \\
// --mounts "(?P<data_mount>.*)" \\
// (--mounts "(?P<media_mount>.*)")?`

// var (
// 	scriptRE = regexp.MustCompile(scriptFormat)
// )

// func decodeParamsString(content string) (*sInstantAppImportParams, error) {
// 	params := sInstantAppImportParams{}
// 	matches := scriptRE.FindStringSubmatch(content)

// 	log.Debugf("decodeParamsString matches: %v", jsonutils.Marshal(matches))

// 	if len(matches) > 6 {
// 		params.ImageName = matches[1]
// 		params.InstantAppName = matches[2]
// 		params.Package = matches[3]
// 		params.Version = matches[4]
// 		params.AppMount = matches[5]
// 		params.DataMount = matches[6]
// 		if len(matches) > 8 {
// 			params.MediaMount = matches[8]
// 		}
// 	}
// 	return &params, nil
// }

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
