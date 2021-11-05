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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// TODO: 1. Access to a non-existent role will still trigger synchronization.
//       2. The failure of the watch mechanism makes the cache miss an event,
//          then there is outdated data in the cache before the expiration
//          time comes or the event about this data occurs again.

var (
	DefaultRoleFetcher func(ctx context.Context, id string) (*SRole, error)
)

type SRoleCacheManager struct {
	SKeystoneCacheObjectManager
	watching bool
}

type SRole struct {
	SKeystoneCacheObject
}

func (role *SRole) GetModelManager() IModelManager {
	return RoleCacheManager
}

var RoleCacheManager *SRoleCacheManager

func init() {
	RoleCacheManager = &SRoleCacheManager{
		NewKeystoneCacheObjectManager(SRole{}, "roles_cache_tbl", "role", "roles"), false}
	// log.Debugf("initialize role cache manager %s", RoleCacheManager.KeywordPlural())
	RoleCacheManager.SetVirtualObject(RoleCacheManager)

	DefaultRoleFetcher = RoleCacheManager.FetchRoleByIdOrName
}

func (manager *SRoleCacheManager) FetchRoleByIdOrName(ctx context.Context, idStr string) (*SRole, error) {
	return manager.fetchRole(ctx, idStr, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		if stringutils2.IsUtf8(idStr) {
			return q.Equals("name", idStr)
		} else {
			return q.Filter(sqlchemy.OR(
				sqlchemy.Equals(q.Field("id"), idStr),
				sqlchemy.Equals(q.Field("name"), idStr),
			))
		}
	})
}

func (manager *SRoleCacheManager) FetchRoleById(ctx context.Context, idStr string) (*SRole, error) {
	return manager.fetchRole(ctx, idStr, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Filter(sqlchemy.Equals(q.Field("id"), idStr))
	})
}

func (manager *SRoleCacheManager) FetchRoleByName(ctx context.Context, idStr string) (*SRole, error) {
	return manager.fetchRole(ctx, idStr, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Filter(sqlchemy.Equals(q.Field("name"), idStr))
	})
}

func (manager *SRoleCacheManager) fetchRole(ctx context.Context, idStr string, noExpireCheck bool, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) (*SRole, error) {
	q := manager.Query()
	q = filter(q)
	tobj, err := NewModelObject(manager)
	if err != nil {
		return nil, errors.Wrap(err, "NewModelObject")
	}
	err = q.First(tobj)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query")
	} else if tobj != nil {
		role := tobj.(*SRole)
		if noExpireCheck || !role.IsExpired() {
			return role, nil
		}
	}
	return manager.FetchRoleFromKeystone(ctx, idStr)
}

func (manager *SRoleCacheManager) FetchRoleFromKeystone(ctx context.Context, idStr string) (*SRole, error) {
	if len(idStr) == 0 {
		log.Debugf("fetch empty role!!!!\n%s", debug.Stack())
		return nil, fmt.Errorf("Empty idStr")
	}

	// It's to query the full list of roles(contains other domain's ones and system ones)
	query := jsonutils.NewDict()
	query.Set("scope", jsonutils.NewString("system"))
	query.Set("system", jsonutils.JSONTrue)

	s := auth.GetAdminSession(ctx, consts.GetRegion(), "")
	role, err := modules.RolesV3.GetById(s, idStr, query)
	if err != nil {
		if je, ok := err.(*httputils.JSONClientError); ok && je.Code == 404 {
			role, err = modules.RolesV3.GetByName(s, idStr, query)
			if je, ok := err.(*httputils.JSONClientError); ok && je.Code == 404 {
				return nil, sql.ErrNoRows
			}
		}
		if err != nil {
			log.Errorf("fetch role %s fail %s", idStr, err)
			return nil, errors.Wrap(err, "modules.RolesV3.Get")
		}
	}
	id, _ := role.GetString("id")
	name, _ := role.GetString("name")
	domainId, _ := role.GetString("domain_id")
	domainName, _ := role.GetString("project_domain")
	return manager.Save(ctx, id, name, domainId, domainName)
}

func (manager *SRoleCacheManager) Save(ctx context.Context, idStr string, name string, domainId string, domain string) (*SRole, error) {
	lockman.LockRawObject(ctx, manager.KeywordPlural(), idStr)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), idStr)

	objo, err := manager.FetchById(idStr)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("FetchRolebyId fail %s", err)
		return nil, err
	}
	if err == nil {
		obj := objo.(*SRole)
		_, err = Update(obj, func() error {
			obj.Id = idStr
			obj.Name = name
			obj.Domain = domain
			obj.DomainId = domainId
			obj.LastCheck = time.Now().UTC()
			return nil
		})
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else {
		objm, err := NewModelObject(manager)
		obj := objm.(*SRole)
		obj.Id = idStr
		obj.Name = name
		obj.Domain = domain
		obj.DomainId = domainId
		obj.LastCheck = time.Now().UTC()
		err = manager.TableSpec().InsertOrUpdate(ctx, obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	}
}

func (manager *SRoleCacheManager) OnAdd(obj *jsonutils.JSONDict) {
	id, err := obj.GetString("id")
	if err != nil {
		log.Errorf("unable to get Id: %v", err)
		return
	}
	ctx := context.Background()
	lockman.LockRawObject(ctx, manager.KeywordPlural(), id)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), id)
	role := new(SRole)
	role.Id = id
	role.Name, _ = obj.GetString("name")
	role.DomainId, _ = obj.GetString("domain_id")
	role.LastCheck = time.Now()
	err = manager.TableSpec().Insert(ctx, role)
	if err != nil {
		log.Errorf("unable to insert: %v", err)
	}
}

func (manager *SRoleCacheManager) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	id, err := oldObj.GetString("id")
	if err != nil {
		log.Errorf("unable to get Id: %v", err)
		return
	}
	ctx := context.Background()
	role, err := manager.fetchRole(ctx, id, true, nil)
	if err != nil {
		log.Errorf("unable to fetch Role from db: %v", err)
		return
	}
	name, _ := newObj.GetString("name")
	domainId, _ := newObj.GetString("domain_id")
	if role.Name == name && role.DomainId == domainId {
		return
	}
	lockman.LockRawObject(ctx, manager.KeywordPlural(), id)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), id)
	_, err = Update(role, func() error {
		role.Name = name
		role.DomainId = domainId
		role.LastCheck = time.Now()
		return nil
	})
	if err != nil {
		log.Errorf("unable to update: %v", err)
	}
}

func (manager *SRoleCacheManager) OnDelete(obj *jsonutils.JSONDict) {
	id, err := obj.GetString("id")
	if err != nil {
		log.Errorf("unable to get Id: %v", err)
		return
	}
	ctx := context.Background()
	role, err := manager.fetchRole(ctx, id, true, nil)
	if err != nil {
		log.Errorf("unable to fetch Role from db: %v", err)
		return
	}
	lockman.LockRawObject(ctx, manager.KeywordPlural(), id)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), id)
	_, err = Update(role, func() error {
		role.Deleted = false
		return nil
	})
	if err != nil {
		log.Errorf("unable to delete role: %v", err)
	}
}

func (manager *SRoleCacheManager) StartWatchRoleInKeystone() error {
	if manager.watching {
		return nil
	}
	ctx := context.Background()
	s := auth.GetAdminSession(ctx, "", "")
	watchMan, err := informer.NewWatchManagerBySession(s)
	if err != nil {
		return err
	}
	resMan := &modules.RolesV3
	return watchMan.For(resMan).AddEventHandler(ctx, manager)
}

func (r *SRole) IsExpired() bool {
	if r.LastCheck.IsZero() {
		return true
	}
	now := time.Now().UTC()
	if r.LastCheck.Add(consts.GetRoleCacheExpireHours()).Before(now) {
		return true
	}
	return false
}
