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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAccessGroupManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var AccessGroupManager *SAccessGroupManager

func init() {
	AccessGroupManager = &SAccessGroupManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SAccessGroup{},
			"access_groups_tbl",
			"access_group",
			"access_groups",
		),
	}
	AccessGroupManager.SetVirtualObject(AccessGroupManager)
}

type SAccessGroup struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	SManagedResourceBase

	// 已关联的挂载点数量
	MountTargetCount int `nullable:"false" list:"user" json:"mount_target_count"`

	FileSystemType string `width:"16" charset:"ascii" nullable:"false" index:"true" list:"user"`
	NetworkType    string `width:"8" charset:"ascii" nullable:"false" index:"true" list:"user" default:"vpc"`
}

func (manager *SAccessGroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager SAccessGroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AccessGroupDetails {
	rows := make([]api.AccessGroupDetails, len(objs))
	stdRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.AccessGroupDetails{
			StatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:             manRows[i],
			CloudregionResourceInfo:         regRows[i],
		}
	}
	return rows
}

func (manager *SAccessGroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SAccessGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, nil
}

func (manager *SAccessGroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (self *SAccessGroup) GetChangeOwnerCandidateDomainIds() []string {
	return []string{}
}

func (self *SAccessGroup) GetAccessGroupRules() ([]SAccessGroupRule, error) {
	rules := []SAccessGroupRule{}
	q := AccessGroupRuleManager.Query().Equals("access_group_id", self.Id)
	err := db.FetchModelObjects(AccessGroupRuleManager, q, &rules)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return rules, nil
}

func (manager *SAccessGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.AccessGroupCreateInput) (*api.AccessGroupCreateInput, error) {
	var err error
	if len(input.CloudregionId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudregion_id")
	}

	_, err = validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return nil, err
	}

	_, err = validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
	if err != nil {
		return nil, err
	}
	input.ManagerId = input.CloudproviderId

	input.StatusInfrasResourceBaseCreateInput, err = manager.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	input.Status = apis.STATUS_CREATING
	return input, nil
}

func (self *SAccessGroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.StartCreateTask(ctx, userCred)
}

func (self *SAccessGroup) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "AccessGroupCreateTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SAccessGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SAccessGroup) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "AccessGroupDeleteTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(ctx, userCred, api.ACCESS_GROUP_STATUS_DELETE_FAILED, err.Error())
		return nil
	}
	self.SetStatus(ctx, userCred, api.ACCESS_GROUP_STATUS_DELETING, "")
	return nil
}

func (self *SAccessGroup) GetMountTargets() ([]SMountTarget, error) {
	mts := []SMountTarget{}
	q := MountTargetManager.Query().Equals("access_group_id", self.Id)
	err := db.FetchModelObjects(MountTargetManager, q, &mts)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return mts, nil
}

func (self *SAccessGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SAccessGroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SAccessGroup) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.MountTargetCount > 0 {
		return httperrors.NewNotEmptyError("access group not empty, please delete mount target first")
	}
	return self.SStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

// 同步权限组状态
func (self *SAccessGroup) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.MountTargetSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SAccessGroup) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "AccessGroupSyncstatusTask", parentTaskId)
}

func (self *SCloudregion) GetAccessGroups(managerId string) ([]SAccessGroup, error) {
	q := AccessGroupManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SAccessGroup{}
	err := db.FetchModelObjects(AccessGroupManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudregion) SyncAccessGroups(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	iAccessGroups []cloudprovider.ICloudAccessGroup,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, AccessGroupManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, self.Id, AccessGroupManager.Keyword())

	result := compare.SyncResult{}

	dbGroups, err := self.GetAccessGroups(provider.Id)
	if err != nil {
		result.Error(errors.Wrapf(err, "self.GetAccessGroup"))
		return result
	}

	removed := make([]SAccessGroup, 0)
	commondb := make([]SAccessGroup, 0)
	commonext := make([]cloudprovider.ICloudAccessGroup, 0)
	added := make([]cloudprovider.ICloudAccessGroup, 0)
	err = compare.CompareSets(dbGroups, iAccessGroups, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithAccessGroup(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		err := provider.newFromCloudAccessGroup(ctx, userCred, self, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SCloudprovider) newFromCloudAccessGroup(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, iAccessGroup cloudprovider.ICloudAccessGroup) error {
	ret := &SAccessGroup{}
	ret.SetModelManager(AccessGroupManager, ret)
	ret.CloudregionId = region.Id
	ret.ManagerId = self.Id
	ret.DomainId = self.DomainId
	ret.Status = api.ACCESS_GROUP_STATUS_AVAILABLE
	ret.init(iAccessGroup)
	var err error
	err = func() error {
		lockman.LockRawObject(ctx, AccessGroupManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, AccessGroupManager.Keyword(), "name")

		ret.Name, err = db.GenerateName(ctx, AccessGroupManager, self.GetOwnerId(), iAccessGroup.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}
		return AccessGroupManager.TableSpec().Insert(ctx, ret)
	}()
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}

	rules, err := iAccessGroup.GetRules()
	if err != nil {
		return errors.Wrapf(err, "GetRules")
	}

	ret.SyncRules(ctx, userCred, rules)
	return nil
}

func (self *SAccessGroup) init(iAccessGroup cloudprovider.ICloudAccessGroup) {
	self.Name = iAccessGroup.GetName()
	self.ExternalId = iAccessGroup.GetGlobalId()
	self.NetworkType = iAccessGroup.GetNetworkType()
	self.FileSystemType = iAccessGroup.GetFileSystemType()
	self.MountTargetCount = iAccessGroup.GetMountTargetCount()
	self.ExternalId = iAccessGroup.GetGlobalId()
	self.Description = iAccessGroup.GetDesc()
	self.Status = api.ACCESS_GROUP_STATUS_AVAILABLE
}

func (self *SAccessGroup) syncAccessGroupBaseInfo(ctx context.Context, userCred mcclient.TokenCredential, iAccessGroup cloudprovider.ICloudAccessGroup) error {
	_, err := db.Update(self, func() error {
		self.init(iAccessGroup)
		return nil
	})
	return errors.Wrapf(err, "db.Update")
}

func (self *SAccessGroup) SyncWithAccessGroup(ctx context.Context, userCred mcclient.TokenCredential, iAccessGroup cloudprovider.ICloudAccessGroup) error {
	err := self.syncAccessGroupBaseInfo(ctx, userCred, iAccessGroup)
	if err != nil {
		return errors.Wrapf(err, "syncAccessGroupBaseInfo")
	}
	rules, err := iAccessGroup.GetRules()
	if err != nil {
		return errors.Wrapf(err, "GetRules")
	}
	result := self.SyncRules(ctx, userCred, rules)
	log.Debugf("sync rules for access group %s result: %s", self.Name, result.Result())
	return nil
}

func (self *SAccessGroup) SyncRules(ctx context.Context, userCred mcclient.TokenCredential, rules []cloudprovider.IAccessGroupRule) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, AccessGroupManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, self.Id, AccessGroupManager.Keyword())

	result := compare.SyncResult{}

	dbRules, err := self.GetAccessGroupRules()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetAccessGroupRules"))
		return result
	}

	removed := make([]SAccessGroupRule, 0)
	commondb := make([]SAccessGroupRule, 0)
	commonext := make([]cloudprovider.IAccessGroupRule, 0)
	added := make([]cloudprovider.IAccessGroupRule, 0)
	err = compare.CompareSets(dbRules, rules, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithAccessGroupRule(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		err := self.newAccessGroupRule(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SAccessGroup) newAccessGroupRule(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.IAccessGroupRule) error {
	rule := &SAccessGroupRule{}
	rule.SetModelManager(AccessGroupRuleManager, rule)
	rule.AccessGroupId = self.Id
	rule.Source = ext.GetSource()
	rule.ExternalId = ext.GetGlobalId()
	rule.RWAccessType = string(ext.GetRWAccessType())
	rule.UserAccessType = string(ext.GetUserAccessType())
	rule.Priority = ext.GetPriority()
	return AccessGroupRuleManager.TableSpec().Insert(ctx, rule)
}

func (self *SAccessGroup) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	if len(self.CloudregionId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty cloudregion id")
	}
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetDriver")
	}
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetRegion")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SAccessGroup) GetICloudAccessGroup(ctx context.Context) (cloudprovider.ICloudAccessGroup, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetIRegion")
	}
	iAccessGroup, err := iRegion.GetICloudAccessGroupById(self.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "iRegion.GetICloudAccessGroupById(%s)", self.ExternalId)
	}
	return iAccessGroup, nil
}
