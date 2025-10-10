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

}

var llmModelManager *SLLMModelManager

func GetLLMModelManager() *SLLMModelManager {
	if llmModelManager != nil {
		return llmModelManager
	}
	llmModelManager = &SLLMModelManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
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
	db.SSharableVirtualResourceBaseManager
	// SMountedAppsResourceManager
}

type SLLMModel struct {
	db.SSharableVirtualResourceBase
	// SMountedAppsResource

	LLMImageId   string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMType      string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMModelName string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	BandwidthMb  int               `nullable:"false" default:"0" create:"optional" list:"user" update:"user"`
	Cpu          int               `nullable:"false" default:"1" create:"optional" list:"user" update:"user"`
	Memory       int               `nullable:"false" default:"512" create:"optional" list:"user" update:"user"`
	Volumes      *api.Volumes      `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	PortMappings *api.PortMappings `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	Devices      *api.Devices      `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	Envs         *api.Envs         `charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
	// Properties
	Properties map[string]string `charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
}

func (man *SLLMModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LLMModelListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableBaseResourceManager.ListItemFilter")
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
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}
	if input.Cpu <= 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "cpu must > 0")
	}
	if input.Memory <= 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "mem must > 0")
	}
	if input.Volumes == nil {
		return input, errors.Wrap(httperrors.ErrInputParameter, "volumes cannot be empty")
	}
	if !api.IsLLMContainerType(input.LLMType) {
		return input, errors.Wrap(httperrors.ErrInputParameter, "llm_type must be one of "+strings.Join(api.LLM_CONTAINER_TYPES.List(), ","))
	}

	_, err = validators.ValidateModel(ctx, userCred, GetLLMImageManager(), &input.LLMImageId)
	if err != nil {
		return input, errors.Wrapf(err, "validate image_id %s", input.LLMImageId)
	}

	input.Status = api.LLM_STATUS_READY
	return input, nil
}
