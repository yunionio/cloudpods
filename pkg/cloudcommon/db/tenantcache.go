package db

import (
	"context"
	"database/sql"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

type STenantCacheManager struct {
	SKeystoneCacheObjectManager
}

type STenant struct {
	SKeystoneCacheObject
}

func NewTenant(idStr string, name string) STenant {
	return STenant{SKeystoneCacheObject: NewKeystoneCacheObject(idStr, name, "", "")}
}

func (tenant *STenant) GetModelManager() IModelManager {
	return TenantCacheManager
}

var TenantCacheManager *STenantCacheManager

func init() {
	TenantCacheManager = &STenantCacheManager{NewKeystoneCacheObjectManager(STenant{}, "tenant_cache_tbl", "tenant", "tenants")}
	// log.Debugf("Initialize tenant cache manager %s %s", TenantCacheManager.KeywordPlural(), TenantCacheManager)
}

func (manager *STenantCacheManager) FetchTenantByIdOrName(ctx context.Context, idStr string) (*STenant, error) {
	tenant, err := manager.FetchByIdOrName(nil, idStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return manager.fetchTenantFromKeystone(ctx, idStr)
		} else {
			log.Errorf("FetchTenantByIdOrName fail: %s", err)
			return nil, err
		}
	} else {
		return tenant.(*STenant), nil
	}
}

func (manager *STenantCacheManager) FetchTenantById(ctx context.Context, idStr string) (*STenant, error) {
	tenant, err := manager.FetchById(idStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return manager.fetchTenantFromKeystone(ctx, idStr)
		} else {
			log.Errorf("FetchTenantById fail: %s", err)
			return nil, err
		}
	} else {
		return tenant.(*STenant), nil
	}
}

func (manager *STenantCacheManager) FetchTenantByName(ctx context.Context, idStr string) (*STenant, error) {
	tenant, err := manager.FetchByName(nil, idStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return manager.fetchTenantFromKeystone(ctx, idStr)
		} else {
			log.Errorf("FetchTenantById fail: %s", err)
			return nil, err
		}
	} else {
		return tenant.(*STenant), nil
	}
}

func (manager *STenantCacheManager) fetchTenantFromKeystone(ctx context.Context, idStr string) (*STenant, error) {
	s := auth.GetAdminSession(consts.GetRegion(), "v1")
	tenant, err := modules.Projects.Get(s, idStr, nil)
	if err != nil {
		log.Errorf("fetch project fail %s", err)
		return nil, err
	}
	tenantId, err := tenant.GetString("id")
	tenantName, err := tenant.GetString("name")
	return manager.Save(ctx, tenantId, tenantName, "", "")
}

func (manager *STenantCacheManager) Save(ctx context.Context, idStr string, name string, domainId string, domain string) (*STenant, error) {
	lockman.LockRawObject(ctx, manager.keyword, idStr)
	defer lockman.ReleaseRawObject(ctx, manager.keyword, idStr)

	objo, err := manager.FetchById(idStr)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("FetchTenantbyId fail %s", err)
		return nil, err
	}
	if err == nil {
		obj := objo.(*STenant)
		_, err = manager.TableSpec().Update(obj, func() error {
			obj.Id = idStr
			obj.Name = name
			obj.Domain = domain
			obj.DomainId = domainId
			return nil
		})
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else {
		objm, err := NewModelObject(manager)
		obj := objm.(*STenant)
		obj.Id = idStr
		obj.Name = name
		obj.Domain = domain
		obj.DomainId = domainId
		err = manager.TableSpec().Insert(obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	}
}
