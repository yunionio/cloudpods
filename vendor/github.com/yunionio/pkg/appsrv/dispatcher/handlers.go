package dispatcher

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient/modules"
	"github.com/yunionio/pkg/appsrv"
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
	Get(ctx context.Context, idstr string, query jsonutils.JSONObject) (jsonutils.JSONObject, error)
	GetSpecific(ctx context.Context, idstr string, spec string, query jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Create(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxId string) (jsonutils.JSONObject, error)
	BatchCreate(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, count int, ctxId string) ([]modules.SubmitResult, error)
	PerformClassAction(ctx context.Context, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PerformAction(ctx context.Context, idstr string, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// UpdateClass(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Update(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// DeleteClass(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Delete(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
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
