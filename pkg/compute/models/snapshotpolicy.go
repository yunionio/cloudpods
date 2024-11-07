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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=snapshotpolicy
// +onecloud:swagger-gen-model-plural=snapshotpolicies
type SSnapshotPolicyManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

type SSnapshotPolicy struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase

	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" default:"default"`

	RetentionDays int `nullable:"false" list:"user" get:"user" create:"required"`

	// 1~7, 1 is Monday
	RepeatWeekdays api.RepeatWeekdays `charset:"utf8" create:"required" list:"user" get:"user"`
	// 0~23
	TimePoints api.TimePoints `charset:"utf8" create:"required" list:"user" get:"user"`
}

var SnapshotPolicyManager *SSnapshotPolicyManager

func init() {
	SnapshotPolicyManager = &SSnapshotPolicyManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SSnapshotPolicy{},
			"snapshot_policies_tbl",
			"snapshotpolicy",
			"snapshotpolicies",
		),
	}
	SnapshotPolicyManager.SetVirtualObject(SnapshotPolicyManager)
}

func (manager *SSnapshotPolicyManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.SSnapshotPolicyCreateInput,
) (*api.SSnapshotPolicyCreateInput, error) {
	if input.RetentionDays < -1 || input.RetentionDays == 0 || input.RetentionDays > options.Options.RetentionDaysLimit {
		return nil, httperrors.NewInputParameterError("Retention days must in 1~%d or -1", options.Options.RetentionDaysLimit)
	}

	var err error
	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}

	input.Status = apis.STATUS_CREATING

	if len(input.CloudregionId) == 0 {
		input.CloudregionId = api.DEFAULT_REGION_ID
	}
	regionObj, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return nil, err
	}
	region := regionObj.(*SCloudregion)

	input, err = region.GetDriver().ValidateCreateSnapshotPolicy(ctx, userCred, region, input)
	if err != nil {
		return nil, err
	}

	err = input.Validate()
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (sp *SSnapshotPolicy) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	sp.StartCreateTask(ctx, userCred)
}

func (sp *SSnapshotPolicy) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyCreateTask", sp, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SSnapshotPolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.SSnapshotPolicyUpdateInput) (*api.SSnapshotPolicyUpdateInput, error) {
	var err error
	input.VirtualResourceBaseUpdateInput, err = self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}

	if input.RetentionDays != nil {
		if *input.RetentionDays < -1 || *input.RetentionDays == 0 || *input.RetentionDays > options.Options.RetentionDaysLimit {
			return nil, httperrors.NewInputParameterError("Retention days must in 1~%d or -1", options.Options.RetentionDaysLimit)
		}
	}

	err = input.Validate()
	if err != nil {
		return nil, err
	}

	return input, nil
}

func (sp *SSnapshotPolicy) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return sp.StartDeleteTask(ctx, userCred)
}

func (sp *SSnapshotPolicy) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	sp.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyDeleteTask", sp, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (manager *SSnapshotPolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SnapshotPolicyDetails {
	rows := make([]api.SnapshotPolicyDetails, len(objs))

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	policyIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.SnapshotPolicyDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regionRows[i],
		}
		policy := objs[i].(*SSnapshotPolicy)
		policyIds[i] = policy.Id
	}

	q := SnapshotPolicyDiskManager.Query().In("snapshotpolicy_id", policyIds)
	pds := []SSnapshotPolicyDisk{}
	err := q.All(&pds)
	if err != nil {
		return rows
	}
	pdMap := map[string][]SSnapshotPolicyDisk{}
	for _, pd := range pds {
		_, ok := pdMap[pd.SnapshotpolicyId]
		if !ok {
			pdMap[pd.SnapshotpolicyId] = []SSnapshotPolicyDisk{}
		}
		pdMap[pd.SnapshotpolicyId] = append(pdMap[pd.SnapshotpolicyId], pd)
	}
	for i := range rows {
		res, _ := pdMap[policyIds[i]]
		rows[i].BindingDiskCount = len(res)
	}

	return rows
}

func (sp *SSnapshotPolicy) ExecuteNotify(ctx context.Context, userCred mcclient.TokenCredential, diskName string) {
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    sp,
		Action: notifyclient.ActionExecute,
		ObjDetailsDecorator: func(ctx context.Context, details *jsonutils.JSONDict) {
			details.Set("disk", jsonutils.NewString(diskName))
		},
	})
}

func (self *SCloudregion) GetSnapshotPolicies(managerId string) ([]SSnapshotPolicy, error) {
	q := SnapshotPolicyManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SSnapshotPolicy{}
	err := db.FetchModelObjects(SnapshotPolicyManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SCloudregion) SyncSnapshotPolicies(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	policies []cloudprovider.ICloudSnapshotPolicy,
	syncOwnerId mcclient.IIdentityProvider,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, SnapshotPolicyManager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, SnapshotPolicyManager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, region.Id))
	result := compare.SyncResult{}

	dbPolicies, err := region.GetSnapshotPolicies(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	added := make([]cloudprovider.ICloudSnapshotPolicy, 0, 1)
	commonext := make([]cloudprovider.ICloudSnapshotPolicy, 0, 1)
	commondb := make([]SSnapshotPolicy, 0, 1)
	removed := make([]SSnapshotPolicy, 0, 1)

	err = compare.CompareSets(dbPolicies, policies, &removed, &commondb, &commonext, &added)
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
			err = commondb[i].SyncWithCloudPolicy(ctx, userCred, provider, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
		}
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		_, err := region.newFromCloudPolicy(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SSnapshotPolicy) SyncWithCloudPolicy(
	ctx context.Context, userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	ext cloudprovider.ICloudSnapshotPolicy,
) error {
	_, err := db.Update(self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, ext.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}

		self.RetentionDays = ext.GetRetentionDays()
		var err error
		self.RepeatWeekdays, err = ext.GetRepeatWeekdays()
		if err != nil {
			return errors.Wrapf(err, "GetRepeatWeekdays")
		}
		self.TimePoints, err = ext.GetTimePoints()
		if err != nil {
			return errors.Wrapf(err, "GetTimePoints")
		}
		self.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "Update")
	}

	syncOwnerId := provider.GetOwnerId()

	SyncCloudProject(ctx, userCred, self, syncOwnerId, ext, provider)
	if account, _ := provider.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}

	err = self.SyncDisks(ctx, userCred, ext)
	if err != nil {
		return errors.Wrapf(err, "SyncDisks")
	}

	return nil
}

func (self *SCloudregion) newFromCloudPolicy(
	ctx context.Context, userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	ext cloudprovider.ICloudSnapshotPolicy,
) (*SSnapshotPolicy, error) {
	policy := &SSnapshotPolicy{}
	policy.SetModelManager(SnapshotPolicyManager, policy)
	policy.CloudregionId = self.Id
	policy.ManagerId = provider.Id
	policy.ExternalId = ext.GetGlobalId()
	policy.RetentionDays = ext.GetRetentionDays()
	var err error
	policy.RepeatWeekdays, err = ext.GetRepeatWeekdays()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRepeatWeekdays")
	}
	policy.TimePoints, err = ext.GetTimePoints()
	if err != nil {
		return nil, errors.Wrapf(err, "GetTimePoints")
	}
	policy.Status = ext.GetStatus()
	policy.Name = ext.GetName()
	syncOwnerId := provider.GetOwnerId()

	err = func() error {
		lockman.LockRawObject(ctx, SnapshotPolicyManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, SnapshotPolicyManager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, SnapshotPolicyManager, syncOwnerId, policy.Name)
		if err != nil {
			return err
		}
		policy.Name = newName

		return SnapshotPolicyManager.TableSpec().Insert(ctx, policy)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}
	SyncCloudProject(ctx, userCred, policy, syncOwnerId, ext, provider)
	syncVirtualResourceMetadata(ctx, userCred, policy, ext, false)

	err = policy.SyncDisks(ctx, userCred, ext)
	if err != nil {
		return nil, errors.Wrapf(err, "SyncDisks")
	}

	return policy, nil
}

func (sp *SSnapshotPolicy) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (sp *SSnapshotPolicy) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := SnapshotPolicyDiskManager.RemoveBySnapshotpolicy(sp.Id)
	if err != nil {
		return errors.Wrapf(err, "delete snapshot policy disks for policy %s", sp.Name)
	}
	return db.DeleteModel(ctx, userCred, sp)
}

func (sp *SSnapshotPolicy) StartBindDisksTask(ctx context.Context, userCred mcclient.TokenCredential, diskIds []string) error {
	sp.SetStatus(ctx, userCred, api.SNAPSHOT_POLICY_APPLY, jsonutils.Marshal(diskIds).String())
	params := jsonutils.Marshal(map[string]interface{}{"disk_ids": diskIds}).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotpolicyBindDisksTask", sp, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (sp *SSnapshotPolicy) PerformBindDisks(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.SnapshotPolicyDisksInput,
) (jsonutils.JSONObject, error) {
	if len(input.Disks) == 0 {
		return nil, httperrors.NewMissingParameterError("disks")
	}
	diskIds := []string{}
	for i := range input.Disks {
		diskObj, err := validators.ValidateModel(ctx, userCred, DiskManager, &input.Disks[i])
		if err != nil {
			return nil, err
		}
		disk := diskObj.(*SDisk)
		if len(sp.ManagerId) > 0 {
			storage, err := disk.GetStorage()
			if err != nil {
				return nil, errors.Wrapf(err, "GetStorage for disk %s", disk.Name)
			}
			if storage.ManagerId != sp.ManagerId {
				return nil, httperrors.NewConflictError("The snapshot policy %s and disk account are different", sp.Name)
			}
			zone, err := storage.GetZone()
			if err != nil {
				return nil, errors.Wrapf(err, "GetZone")
			}
			if sp.CloudregionId != zone.CloudregionId {
				return nil, httperrors.NewConflictError("The snapshot policy %s and the disk are in different region", sp.Name)
			}
		}
		if !utils.IsInStringArray(disk.Id, diskIds) {
			diskIds = append(diskIds, disk.Id)
		}
	}
	return nil, sp.StartBindDisksTask(ctx, userCred, diskIds)
}

func (sp *SSnapshotPolicy) StartUnbindDisksTask(ctx context.Context, userCred mcclient.TokenCredential, diskIds []string) error {
	sp.SetStatus(ctx, userCred, api.SNAPSHOT_POLICY_CANCEL, jsonutils.Marshal(diskIds).String())
	params := jsonutils.Marshal(map[string]interface{}{"disk_ids": diskIds}).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotpolicyUnbindDisksTask", sp, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (sp *SSnapshotPolicy) PerformUnbindDisks(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.SnapshotPolicyDisksInput,
) (jsonutils.JSONObject, error) {
	if len(input.Disks) == 0 {
		return nil, httperrors.NewMissingParameterError("disks")
	}
	diskIds := []string{}
	for i := range input.Disks {
		diskObj, err := validators.ValidateModel(ctx, userCred, DiskManager, &input.Disks[i])
		if err != nil {
			return nil, err
		}
		disk := diskObj.(*SDisk)
		if utils.IsInStringArray(disk.Id, diskIds) {
			diskIds = append(diskIds, disk.Id)
		}
	}
	return nil, sp.StartUnbindDisksTask(ctx, userCred, diskIds)
}

func (self *SSnapshotPolicy) PerformSyncstatus(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if self.CloudregionId == api.DEFAULT_REGION_ID {
		return nil, self.SetStatus(ctx, userCred, apis.STATUS_AVAILABLE, "")
	}
	return nil, self.StartSyncstatusTask(ctx, userCred, "")
}

func (sp *SSnapshotPolicy) StartSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, sp, "SnapshotpolicySyncstatusTask", parentTaskId)
}

// 快照策略列表
func (manager *SSnapshotPolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.SnapshotPolicyListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
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

func (manager *SSnapshotPolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.SnapshotPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	if db.NeedOrderQuery([]string{input.OrderByBindDiskCount}) {
		sdQ := SnapshotPolicyDiskManager.Query()
		sdSQ := sdQ.AppendField(sdQ.Field("snapshotpolicy_id"), sqlchemy.COUNT("disk_count")).GroupBy("snapshotpolicy_id").SubQuery()
		q = q.LeftJoin(sdSQ, sqlchemy.Equals(sdSQ.Field("snapshotpolicy_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(sdSQ.Field("disk_count"))
		q = db.OrderByFields(q, []string{input.OrderByBindDiskCount}, []sqlchemy.IQueryField{q.Field("disk_count")})
	}
	return q, nil
}

func (manager *SSnapshotPolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

func (manager *SSnapshotPolicyManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (self *SSnapshotPolicy) GetISnapshotPolicy(ctx context.Context) (cloudprovider.ICloudSnapshotPolicy, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	return iRegion.GetISnapshotPolicyById(self.ExternalId)
}

func (self *SSnapshotPolicy) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := self.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SSnapshotPolicy) GetCloudprovider() (*SCloudprovider, error) {
	providerObj, err := CloudproviderManager.FetchById(self.ManagerId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById")
	}
	return providerObj.(*SCloudprovider), nil
}

func (self *SSnapshotPolicy) GetProvider(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	manager, err := self.GetCloudprovider()
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	return manager.GetProvider(ctx)
}

func (self *SSnapshotPolicy) GetUnbindDisks(diskIds []string) ([]SDisk, error) {
	sq := SnapshotPolicyDiskManager.Query("disk_id").Equals("snapshotpolicy_id", self.Id).SubQuery()
	q := DiskManager.Query().In("id", diskIds)
	q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
	ret := []SDisk{}
	err := db.FetchModelObjects(DiskManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SSnapshotPolicy) GetBindDisks(diskIds []string) ([]SDisk, error) {
	sq := SnapshotPolicyDiskManager.Query("disk_id").Equals("snapshotpolicy_id", self.Id).SubQuery()
	q := DiskManager.Query().In("id", diskIds)
	q = q.Filter(sqlchemy.In(q.Field("id"), sq))
	ret := []SDisk{}
	err := db.FetchModelObjects(DiskManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SSnapshotPolicy) GetDisks() ([]SDisk, error) {
	sq := SnapshotPolicyDiskManager.Query().Equals("snapshotpolicy_id", self.Id).SubQuery()
	q := DiskManager.Query()
	q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("disk_id")))
	ret := []SDisk{}
	err := db.FetchModelObjects(DiskManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (sp *SSnapshotPolicy) BindDisks(ctx context.Context, disks []SDisk) error {
	for i := range disks {
		spd := &SSnapshotPolicyDisk{}
		spd.SetModelManager(SnapshotPolicyDiskManager, spd)
		spd.DiskId = disks[i].Id
		spd.SnapshotpolicyId = sp.Id
		err := SnapshotPolicyDiskManager.TableSpec().Insert(ctx, spd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sp *SSnapshotPolicy) UnbindDisks(diskIds []string) error {
	vars := []interface{}{sp.Id}
	placeholders := make([]string, len(diskIds))
	for i := range placeholders {
		placeholders[i] = "?"
		vars = append(vars, diskIds[i])
	}
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where snapshotpolicy_id = ? and disk_id in (%s)",
			SnapshotPolicyDiskManager.TableSpec().Name(), strings.Join(placeholders, ","),
		), vars...,
	)
	return err
}

func (sp *SSnapshotPolicy) SyncDisks(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSnapshotPolicy) error {
	extIds, err := ext.GetApplyDiskIds()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported {
			return nil
		}
		return errors.Wrapf(err, "GetApplyDiskIds")
	}
	{
		sq := SnapshotPolicyDiskManager.Query("disk_id").Equals("snapshotpolicy_id", sp.Id).SubQuery()
		q := DiskManager.Query().In("id", sq).NotIn("external_id", extIds)
		needCancel := []SDisk{}
		err = db.FetchModelObjects(DiskManager, q, &needCancel)
		if err != nil {
			return errors.Wrapf(err, "db.FetchModelObjects")
		}
		diskIds := []string{}
		for _, disk := range needCancel {
			diskIds = append(diskIds, disk.Id)
		}
		if len(diskIds) > 0 {
			err = sp.UnbindDisks(diskIds)
			if err != nil {
				return errors.Wrapf(err, "UnbindDisks")
			}
		}
	}
	{
		sq := SnapshotPolicyDiskManager.Query("disk_id").Equals("snapshotpolicy_id", sp.Id).SubQuery()
		storages := StorageManager.Query().Equals("manager_id", sp.ManagerId).SubQuery()
		q := DiskManager.Query()
		q = q.Join(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
		q = q.Filter(
			sqlchemy.AND(
				sqlchemy.NotIn(q.Field("id"), sq),
				sqlchemy.In(q.Field("external_id"), extIds),
			),
		)
		needApply := []SDisk{}
		err = db.FetchModelObjects(DiskManager, q, &needApply)
		if err != nil {
			return errors.Wrapf(err, "db.FetchModelObjects")
		}
		err = sp.BindDisks(ctx, needApply)
		if err != nil {
			return errors.Wrapf(err, "BindDisks")
		}
	}
	return nil
}
