package base

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SEtcdBaseModel struct {
	manager IEtcdModelManager

	ID string
}

func (model *SEtcdBaseModel) Keyword() string {
	return model.manager.Keyword()
}

func (model *SEtcdBaseModel) GetModelManager() IEtcdModelManager {
	return model.manager
}

func (model *SEtcdBaseModel) SetModelManager(manager IEtcdModelManager) {
	model.manager = manager
}

func (model *SEtcdBaseModel) GetId() string {
	return model.ID
}

func (model *SEtcdBaseModel) SetId(id string) {
	model.ID = id
}

func (model *SEtcdBaseModel) GetExtraDetailsHeaders(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) map[string]string {
	return nil
}
