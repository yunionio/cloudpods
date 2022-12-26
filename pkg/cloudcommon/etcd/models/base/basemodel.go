// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package base

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/object"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SEtcdBaseModel struct {
	object.SObject

	manager IEtcdModelManager

	ID string
}

func (model *SEtcdBaseModel) Keyword() string {
	return model.manager.Keyword()
}

func (model *SEtcdBaseModel) GetModelManager() IEtcdModelManager {
	return model.manager
}

func (model *SEtcdBaseModel) SetModelManager(manager IEtcdModelManager, virtual IEtcdModel) {
	model.manager = manager
	model.SetVirtualObject(virtual)
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
