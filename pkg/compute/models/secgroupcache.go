package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"
)

type SSecurityGroupCacheManager struct {
	db.SResourceBaseManager
}

type SSecurityGroupCache struct {
	db.SResourceBase
	SManagedResourceBase

	Id            string `width:"128" charset:"ascii" primary:"true" list:"user"`
	SecgroupId    string `width:"128" charset:"ascii" create:"required"`
	ExternalId    string `width:"256" charset:"utf8" index:"true" list:"admin" create:"admin_optional"`
	CloudregionId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
}

var SecurityGroupCacheManager *SSecurityGroupCacheManager

func init() {
	SecurityGroupCacheManager = &SSecurityGroupCacheManager{SResourceBaseManager: db.NewResourceBaseManager(SSecurityGroupCache{}, "secgroupcache_tbl", "secgroupcache", "secgroupcaches")}
}

func (self *SSecurityGroupCache) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = stringutils.UUID4()
	}
}

func (manager *SSecurityGroupCacheManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SSecurityGroupCacheManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSecurityGroupCache) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SSecurityGroupCache) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SSecurityGroupCacheManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *SSecurityGroupCacheManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (sql *sqlchemy.SQuery, err error) {
	sql, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if defsecgroup, _ := query.GetString("secgroup"); len(defsecgroup) > 0 {
		if secgroup, _ := SecurityGroupManager.FetchByIdOrName(userCred.GetProjectId(), defsecgroup); secgroup != nil {
			sql = sql.Equals("secgroup_id", secgroup.GetId())
		} else {
			return nil, httperrors.NewNotFoundError(fmt.Sprintf("Security Group %s not found", defsecgroup))
		}
	}
	return sql, nil
}

func (self *SSecurityGroupCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (manager *SSecurityGroupCacheManager) GetSecgroupCache(ctx context.Context, userCred mcclient.TokenCredential, secgroupId, regionId, providerId string) *SSecurityGroupCache {
	secgroupCache := SSecurityGroupCache{}
	query := manager.Query()
	cond := sqlchemy.AND(sqlchemy.Equals(query.Field("secgroup_id"), secgroupId), sqlchemy.Equals(query.Field("cloudregion_id"), regionId), sqlchemy.Equals(query.Field("manager_id"), providerId))
	query = query.Filter(cond)

	count := query.Count()
	if count > 1 {
		log.Errorf("duplicate secgroupcache for secgroup: %s regionId: %s providerId: %s", secgroupId, regionId, providerId)
	} else if count == 0 {
		return nil
	}
	query.First(&secgroupCache)
	secgroupCache.SetModelManager(manager)
	return &secgroupCache
}

func (manager *SSecurityGroupCacheManager) Register(ctx context.Context, userCred mcclient.TokenCredential, secgroupId, regionId, providerId string) *SSecurityGroupCache {
	lockman.LockClass(ctx, manager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, manager, userCred.GetProjectId())

	secgroupCache := manager.GetSecgroupCache(ctx, userCred, secgroupId, regionId, providerId)
	if secgroupCache != nil {
		return secgroupCache
	}

	secgroupCache = &SSecurityGroupCache{
		SecgroupId:    secgroupId,
		CloudregionId: regionId,
	}
	secgroupCache.SetModelManager(manager)
	secgroupCache.ManagerId = providerId
	if err := manager.TableSpec().Insert(secgroupCache); err != nil {
		log.Errorf("insert secgroupcache error: %v", err)
		return nil
	}
	return secgroupCache
}

func (self *SSecurityGroupCache) SetExternalId(externalId string) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.ExternalId = externalId
		return nil
	})
	return err
}
