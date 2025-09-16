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
	"fmt"

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

// +onecloud:swagger-gen-model-singular=loadbalanceracl
// +onecloud:swagger-gen-model-plural=loadbalanceracls
type SLoadbalancerAclManager struct {
	SLoadbalancerLogSkipper

	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager

	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var LoadbalancerAclManager *SLoadbalancerAclManager

func init() {
	LoadbalancerAclManager = &SLoadbalancerAclManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLoadbalancerAcl{},
			"loadbalanceracls_tbl",
			"loadbalanceracl",
			"loadbalanceracls",
		),
	}
	LoadbalancerAclManager.SetVirtualObject(LoadbalancerAclManager)
}

type SLoadbalancerAcl struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	AclEntries *api.SAclEntries `list:"user" update:"user" create:"required"`
}

func (man *SLoadbalancerAclManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.LoadbalancerAclCreateInput,
) (*api.LoadbalancerAclCreateInput, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	input.Status = apis.STATUS_CREATING
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	if len(input.CloudregionId) == 0 {
		input.CloudregionId = api.DEFAULT_REGION_ID
	}
	regionObj, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return nil, err
	}
	region := regionObj.(*SCloudregion)
	if len(input.CloudproviderId) > 0 {
		providerObj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return nil, err
		}
		input.ManagerId = input.CloudproviderId
		provider := providerObj.(*SCloudprovider)
		if provider.Provider != region.Provider {
			return nil, httperrors.NewConflictError("conflict region %s and cloudprovider %s", region.Name, provider.Name)
		}
	}
	return input, nil
}

func (self *SLoadbalancerAcl) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartCreateTask(ctx, userCred, "")
}

func (lbacl *SLoadbalancerAcl) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclCreateTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lbacl *SLoadbalancerAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LoadbalancerAclUpdateInput) (*api.LoadbalancerAclUpdateInput, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}

	input.SharableVirtualResourceBaseUpdateInput, err = lbacl.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (lbacl *SLoadbalancerAcl) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SSharableVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	lbacl.StartLoadBalancerAclUpdateTask(ctx, userCred, "")
}

func (lbacl *SLoadbalancerAcl) StartLoadBalancerAclUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclUpdateTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "LoadbalancerAclSyncTask")
	}
	return task.ScheduleRun(nil)
}

func (nm *SLoadbalancerAclManager) query(manager db.IModelManager, field string, aclIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("acl_id"),
		sqlchemy.COUNT(field),
	).In("acl_id", aclIds).GroupBy(sq.Field("acl_id")).SubQuery()
}

type SAclUsageCount struct {
	Id string
	api.LoadbalancerAclUsage
}

func (manager *SLoadbalancerAclManager) TotalResourceCount(aclIds []string) (map[string]api.LoadbalancerAclUsage, error) {
	// listener
	listenerSQ := manager.query(LoadbalancerListenerManager, "listener_cnt", aclIds, nil)

	acls := manager.Query().SubQuery()
	aclQ := acls.Query(
		sqlchemy.SUM("lb_listener_count", listenerSQ.Field("listener_cnt")),
	)

	aclQ.AppendField(aclQ.Field("id"))

	aclQ = aclQ.LeftJoin(listenerSQ, sqlchemy.Equals(aclQ.Field("id"), listenerSQ.Field("acl_id")))
	aclQ = aclQ.Filter(sqlchemy.In(aclQ.Field("id"), aclIds)).GroupBy(aclQ.Field("id"))

	aclCount := []SAclUsageCount{}
	err := aclQ.All(&aclCount)
	if err != nil {
		return nil, errors.Wrapf(err, "aclQ.All")
	}

	result := map[string]api.LoadbalancerAclUsage{}
	for i := range aclCount {
		result[aclCount[i].Id] = aclCount[i].LoadbalancerAclUsage
	}

	return result, nil
}

func (manager *SLoadbalancerAclManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerAclDetails {
	rows := make([]api.LoadbalancerAclDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	aclIds := make([]string, len(objs))

	for i := range rows {
		rows[i] = api.LoadbalancerAclDetails{
			SharableVirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:            managerRows[i],
			CloudregionResourceInfo:        regionRows[i],
		}
		acl := objs[i].(*SLoadbalancerAcl)
		aclIds[i] = acl.Id
	}

	usage, err := manager.TotalResourceCount(aclIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}
	for i := range rows {
		rows[i].LoadbalancerAclUsage, _ = usage[aclIds[i]]
	}

	return rows
}

// PerformPatch patches acl entries by adding then deleting the specified acls.
// This is intended mainly for command line operations.
func (lbacl *SLoadbalancerAcl) PerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LoadbalancerAclPatchInput) (*jsonutils.JSONDict, error) {
	entries := *lbacl.AclEntries
	{
		for i := range input.Adds {
			add := input.Adds[i]
			err := add.Validate()
			if err != nil {
				return nil, err
			}
			found := false
			for _, aclEntry := range entries {
				if aclEntry.Cidr == add.Cidr {
					found = true
					aclEntry.Comment = add.Comment
					break
				}
			}
			if !found {
				entries = append(entries, add)
			}
		}
	}
	{
		for i := range input.Dels {
			del := input.Dels[i]
			err := del.Validate()
			if err != nil {
				return nil, err
			}
			for j := len(entries) - 1; j >= 0; j-- {
				aclEntry := entries[j]
				if aclEntry.Cidr == del.Cidr {
					entries = append(entries[:j], entries[j+1:]...)
					break
				}
			}
		}
	}
	diff, err := db.Update(lbacl, func() error {
		lbacl.AclEntries = &entries
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(diff) > 0 {
		db.OpsLog.LogEvent(lbacl, db.ACT_UPDATE, diff, userCred)
		lbacl.StartLoadBalancerAclUpdateTask(ctx, userCred, "")
	}
	return nil, nil
}

func (lbacl *SLoadbalancerAcl) ValidateDeleteCondition(ctx context.Context, info *api.LoadbalancerAclDetails) error {
	if info != nil && info.ListenerCount > 0 {
		return httperrors.NewResourceBusyError("acl %s is still referred to by %d listener", lbacl.Name, info.ListenerCount)
	}
	return lbacl.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, jsonutils.Marshal(info))
}

func (lbacl *SLoadbalancerAcl) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lbacl.RealDelete(ctx, userCred)
}

func (lbacl *SLoadbalancerAcl) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lbacl *SLoadbalancerAcl) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return lbacl.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SLoadbalancerAcl) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (lbacl *SLoadbalancerAcl) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclDeleteTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	lbacl.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

// 负载均衡ACL规则列表
func (manager *SLoadbalancerAclManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SLoadbalancerAclManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SLoadbalancerAclManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancerAclManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
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

func (self *SLoadbalancerAcl) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SLoadbalancerAcl) GetILoadbalancerAcl(ctx context.Context) (cloudprovider.ICloudLoadbalancerAcl, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, err
	}
	return iRegion.GetILoadBalancerAclById(self.ExternalId)
}

func (lbacl *SLoadbalancerAcl) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, lbacl, "LoadbalancerAclSyncstatusTask", "")
}

func (self *SCloudregion) GetLoadbalancerAcls(managerId string) ([]SLoadbalancerAcl, error) {
	q := LoadbalancerAclManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SLoadbalancerAcl{}
	err := db.FetchModelObjects(LoadbalancerAclManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudregion) SyncLoadbalancerAcls(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudLoadbalancerAcl, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, LoadbalancerAclManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, LoadbalancerAclManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbAcls, err := self.GetLoadbalancerAcls(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SLoadbalancerAcl, 0)
	commondb := make([]SLoadbalancerAcl, 0)
	commonext := make([]cloudprovider.ICloudLoadbalancerAcl, 0)
	added := make([]cloudprovider.ICloudLoadbalancerAcl, 0)

	err = compare.CompareSets(dbAcls, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
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
		if !xor {
			err = commondb[i].SyncWithCloudAcl(ctx, userCred, commonext[i], provider)
			if err != nil {
				result.UpdateError(err)
				continue
			}
		}
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		err := self.newFromCloudAcl(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (lbacl *SLoadbalancerAcl) SyncWithCloudAcl(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudLoadbalancerAcl, provider *SCloudprovider) error {
	_, err := db.Update(lbacl, func() error {
		rules := api.SAclEntries{}
		entries := ext.GetAclEntries()
		for _, entry := range entries {
			rules = append(rules, api.SAclEntry{
				Cidr:    entry.CIDR,
				Comment: entry.Comment,
			})
		}
		lbacl.AclEntries = &rules
		lbacl.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return err
	}

	syncVirtualResourceMetadata(ctx, userCred, lbacl, ext, false)
	SyncCloudProject(ctx, userCred, lbacl, provider.GetOwnerId(), ext, provider)

	return nil
}

func (self *SCloudregion) newFromCloudAcl(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudLoadbalancerAcl) error {
	acl := &SLoadbalancerAcl{}
	acl.SetModelManager(LoadbalancerAclManager, acl)
	acl.ExternalId = ext.GetGlobalId()
	acl.CloudregionId = self.Id
	acl.ManagerId = provider.Id
	acl.Name = ext.GetName()
	acl.Status = ext.GetStatus()
	entries := api.SAclEntries{}
	for _, entry := range ext.GetAclEntries() {
		entries = append(entries, api.SAclEntry{
			Cidr:    entry.CIDR,
			Comment: entry.Comment,
		})
	}
	acl.AclEntries = &entries

	err := LoadbalancerAclManager.TableSpec().Insert(ctx, acl)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}

	syncVirtualResourceMetadata(ctx, userCred, acl, ext, false)
	SyncCloudProject(ctx, userCred, acl, provider.GetOwnerId(), ext, provider)

	return nil
}

func (manager *SLoadbalancerAclManager) InitializeData() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set deleted = true where pending_deleted = true",
			manager.TableSpec().Name(),
		),
	)
	return err
}
