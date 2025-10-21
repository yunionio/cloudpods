package models

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

func init() {

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
