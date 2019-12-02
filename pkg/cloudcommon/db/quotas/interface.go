package quotas

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IQuotaKeys interface {
	Fields() []string
	Values() []string
	Compare(IQuotaKeys) int
}

type IQuota interface {
	GetKeys() IQuotaKeys
	SetKeys(IQuotaKeys)

	FetchSystemQuota()
	FetchUsage(ctx context.Context) error
	Update(quota IQuota)
	Add(quota IQuota)
	Sub(quota IQuota)
	Exceed(request IQuota, quota IQuota) error
	IsEmpty() bool
	ToJSON(prefix string) jsonutils.JSONObject
}

type IQuotaStore interface {
	GetQuota(ctx context.Context, keys IQuotaKeys, quota IQuota) error
	GetChildrenQuotas(ctx context.Context, keys IQuotaKeys) ([]IQuota, error)
	GetParentQuotas(ctx context.Context, keys IQuotaKeys) ([]IQuota, error)

	SetQuota(ctx context.Context, userCred mcclient.TokenCredential, quota IQuota) error
	AddQuota(ctx context.Context, userCred mcclient.TokenCredential, diff IQuota, target IQuota) error
	SubQuota(ctx context.Context, userCred mcclient.TokenCredential, diff IQuota, target IQuota) error

	DeleteQuota(ctx context.Context, userCred mcclient.TokenCredential, keys IQuotaKeys) error
	DeleteAllQuotas(ctx context.Context, userCred mcclient.TokenCredential, keys IQuotaKeys) error
}

type IQuotaManager interface {
	db.IResourceModelManager

	FetchIdNames(ctx context.Context, idMap map[string]map[string]string) (map[string]map[string]string, error)
}
