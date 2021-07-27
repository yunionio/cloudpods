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

package models

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudgroupcacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
}

var CloudgroupcacheManager *SCloudgroupcacheManager

func init() {
	CloudgroupcacheManager = &SCloudgroupcacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SCloudgroupcache{},
			"cloudgroupcaches_tbl",
			"cloudgroupcache",
			"cloudgroupcaches",
		),
	}
	CloudgroupcacheManager.SetVirtualObject(CloudgroupcacheManager)
}

type SCloudgroupcache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudaccountResourceBase

	// 用户组Id
	CloudgroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" index:"true" json:"cloudgroup_id"`
}

func (manager *SCloudgroupcacheManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, manager)
}

// 公有云权限组缓存
func (manager *SCloudgroupcacheManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudgroupcacheListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SCloudaccountResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudaccountResourceListInput)
	if err != nil {
		return nil, err
	}

	if len(query.CloudgroupId) > 0 {
		_, err = CloudgroupManager.FetchById(query.CloudgroupId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudgroup", query.CloudgroupId)
			}
			return q, httperrors.NewGeneralError(errors.Wrap(err, "CloudgroupManager.FetchById"))
		}
		q = q.Equals("cloudgroup_id", query.CloudgroupId)
	}

	return q, nil
}

// +onecloud:swagger-gen-ignore
func (self *SCloudgroupcache) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupUpdateInput) (api.CloudgroupUpdateInput, error) {
	return input, httperrors.NewNotSupportedError("Not support")
}

// +onecloud:swagger-gen-ignore
func (manager *SCloudgroupcacheManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.CloudgroupcacheCreateInput) (api.CloudgroupcacheCreateInput, error) {
	return input, httperrors.NewNotSupportedError("Not support")
}

// 删除权限组缓存
func (self *SCloudgroupcache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartCloudgroupcacheDeleteTask(ctx, userCred, "")
}

func (manager *SCloudgroupcacheManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (self *SCloudgroupcache) GetOwnerId() mcclient.IIdentityProvider {
	group, err := self.GetCloudgroup()
	if err != nil {
		return nil
	}
	return group.GetOwnerId()
}

func (self *SCloudgroupcache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SCloudgroupcache) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (manager *SCloudgroupcacheManager) newFromCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup, group *SCloudgroup, cloudaccountId string) (*SCloudgroupcache, error) {
	cache := &SCloudgroupcache{}
	cache.SetModelManager(manager, cache)
	cache.CloudgroupId = group.Id
	cache.Name = iGroup.GetName()
	cache.Description = iGroup.GetDescription()
	cache.Status = api.CLOUD_GROUP_CACHE_STATUS_AVAILABLE
	cache.ExternalId = iGroup.GetGlobalId()
	cache.CloudaccountId = cloudaccountId
	return cache, manager.TableSpec().Insert(ctx, cache)
}

func (self *SCloudgroupcache) syncWithCloudgroupcache(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup) error {
	_, err := db.Update(self, func() error {
		self.Name = iGroup.GetName()
		self.Description = iGroup.GetDescription()
		self.Status = api.CLOUD_GROUP_CACHE_STATUS_AVAILABLE
		return nil
	})
	return err
}

func (manager *SCloudgroupcacheManager) Register(group *SCloudgroup, account *SCloudaccount) (*SCloudgroupcache, error) {
	q := manager.Query().Equals("cloudgroup_id", group.Id).Equals("cloudaccount_id", account.Id)
	caches := []SCloudgroupcache{}
	err := db.FetchModelObjects(manager, q, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	if len(caches) == 0 {
		cache := &SCloudgroupcache{}
		cache.SetModelManager(manager, cache)
		cache.Name = group.Name
		cache.Description = group.Description
		cache.Status = api.CLOUD_GROUP_CACHE_STATUS_CREATING
		cache.CloudgroupId = group.Id
		cache.CloudaccountId = account.Id
		return cache, manager.TableSpec().Insert(context.Background(), cache)
	}
	for i := range caches {
		if len(caches[i].ExternalId) > 0 {
			return &caches[i], nil
		}
	}
	return &caches[0], nil
}

// 获取权限组缓存详情
func (manager *SCloudgroupcacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudgroupcacheDetails {
	rows := make([]api.CloudgroupcacheDetails, len(objs))
	statusRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.CloudgroupcacheDetails{
			StatusStandaloneResourceDetails: statusRows[i],
			CloudaccountResourceDetails:     acRows[i],
		}
	}
	return rows
}

func (self *SCloudgroupcache) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "syncstatus")
}

// 同步权限组缓存状态
func (self *SCloudgroupcache) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartCloudgroupcacheSyncstatusTask(ctx, userCred, "")
}

func (self *SCloudgroupcache) StartCloudgroupcacheSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupcacheSyncstatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_GROUP_CACHE_STATUS_SYNC_STATUS, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudgroupcache) StartCloudgroupcacheDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupcacheDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_GROUP_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudgroupcache) GetOrCreateICloudgroup(ctx context.Context, userCred mcclient.TokenCredential) (cloudprovider.ICloudgroup, error) {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)
	if len(self.ExternalId) > 0 {
		iGroup, err := self.GetICloudgroup()
		if err == nil {
			return iGroup, nil
		}
		if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, errors.Wrap(err, "GetICloudgroup")
		}
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudaccount")
	}

	provider, err := account.GetProvider()
	if err != nil {
		return nil, errors.Wrap(err, "account.GetProvider")
	}

	randomString := func(prefix string, length int) string {
		return fmt.Sprintf("%s-%s", prefix, rand.String(length))
	}

	groupName := self.Name
	for i := 2; i < 30; i++ {
		_, err := provider.GetICloudgroupByName(groupName)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				break
			}
			return nil, errors.Wrapf(err, "GetICloudgroupByName(%s)", groupName)
		}
		groupName = randomString(self.Name, i)
	}

	iGroup, err := provider.CreateICloudgroup(groupName, self.Description)
	if err != nil {
		logclient.AddSimpleActionLog(self, logclient.ACT_CREATE, err, userCred, false)
		return nil, errors.Wrap(err, "CreateICloudgroup")
	}
	_, err = db.Update(self, func() error {
		self.Name = groupName
		self.ExternalId = iGroup.GetGlobalId()
		self.Status = api.CLOUD_GROUP_CACHE_STATUS_AVAILABLE
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "db.Update")
	}
	err = self.SyncSystemCloudpoliciesForCloud(ctx, userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "SyncSystemCloudpoliciesForCloud")
	}
	err = self.SyncCustomCloudpoliciesForCloud(ctx, userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "SyncCustomCloudpoliciesForCloud")
	}
	err = self.SyncCloudusersForCloud(ctx, userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "SyncCloudusersForCloud")
	}
	return iGroup, nil
}

func (self *SCloudgroupcache) GetCloudgroup() (*SCloudgroup, error) {
	group, err := CloudgroupManager.FetchById(self.CloudgroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.CloudgroupId)
	}
	return group.(*SCloudgroup), nil
}

func (self *SCloudgroupcache) GetICloudgroup() (cloudprovider.ICloudgroup, error) {
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetCloudaccount")
	}
	provider, err := account.GetProvider()
	if err != nil {
		return nil, errors.Wrap(err, "account.GetProvider")
	}
	return provider.GetICloudgroupByName(self.Name)
}

func (self *SCloudgroupcache) SyncCustomCloudpoliciesForCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	group, err := self.GetCloudgroup()
	if err != nil {
		return errors.Wrap(err, "GetCloudgroup")
	}
	policies, err := group.GetCustomCloudpolicies()
	if err != nil {
		return errors.Wrap(err, "GetCustomCloudpolicies")
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrap(err, "GetCloudaccount")
	}
	policyIds := []string{}
	for i := range policies {
		err = account.getOrCacheCustomCloudpolicy(ctx, "", policies[i].Id)
		if err != nil {
			return errors.Wrapf(err, "getOrCacheCloudpolicy %s(%s) for account %s", policies[i].Name, policies[i].Provider, self.Name)
		}
		policyIds = append(policyIds, policies[i].Id)
	}
	dbCaches := []SCloudpolicycache{}
	if len(policyIds) > 0 {
		dbCaches, err = account.GetCloudpolicycaches(policyIds, "")
		if err != nil {
			return errors.Wrapf(err, "GetCloudpolicycaches")
		}
	}
	iGroup, err := self.GetICloudgroup()
	if err != nil {
		return errors.Wrapf(err, "GetICloudgroup")
	}
	iPolicies, err := iGroup.GetICustomCloudpolicies()
	if err != nil {
		return errors.Wrap(err, "iGroup.GetICustomCloudpolicies")
	}

	added := make([]SCloudpolicycache, 0)
	commondb := make([]SCloudpolicycache, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	removed := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbCaches, iPolicies, &added, &commondb, &commonext, &removed)
	if err != nil {
		return errors.Wrap(err, "compare.CompareSets")
	}

	result := compare.SyncResult{}
	for i := 0; i < len(removed); i++ {
		err = iGroup.DetachCustomPolicy(removed[i].GetGlobalId())
		if err != nil {
			result.DeleteError(err)
			logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, err, userCred, false)
			continue
		}
		logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, jsonutils.Marshal(removed[i]), userCred, true)
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = iGroup.AttachCustomPolicy(added[i].ExternalId)
		if err != nil {
			result.AddError(err)
			logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, err, userCred, false)
			continue
		}
		logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, added[i], userCred, true)
		result.Add()
	}

	if result.IsError() {
		return result.AllError()
	}

	log.Infof("Sync %s(%s) custom policies for cloudgroupcache %s result: %s", account.Name, account.Provider, self.Name, result.Result())
	return nil
}

// 将本地的权限推送到云上(覆盖云上设置)
func (self *SCloudgroupcache) SyncSystemCloudpoliciesForCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrap(err, "GetCloudaccount")
	}

	iGroup, err := self.GetICloudgroup()
	if err != nil {
		return errors.Wrap(err, "GetICloudgroup")
	}
	iPolicies, err := iGroup.GetISystemCloudpolicies()
	if err != nil {
		return errors.Wrap(err, "GetISystemCloudpolicies")
	}
	group, err := self.GetCloudgroup()
	if err != nil {
		return errors.Wrap(err, "GetCloudgroup")
	}
	dbPolicies, err := group.GetSystemCloudpolicies()
	if err != nil {
		return errors.Wrap(err, "GetCloudpolicies")
	}

	added := make([]SCloudpolicy, 0)
	commondb := make([]SCloudpolicy, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	removed := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbPolicies, iPolicies, &added, &commondb, &commonext, &removed)
	if err != nil {
		return errors.Wrap(err, "compare.CompareSets")
	}

	result := compare.SyncResult{}

	for i := 0; i < len(removed); i++ {
		err = iGroup.DetachSystemPolicy(removed[i].GetGlobalId())
		if err != nil {
			logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, err, userCred, false)
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = iGroup.AttachSystemPolicy(added[i].ExternalId)
		if err != nil {
			logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, err, userCred, false)
			result.AddError(err)
			continue
		}
		result.Add()
	}

	if result.IsError() {
		return result.AllError()
	}

	log.Infof("Sync %s(%s) system policies for cloudgroupcache %s result: %s", account.Name, account.Provider, self.Name, result.Result())
	return nil
}

func (self *SCloudgroupcache) GetCloudusers() ([]SClouduser, error) {
	sq := CloudgroupUserManager.Query("clouduser_id").Equals("cloudgroup_id", self.CloudgroupId)
	q := ClouduserManager.Query().Equals("cloudaccount_id", self.CloudaccountId).In("id", sq.SubQuery())
	users := []SClouduser{}
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return users, nil
}

// 将本地的用户推送到云上(覆盖云上设置)
func (self *SCloudgroupcache) SyncCloudusersForCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	iGroup, err := self.GetICloudgroup()
	if err != nil {
		return errors.Wrap(err, "GetICloudgroup")
	}
	iUsers, err := iGroup.GetICloudusers()
	if err != nil {
		return errors.Wrap(err, "GetICloudusers")
	}
	dbUsers, err := self.GetCloudusers()
	if err != nil {
		return errors.Wrap(err, "GetCloudusers")
	}

	added := make([]SClouduser, 0)
	commondb := make([]SClouduser, 0)
	commonext := make([]cloudprovider.IClouduser, 0)
	removed := make([]cloudprovider.IClouduser, 0)

	err = compare.CompareSets(dbUsers, iUsers, &added, &commondb, &commonext, &removed)
	if err != nil {
		return errors.Wrap(err, "compare.CompareSets")
	}

	result := compare.SyncResult{}

	for i := 0; i < len(removed); i++ {
		err = iGroup.RemoveUser(removed[i].GetName())
		if err != nil {
			result.DeleteError(err)
			logclient.AddSimpleActionLog(self, logclient.ACT_REMOVE_USER, err, userCred, false)
			continue
		}
		logclient.AddSimpleActionLog(self, logclient.ACT_REMOVE_USER, jsonutils.Marshal(removed[i]), userCred, true)
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = iGroup.AddUser(added[i].GetName())
		if err != nil {
			result.AddError(err)
			logclient.AddSimpleActionLog(self, logclient.ACT_ADD_USER, err, userCred, false)
			continue
		}
		logclient.AddSimpleActionLog(self, logclient.ACT_ADD_USER, jsonutils.Marshal(added[i]), userCred, true)
		result.Add()
	}

	if result.IsError() {
		return result.AllError()
	}

	log.Infof("sync cloudusers for cloudgroupcache %s(%s) result: %s", self.Name, self.Id, result.Result())
	return nil
}
