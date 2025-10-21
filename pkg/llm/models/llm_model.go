package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

func init() {
	GetLLMModelManager()
}

var llmModelManager *SLLMModelManager

func GetLLMModelManager() *SLLMModelManager {
	if llmModelManager != nil {
		return llmModelManager
	}
	llmModelManager = &SLLMModelManager{
		SLLMModelBaseManager: NewSLLMModelBaseManager(
			SLLMModel{},
			"llm_models_tbl",
			"llm_model",
			"llm_models",
		),
	}
	llmModelManager.SetVirtualObject(llmModelManager)
	return llmModelManager
}

type SLLMModelManager struct {
	SLLMModelBaseManager
}

type SLLMModel struct {
	SLLMModelBase
	// SMountedAppsResource

	LLMImageId   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMType      string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMModelName string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (man *SLLMModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LLMModelListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SLLMModelBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SLLMModelBaseManager.ListItemFilter")
	}
	if len(input.LLMType) > 0 {
		q = q.Equals("llm_type", input.LLMType)
	}
	// q, err = man.SMountedAppsResourceManager.ListItemFilter(ctx, q, userCred, input.MountedAppResourceListInput)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "SMountedAppsResourceManager")
	// }
	return q, nil
}

func (manager *SLLMModelManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LLMModelDetails {
	// skuIds := []string{}
	imageIds := []string{}
	// templateIds := []string{}

	skus := []SLLMModel{}
	jsonutils.Update(&skus, objs)
	virows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for _, sku := range skus {
		// skuIds = append(skuIds, sku.Id)
		imageIds = append(imageIds, sku.LLMImageId)
		// if sku.Volumes != nil && len(*sku.Volumes) > 0 && len((*sku.Volumes)[0].TemplateId) > 0 {
		// 	templateIds = append(templateIds, (*sku.Volumes)[0].TemplateId)
		// }
	}

	// q := GetLLMManager().Query().In("llm_model_id", skuIds).GroupBy("llm_model_id")
	// q = q.AppendField(q.Field("llm_model_id"))
	// q = q.AppendField(sqlchemy.COUNT("llm_capacity"))
	// details := []struct {
	// 	LLMModelId  string
	// 	LLMCapacity int
	// }{}
	// q.All(&details)
	res := make([]api.LLMModelDetails, len(objs))
	for i := range skus {
		res[i].SharableVirtualResourceDetails = virows[i]
		// for _, v := range details {
		// 	if v.LLMModelId == sku.Id {
		// 		res[i].LLMCapacity = v.LLMCapacity
		// 		break
		// 	}
		// }
	}
	{
		images := make(map[string]SLLMImage)
		err := db.FetchModelObjectsByIds(GetLLMImageManager(), "id", imageIds, &images)
		if err == nil {
			for i, sku := range skus {
				if image, ok := images[sku.LLMImageId]; ok {
					res[i].Image = image.Name
					res[i].ImageLabel = image.ImageLabel
					res[i].ImageName = image.ImageName
				}
			}
		} else {
			log.Errorf("FetchModelObjectsByIds DesktopImageManager fail %s", err)
		}
	}

	// if len(templateIds) > 0 {
	// 	templates, err := fetchTemplates(ctx, userCred, templateIds)
	// 	if err == nil {
	// 		for i, sku := range skus {
	// 			if templ, ok := templates[(*sku.Volumes)[0].TemplateId]; ok {
	// 				res[i].Template = templ.Name
	// 			}
	// 		}
	// 	} else {
	// 		log.Errorf("fail to retrive image info %s", err)
	// 	}
	// }

	return res
}

func (man *SLLMModelManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMModelCreateInput) (*api.LLMModelCreateInput, error) {
	var err error
	input.LLMModelBaseCreateInput, err = man.SLLMModelBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMModelBaseCreateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLLMModelBaseManager.ValidateCreateData")
	}
	if !api.IsLLMContainerType(input.LLMType) {
		return input, errors.Wrap(httperrors.ErrInputParameter, "llm_type must be one of "+strings.Join(api.LLM_CONTAINER_TYPES.List(), ","))
	}

	_, err = validators.ValidateModel(ctx, userCred, GetLLMImageManager(), &input.LLMImageId)
	if err != nil {
		return input, errors.Wrapf(err, "validate image_id %s", input.LLMImageId)
	}

	input.Status = api.STATUS_READY
	return input, nil
}

func (model *SLLMModel) GetLLMContainerDriver() ILLMContainerDriver {
	return GetLLMContainerDriver(api.LLMContainerType(model.LLMType))
}

func (model *SLLMModel) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	count, err := GetLLMManager().Query().Equals("llm_model_id", model.Id).CountWithError()
	if nil != err {
		return errors.Wrap(err, "fetch llm")
	}
	if count > 0 {
		return errors.Wrap(errors.ErrNotSupported, "This model is currently in use")
	}
	return nil
}
