package quotas

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SQuotaBaseManager struct {
	db.SModelBaseManager

	pendingStore IQuotaStore
	usageStore   IQuotaStore

	autoCreate bool
}

const (
	nameSeparator = "."

	quotaKeyword  = "quota"
	quotaKeywords = "quotas"
)

func NewQuotaBaseManager(model interface{}, tableName string, pendingStore IQuotaStore, usageStore IQuotaStore) SQuotaBaseManager {
	autoCreate := false
	if usageStore != nil {
		autoCreate = true
	}
	return SQuotaBaseManager{
		SModelBaseManager: db.NewModelBaseManager(model, tableName, quotaKeyword, quotaKeywords),
		pendingStore:      pendingStore,
		usageStore:        usageStore,
		autoCreate:        autoCreate,
	}
}

type SQuotaBase struct {
	db.SModelBase

	DomainId  string `width:"128" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	Platform  string `width:"128" charset:"utf8" nullable:"false" primary:"true" list:"user"`

	UpdatedAt     time.Time `nullable:"false" updated_at:"true" list:"user"`
	UpdateVersion int       `default:"0" nullable:"false" auto_version:"true" list:"user"`
}

func (manager *SQuotaBaseManager) getQuotaInternal(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	q := manager.Query()
	q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	if scope == rbacutils.ScopeProject {
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	} else {
		q = q.IsNullOrEmpty("tenant_id")
	}
	var key string
	if len(platform) > 0 {
		key = strings.Join(platform, nameSeparator)
	}
	q = q.Equals("platform", key)
	err := q.First(quota)
	if err != nil && err != sql.ErrNoRows {
		return err
	} else if err == sql.ErrNoRows && manager.autoCreate {
		quota.FetchSystemQuota()
		return manager.setQuotaInternal(ctx, nil, scope, ownerId, platform, quota)
	}
	return nil
}

func (manager *SQuotaBaseManager) setQuotaInternal(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	base := SQuotaBase{}
	base.DomainId = ownerId.GetProjectDomainId()
	if scope == rbacutils.ScopeProject {
		base.ProjectId = ownerId.GetProjectId()
	}
	if len(platform) > 0 {
		base.Platform = strings.Join(platform, nameSeparator)
	}
	base.SetModelManager(manager, quota.(db.IModel))

	if !reflectutils.FillEmbededStructValue(reflect.Indirect(reflect.ValueOf(quota)), reflect.ValueOf(base)) {
		return errors.Error("no embed SBaseQuota")
	}

	return manager.TableSpec().InsertOrUpdate(quota)
}
