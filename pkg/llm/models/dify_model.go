package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	GetDifyModelManager()
}

var difyModelManager *SDifyModelManager

func GetDifyModelManager() *SDifyModelManager {
	if difyModelManager != nil {
		return difyModelManager
	}
	difyModelManager = &SDifyModelManager{
		SLLMModelBaseManager: NewSLLMModelBaseManager(
			SDifyModel{},
			"dify_models_tbl",
			"dify_model",
			"dify_models",
		),
	}
	difyModelManager.SetVirtualObject(difyModelManager)
	return difyModelManager
}

type SDifyModelManager struct {
	SLLMModelBaseManager
}

type SDifyModel struct {
	SLLMModelBase

	PostgresImageId     string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	RedisImageId        string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	NginxImageId        string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifyApiImageId      string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifyPluginImageId   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifyWebImageId      string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifySandboxImageId  string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifySSRFImageId     string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifyWeaviateImageId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (man *SDifyModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.DifyModelListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SLLMModelBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SLLMModelBaseManager.ListItemFilter")
	}
	return q, nil
}

// func (man *SDifyModelManager) FetchCustomizeColumns(
// 	ctx context.Context,
// 	userCred mcclient.TokenCredential,
// 	query jsonutils.JSONObject,
// 	objs []interface{},
// 	fields stringutils2.SSortedStrings,
// 	isList bool,
// ) []api.LLMModelDetails {

// }

func (man *SDifyModelManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.DifyModelCreateInput) (*api.DifyModelCreateInput, error) {
	var err error
	input.LLMModelBaseCreateInput, err = man.SLLMModelBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMModelBaseCreateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLLMModelBaseManager.ValidateCreateData")
	}

	for _, imgId := range []*string{&input.PostgresImageId, &input.RedisImageId, &input.NginxImageId, &input.DifyApiImageId, &input.DifyPluginImageId, &input.DifyWebImageId, &input.DifySandboxImageId, &input.DifySSRFImageId, &input.DifyWeaviateImageId} {
		_, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), imgId)
		if err != nil {
			return input, errors.Wrapf(err, "validate image_id %s", *imgId)
		}
	}

	return input, nil
}

func (model *SDifyModel) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DifyModelUpdateInput) (api.DifyModelUpdateInput, error) {
	var err error
	input.LLMModelBaseUpdateInput, err = model.SLLMModelBase.ValidateUpdateData(ctx, userCred, query, input.LLMModelBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate LLMModelBaseUpdateInput")
	}

	for _, imgId := range []*string{&input.PostgresImageId, &input.RedisImageId, &input.NginxImageId, &input.DifyApiImageId, &input.DifyPluginImageId, &input.DifyWebImageId, &input.DifySandboxImageId, &input.DifySSRFImageId, &input.DifyWeaviateImageId} {
		if *imgId != "" {
			_, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), imgId)
			if err != nil {
				return input, errors.Wrapf(err, "validate image_id %s", *imgId)
			}
		}
	}

	return input, nil
}

func (model *SDifyModel) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	count, err := GetDifyManager().Query().Equals("dify_model_id", model.Id).CountWithError()
	if nil != err {
		return errors.Wrap(err, "fetch dify")
	}
	if count > 0 {
		return errors.Wrap(errors.ErrNotSupported, "This model is currently in use")
	}
	return nil
}
