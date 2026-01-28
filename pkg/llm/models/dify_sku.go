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
	GetDifySkuManager()
}

var difySkuManager *SDifySkuManager

func GetDifySkuManager() *SDifySkuManager {
	if difySkuManager != nil {
		return difySkuManager
	}
	difySkuManager = &SDifySkuManager{
		SLLMSkuBaseManager: NewSLLMSkuBaseManager(
			SDifySku{},
			"dify_skus_tbl",
			"dify_sku",
			"dify_skus",
		),
	}
	difySkuManager.SetVirtualObject(difySkuManager)
	return difySkuManager
}

type SDifySkuManager struct {
	SLLMSkuBaseManager
}

type SDifySku struct {
	SLLMSkuBase

	PostgresImageId     string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	RedisImageId        string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	NginxImageId        string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	DifyApiImageId      string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	DifyPluginImageId   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	DifyWebImageId      string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	DifySandboxImageId  string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	DifySSRFImageId     string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	DifyWeaviateImageId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
}

func (man *SDifySkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.DifySkulListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SLLMSkuBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SLLMSkuBaseManager.ListItemFilter")
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

func (man *SDifySkuManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.DifySkuCreateInput) (*api.DifySkuCreateInput, error) {
	var err error
	input.LLMSKuBaseCreateInput, err = man.SLLMSkuBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMSKuBaseCreateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLLMSkuBaseManager.ValidateCreateData")
	}

	for _, imgId := range []*string{&input.PostgresImageId, &input.RedisImageId, &input.NginxImageId, &input.DifyApiImageId, &input.DifyPluginImageId, &input.DifyWebImageId, &input.DifySandboxImageId, &input.DifySSRFImageId, &input.DifyWeaviateImageId} {
		_, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), imgId)
		if err != nil {
			return input, errors.Wrapf(err, "validate image_id %s", *imgId)
		}
	}

	return input, nil
}

func (sku *SDifySku) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DifySkuUpdateInput) (api.DifySkuUpdateInput, error) {
	var err error
	input.LLMSkuBaseUpdateInput, err = sku.SLLMSkuBase.ValidateUpdateData(ctx, userCred, query, input.LLMSkuBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate LLMSkuBaseUpdateInput")
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

func (sku *SDifySku) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	count, err := GetDifyManager().Query().Equals("dify_sku_id", sku.Id).CountWithError()
	if nil != err {
		return errors.Wrap(err, "fetch dify")
	}
	if count > 0 {
		return errors.Wrap(errors.ErrNotSupported, "This sku is currently in use")
	}
	return nil
}
