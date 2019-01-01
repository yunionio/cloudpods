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
	Watch(ctx context.Context, onCreate etcd.TEtcdCreateEventFunc, onModify etcd.TEtcdModifyEventFunc)

	CustomizeHandlerInfo(handler *appsrv.SHandlerInfo)
	FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error)
	FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error)
}

type IEtcdModel interface {
	lockman.ILockedObject

	GetModelManager() IEtcdModelManager
	SetModelManager(IEtcdModelManager)

	SetId(id string)

	GetExtraDetailsHeaders(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) map[string]string
}
