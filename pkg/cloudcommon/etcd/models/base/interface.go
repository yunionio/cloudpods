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
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IEtcdModelManager interface {
	lockman.ILockedClass

	KeywordPlural() string

	Allocate() IEtcdModel

	AllJson(ctx context.Context) ([]jsonutils.JSONObject, error)
	GetJson(ctx context.Context, idstr string) (jsonutils.JSONObject, error)
	Get(ctx context.Context, idstr string, model IEtcdModel) error
	All(ctx context.Context, dest interface{}) error
	Save(ctx context.Context, model IEtcdModel) error
	Delete(ctx context.Context, model IEtcdModel) error
	Session(ctx context.Context, model IEtcdModel) error
	Watch(ctx context.Context, onCreate etcd.TEtcdCreateEventFunc, onModify etcd.TEtcdModifyEventFunc, onDelete etcd.TEtcdDeleteEventFunc)

	CustomizeHandlerInfo(handler *appsrv.SHandlerInfo)
	FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error)
	FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error)
}

type IEtcdModel interface {
	lockman.ILockedObject

	GetModelManager() IEtcdModelManager
	SetModelManager(IEtcdModelManager, IEtcdModel)

	SetId(id string)

	GetExtraDetailsHeaders(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) map[string]string
}
