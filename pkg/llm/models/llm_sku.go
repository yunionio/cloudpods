package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	imagemodules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	mcclientoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	GetLLMSkuManager()
}

var llmSkuManager *SLLMSkuManager

func GetLLMSkuManager() *SLLMSkuManager {
	if llmSkuManager != nil {
		return llmSkuManager
	}
	llmSkuManager = &SLLMSkuManager{
		SLLMSkuBaseManager: NewSLLMSkuBaseManager(
			SLLMSku{},
			"llm_skus_tbl",
			"llm_sku",
			"llm_skus",
		),
	}
	llmSkuManager.SetVirtualObject(llmSkuManager)
	return llmSkuManager
}

type SLLMSkuManager struct {
	SLLMSkuBaseManager
	SMountedModelsResourceManager
}

type SLLMSku struct {
	SLLMSkuBase
	// SMountedModelsResource

	LLMType string             `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMSpec *api.LLMSpecHolder `length:"long" list:"user" create:"required" update:"user"`
}

func (man *SLLMSkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LLMSkuListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SLLMSkuBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SLLMSkuBaseManager.ListItemFilter")
	}
	if len(input.LLMType) > 0 {
		q = q.Equals("llm_type", input.LLMType)
	}
	q, err = man.SMountedModelsResourceManager.ListItemFilter(ctx, q, userCred, input.MountedModelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SMountedAppsResourceManager")
	}
	return q, nil
}

func (manager *SLLMSkuManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LLMSkuDetails {
	skuIds := []string{}
	imageIds := []string{}
	templateIds := []string{}

	skus := []SLLMSku{}
	jsonutils.Update(&skus, objs)
	virows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for _, sku := range skus {
		skuIds = append(skuIds, sku.Id)
		if imgId := sku.GetLLMImageId(); imgId != "" {
			imageIds = append(imageIds, imgId)
		}
		if sku.Volumes != nil && len(*sku.Volumes) > 0 && len((*sku.Volumes)[0].TemplateId) > 0 {
			templateIds = append(templateIds, (*sku.Volumes)[0].TemplateId)
		}
	}

	q := GetLLMManager().Query().In("llm_sku_id", skuIds).GroupBy("llm_sku_id")
	q = q.AppendField(q.Field("llm_sku_id"))
	q = q.AppendField(sqlchemy.COUNT("llm_capacity"))
	details := []struct {
		LLMSkuId    string
		LLMCapacity int
	}{}
	q.All(&details)
	res := make([]api.LLMSkuDetails, len(objs))
	mountedModelIds := make([]string, 0)
	for i, sku := range skus {
		res[i].SharableVirtualResourceDetails = virows[i]
		for _, v := range details {
			if v.LLMSkuId == sku.Id {
				res[i].LLMCapacity = v.LLMCapacity
				break
			}
		}
		if spec := sku.GetLLMSpecLLM(); spec != nil && len(spec.MountedModels) > 0 {
			mountedModelIds = append(mountedModelIds, spec.MountedModels...)
		}
	}

	// fetch mounted models
	if len(mountedModelIds) > 0 {
		instModels := make(map[string]SInstantModel)
		err := db.FetchModelObjectsByIds(GetInstantModelManager(), "id", mountedModelIds, &instModels)
		if err != nil {
			log.Errorf("FetchModelObjectsByIds InstantModelManager fail %s", err)
		} else {
			for i, sku := range skus {
				modelIds := sku.GetMountedModels()
				if len(modelIds) > 0 {
					res[i].MountedModelDetails = make([]api.MountedModelInfo, 0)
					for _, modelId := range modelIds {
						if instModel, ok := instModels[modelId]; ok {
							info := api.MountedModelInfo{
								Id:       instModel.Id,
								ModelId:  instModel.ModelId,
								FullName: instModel.ModelName + ":" + instModel.ModelTag,
							}
							res[i].MountedModelDetails = append(res[i].MountedModelDetails, info)
						}
					}
				}
			}
		}
	}
	{
		images := make(map[string]SLLMImage)
		err := db.FetchModelObjectsByIds(GetLLMImageManager(), "id", imageIds, &images)
		if err == nil {
			for i, sku := range skus {
				if imgId := sku.GetLLMImageId(); imgId != "" {
					if image, ok := images[imgId]; ok {
						res[i].Image = image.Name
						res[i].ImageLabel = image.ImageLabel
						res[i].ImageName = image.ImageName
					}
				}
			}
		} else {
			log.Errorf("FetchModelObjectsByIds LLMImageManager fail %s", err)
		}
	}

	if len(templateIds) > 0 {
		templates, err := fetchTemplates(ctx, userCred, templateIds)
		if err == nil {
			for i, sku := range skus {
				if templ, ok := templates[(*sku.Volumes)[0].TemplateId]; ok {
					res[i].Template = templ.Name
				}
			}
		} else {
			log.Errorf("fail to retrive image info %s", err)
		}
	}

	return res
}

func (man *SLLMSkuManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	var err error
	input.LLMSKuBaseCreateInput, err = man.SLLMSkuBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMSKuBaseCreateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLLMSkuBaseManager.ValidateCreateData")
	}
	if !api.IsLLMContainerType(input.LLMType) && input.LLMType != string(api.LLM_CONTAINER_DIFY) {
		return input, errors.Wrap(httperrors.ErrInputParameter, "llm_type must be one of "+strings.Join(api.LLM_CONTAINER_TYPES.List(), ","))
	}

	var holder *api.LLMSpecHolder
	switch input.LLMType {
	case string(api.LLM_CONTAINER_OLLAMA), string(api.LLM_CONTAINER_VLLM):
		imgObj, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), &input.LLMImageId)
		if err != nil {
			return input, errors.Wrapf(err, "validate image_id %s", input.LLMImageId)
		}
		llmImage := imgObj.(*SLLMImage)
		if llmImage.LLMType != input.LLMType {
			return input, errors.Wrapf(httperrors.ErrInvalidStatus, "image %s is not of type %s", input.LLMImageId, input.LLMType)
		}
		input.LLMImageId = llmImage.Id

		if input.MountedModels != nil {
			for i, mdl := range input.MountedModels {
				instMdl, err := GetInstantModelManager().FetchByIdOrName(ctx, userCred, mdl)
				if err != nil {
					return input, errors.Wrapf(err, "validate mounted model %s", mdl)
				}
				instantModle := instMdl.(*SInstantModel)
				if instantModle.LlmType != input.LLMType {
					return input, errors.Wrapf(httperrors.ErrInvalidStatus, "mounted model %s is not of type %s", mdl, input.LLMType)
				}
				input.MountedModels[i] = instantModle.GetId()
			}
		}
		holder = &api.LLMSpecHolder{
			Value: &api.LLMSpecLLM{
				Type:          input.LLMType,
				LLMImageId:    input.LLMImageId,
				MountedModels: input.MountedModels,
			},
		}
	case string(api.LLM_CONTAINER_DIFY):
		// Dify type: client must send llm_spec with type "dify" and data (9 image ids)
		if input.LLMSpec == nil || input.LLMSpec.Value == nil {
			return input, errors.Wrap(httperrors.ErrInputParameter, "dify SKU requires llm_spec with type dify and image ids")
		}
		difySpec, ok := input.LLMSpec.Value.(*api.LLMSpecDify)
		if !ok {
			return input, errors.Wrap(httperrors.ErrInputParameter, "dify SKU llm_spec must be LLMSpecDify")
		}
		for _, imgId := range []*string{&difySpec.PostgresImageId, &difySpec.RedisImageId, &difySpec.NginxImageId, &difySpec.DifyApiImageId, &difySpec.DifyPluginImageId, &difySpec.DifyWebImageId, &difySpec.DifySandboxImageId, &difySpec.DifySSRFImageId, &difySpec.DifyWeaviateImageId} {
			_, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), imgId)
			if err != nil {
				return input, errors.Wrapf(err, "validate image_id %s", *imgId)
			}
		}
		holder = input.LLMSpec
	default:
		return input, errors.Wrap(httperrors.ErrInputParameter, "unsupported llm_type "+input.LLMType)
	}
	input.LLMSpec = holder
	input.Status = api.STATUS_READY
	return input, nil
}

// GetLLMSpecLLM returns the LLM-type spec (ollama/vllm) or nil.
func (sku *SLLMSku) GetLLMSpecLLM() *api.LLMSpecLLM {
	if sku.LLMSpec == nil || sku.LLMSpec.Value == nil {
		return nil
	}
	s, ok := sku.LLMSpec.Value.(*api.LLMSpecLLM)
	if !ok {
		return nil
	}
	return s
}

// GetLLMSpecDify returns the Dify-type spec or nil.
func (sku *SLLMSku) GetLLMSpecDify() *api.LLMSpecDify {
	if sku.LLMSpec == nil || sku.LLMSpec.Value == nil {
		return nil
	}
	s, ok := sku.LLMSpec.Value.(*api.LLMSpecDify)
	if !ok {
		return nil
	}
	return s
}

// GetLLMImageId returns the primary image id for this SKU (from LLMSpec for LLM type, or Dify DifyApiImageId, or legacy column).
func (sku *SLLMSku) GetLLMImageId() string {
	if spec := sku.GetLLMSpecLLM(); spec != nil {
		return spec.LLMImageId
	}
	if spec := sku.GetLLMSpecDify(); spec != nil && spec.DifyApiImageId != "" {
		return spec.DifyApiImageId
	}
	return ""
}

// GetMountedModels returns mounted model ids (from LLMSpec for LLM type, or legacy MountedModels column).
func (sku *SLLMSku) GetMountedModels() []string {
	if spec := sku.GetLLMSpecLLM(); spec != nil && len(spec.MountedModels) > 0 {
		return spec.MountedModels
	}
	return nil
}

func (sku *SLLMSku) GetLLMContainerDriver() ILLMContainerDriver {
	return GetLLMContainerDriver(api.LLMContainerType(sku.LLMType))
}

func (sku *SLLMSku) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMSkuUpdateInput) (api.LLMSkuUpdateInput, error) {
	var err error
	input.LLMSkuBaseUpdateInput, err = sku.SLLMSkuBase.ValidateUpdateData(ctx, userCred, query, input.LLMSkuBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate LLMSkuBaseUpdateInput")
	}

	// Build updated LLMSpec from current spec + input
	if sku.LLMSpec == nil || sku.LLMSpec.Value == nil {
		return input, nil
	}
	switch sku.LLMType {
	case string(api.LLM_CONTAINER_OLLAMA), string(api.LLM_CONTAINER_VLLM):
		spec := sku.GetLLMSpecLLM()
		if spec == nil {
			return input, nil
		}
		llmImageId := spec.LLMImageId
		mountedModels := spec.MountedModels
		if input.LLMImageId != "" {
			imgObj, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), &input.LLMImageId)
			if err != nil {
				return input, errors.Wrapf(err, "validate image_id %s", input.LLMImageId)
			}
			llmImage := imgObj.(*SLLMImage)
			if llmImage.LLMType != sku.LLMType {
				return input, errors.Wrapf(httperrors.ErrInvalidStatus, "image %s is not of type %s", input.LLMImageId, sku.LLMType)
			}
			llmImageId = llmImage.Id
			log.Infof("update llm_image_id %s to %s", spec.LLMImageId, llmImageId)
		}
		if input.MountedModels != nil {
			mountedModels = make([]string, len(input.MountedModels))
			for i, mdl := range input.MountedModels {
				instMdl, err := GetInstantModelManager().FetchByIdOrName(ctx, userCred, mdl)
				if err != nil {
					return input, errors.Wrapf(err, "validate mounted model %s", mdl)
				}
				instantModle := instMdl.(*SInstantModel)
				if instantModle.LlmType != sku.LLMType {
					return input, errors.Wrapf(httperrors.ErrInvalidStatus, "mounted model %s is not of type %s", mdl, sku.LLMType)
				}
				mountedModels[i] = instantModle.GetId()
			}
		}
		input.MountedModels = mountedModels
		input.LLMSpec = &api.LLMSpecHolder{
			Value: &api.LLMSpecLLM{
				Type:          sku.LLMType,
				LLMImageId:    llmImageId,
				MountedModels: mountedModels,
			},
		}
	case string(api.LLM_CONTAINER_DIFY):
		// Dify type: accept llm_spec in request; merge non-empty fields into current spec for partial update
		if input.LLMSpec == nil || input.LLMSpec.Value == nil {
			return input, nil
		}
		difySpec, ok := input.LLMSpec.Value.(*api.LLMSpecDify)
		if !ok {
			return input, errors.Wrap(httperrors.ErrInputParameter, "dify SKU llm_spec must be LLMSpecDify")
		}
		currentSpec := sku.GetLLMSpecDify()
		if currentSpec == nil {
			return input, nil
		}
		// Merge with current spec so CLI can update only some image ids
		updated := *currentSpec
		mergeStr := func(dst *string, src string) {
			if src != "" {
				*dst = src
			}
		}
		mergeStr(&updated.PostgresImageId, difySpec.PostgresImageId)
		mergeStr(&updated.RedisImageId, difySpec.RedisImageId)
		mergeStr(&updated.NginxImageId, difySpec.NginxImageId)
		mergeStr(&updated.DifyApiImageId, difySpec.DifyApiImageId)
		mergeStr(&updated.DifyPluginImageId, difySpec.DifyPluginImageId)
		mergeStr(&updated.DifyWebImageId, difySpec.DifyWebImageId)
		mergeStr(&updated.DifySandboxImageId, difySpec.DifySandboxImageId)
		mergeStr(&updated.DifySSRFImageId, difySpec.DifySSRFImageId)
		mergeStr(&updated.DifyWeaviateImageId, difySpec.DifyWeaviateImageId)
		for _, imgId := range []*string{&updated.PostgresImageId, &updated.RedisImageId, &updated.NginxImageId, &updated.DifyApiImageId, &updated.DifyPluginImageId, &updated.DifyWebImageId, &updated.DifySandboxImageId, &updated.DifySSRFImageId, &updated.DifyWeaviateImageId} {
			if *imgId != "" {
				_, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), imgId)
				if err != nil {
					return input, errors.Wrapf(err, "validate image_id %s", *imgId)
				}
			}
		}
		input.LLMSpec = &api.LLMSpecHolder{Value: &updated}
	}

	return input, nil
}

func (sku *SLLMSku) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	count, err := GetLLMManager().Query().Equals("llm_sku_id", sku.Id).CountWithError()
	if err != nil {
		return errors.Wrap(err, "fetch llm")
	}
	if count > 0 {
		return errors.Wrap(errors.ErrNotSupported, "This sku is currently in use by LLM")
	}
	return nil
}

func fetchTemplates(ctx context.Context, userCred mcclient.TokenCredential, templateIds []string) (map[string]imageapi.ImageDetails, error) {
	s := auth.GetSession(ctx, userCred, "")
	params := mcclientoptions.BaseListOptions{}
	params.Id = templateIds
	limit := len(templateIds)
	params.Limit = &limit
	params.Scope = "maxallowed"
	results, err := imagemodules.Images.List(s, jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "Images.List")
	}
	templates := make(map[string]imageapi.ImageDetails)
	for i := range results.Data {
		tmpl := imageapi.ImageDetails{}
		err := results.Data[i].Unmarshal(&tmpl)
		if err == nil {
			templates[tmpl.Id] = tmpl
		}
	}
	return templates, nil
}
