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

package dispatcher

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type IMiddlewareFilter interface {
	Filter(appsrv.FilterHandler) appsrv.FilterHandler
}

type IModelDispatchHandler interface {
	IMiddlewareFilter

	Keyword() string
	KeywordPlural() string
	ContextKeywordPlural() []string

	List(ctx context.Context, query jsonutils.JSONObject, ctxId string) (*modules.ListResult, error)
	Get(ctx context.Context, idstr string, query jsonutils.JSONObject, isHead bool) (jsonutils.JSONObject, error)
	GetSpecific(ctx context.Context, idstr string, spec string, query jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Create(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxId string) (jsonutils.JSONObject, error)
	BatchCreate(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, count int, ctxId string) ([]modules.SubmitResult, error)
	PerformClassAction(ctx context.Context, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PerformAction(ctx context.Context, idstr string, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// UpdateClass(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Update(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// DeleteClass(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Delete(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	CustomizeHandlerInfo(info *appsrv.SHandlerInfo)
	FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error)
	FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error)
}

type IJointModelDispatchHandler interface {
	IMiddlewareFilter

	Keyword() string
	KeywordPlural() string
	MasterKeywordPlural() string
	SlaveKeywordPlural() string

	List(ctx context.Context, query jsonutils.JSONObject, ctxId string) (*modules.ListResult, error)
	ListMasterDescendent(ctx context.Context, idStr string, query jsonutils.JSONObject) (*modules.ListResult, error)
	ListSlaveDescendent(ctx context.Context, idStr string, query jsonutils.JSONObject) (*modules.ListResult, error)
	Get(ctx context.Context, id1 string, id2 string, query jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Attach(ctx context.Context, id1 string, id2 string, query jsonutils.JSONObject, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Update(ctx context.Context, id1 string, id2 string, query jsonutils.JSONObject, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Detach(ctx context.Context, id1 string, id2 string, query jsonutils.JSONObject, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
}
