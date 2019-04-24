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

package db

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
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
	if len(idStr) == 0 {
		log.Debugf("fetch empty tenant!!!!")
		debug.PrintStack()
		return nil, fmt.Errorf("Empty idStr")
	}
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	tenant, err := modules.Projects.Get(s, idStr, nil)
	if err != nil {
		if je, ok := err.(*httputils.JSONClientError); ok && je.Code == 404 {
			return nil, sql.ErrNoRows
		}
		log.Errorf("fetch project %s fail %s", idStr, err)
		return nil, err
	}
	tenantId, err := tenant.GetString("id")
	tenantName, err := tenant.GetString("name")
	return manager.Save(ctx, tenantId, tenantName, "", "")
}

func (manager *STenantCacheManager) Save(ctx context.Context, idStr string, name string, domainId string, domain string) (*STenant, error) {
	lockman.LockRawObject(ctx, manager.KeywordPlural(), idStr)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), idStr)

	objo, err := manager.FetchById(idStr)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("FetchTenantbyId fail %s", err)
		return nil, err
	}
	if err == nil {
		obj := objo.(*STenant)
		_, err = Update(obj, func() error {
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

func (manager *STenantCacheManager) GenerateProjectUserCred(ctx context.Context, projectName string) (mcclient.TokenCredential, error) {
	project, err := manager.FetchTenantByIdOrName(ctx, projectName)
	if err != nil {
		return nil, err
	}
	return &mcclient.SSimpleToken{
		Project:   project.Name,
		ProjectId: project.Id,
	}, nil
}
