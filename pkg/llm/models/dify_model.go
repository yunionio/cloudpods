package models

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
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
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
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
	db.SSharableVirtualResourceBaseManager
}

type SDifyModel struct {
	db.SSharableVirtualResourceBase

	PostgresImageId     string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	RedisImageId        string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	NginxImageId        string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifyApiImageId      string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifyPluginImageId   string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifyWebImageId      string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifySandboxImageId  string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifySSRFImageId     string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DifyWeaviateImageId string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	BandwidthMb         int               `nullable:"false" default:"0" create:"optional" list:"user" update:"user"`
	Cpu                 int               `nullable:"false" default:"1" create:"optional" list:"user" update:"user"`
	Memory              int               `nullable:"false" default:"512" create:"optional" list:"user" update:"user"`
	Volumes             *api.Volumes      `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	PortMappings        *api.PortMappings `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	Devices             *api.Devices      `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	Envs                *api.Envs         `charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
	// Properties
	Properties map[string]string `charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
}

func (man *SDifyModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.DifyModelListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableBaseResourceManager.ListItemFilter")
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

	for _, imgId := range []*string{&input.PostgresImageId, &input.RedisImageId, &input.NginxImageId, &input.DifyApiImageId, &input.DifyPluginImageId, &input.DifyWebImageId, &input.DifySandboxImageId, &input.DifySSRFImageId, &input.DifyWeaviateImageId} {
		_, err := validators.ValidateModel(ctx, userCred, GetLLMImageManager(), imgId)
		if err != nil {
			return input, errors.Wrapf(err, "validate image_id %s", *imgId)
		}
	}

	input.Status = api.STATUS_READY
	return input, nil
}
