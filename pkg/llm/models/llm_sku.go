package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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
}

type SLLMSku struct {
	SLLMSkuBase
	// SMountedAppsResource

	LLMImageId   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMType      string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMModelName string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
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
	// q, err = man.SMountedAppsResourceManager.ListItemFilter(ctx, q, userCred, input.MountedAppResourceListInput)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "SMountedAppsResourceManager")
	// }
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
	// skuIds := []string{}
	imageIds := []string{}
	// templateIds := []string{}

	skus := []SLLMSku{}
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
	res := make([]api.LLMSkuDetails, len(objs))
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

func (man *SLLMSkuManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	var err error
	input.LLMSKuBaseCreateInput, err = man.SLLMSkuBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMSKuBaseCreateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLLMSkuBaseManager.ValidateCreateData")
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

func (sku *SLLMSku) GetLLMContainerDriver() ILLMContainerDriver {
	return GetLLMContainerDriver(api.LLMContainerType(sku.LLMType))
}

func (sku *SLLMSku) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMSkuUpdateInput) (api.LLMSkuUpdateInput, error) {
	var err error
	input.LLMSkuBaseUpdateInput, err = sku.SLLMSkuBase.ValidateUpdateData(ctx, userCred, query, input.LLMSkuBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate LLMSkuBaseUpdateInput")
	}

	if input.LLMImageId != "" {
		imgObj, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), &input.LLMImageId)
		if err != nil {
			return input, errors.Wrapf(err, "validate image_id %s", input.LLMImageId)
		}
		input.LLMImageId = imgObj.GetId()
	}

	return input, nil
}

func (sku *SLLMSku) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	count, err := GetLLMManager().Query().Equals("llm_sku_id", sku.Id).CountWithError()
	if nil != err {
		return errors.Wrap(err, "fetch llm")
	}
	if count > 0 {
		return errors.Wrap(errors.ErrNotSupported, "This sku is currently in use")
	}
	return nil
}
