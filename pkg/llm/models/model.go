package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func NewSLLMModelBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SLLMModelBaseManager {
	return SLLMModelBaseManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
		),
	}
}

type SLLMModelBaseManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SLLMModelBase struct {
	db.SSharableVirtualResourceBase

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

func (man *SLLMModelBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.SharableVirtualResourceListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	return q, nil
}

func (man *SLLMModelBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.LLMModelBaseCreateInput) (api.LLMModelBaseCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
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

	input.Status = api.STATUS_READY
	return input, nil
}
