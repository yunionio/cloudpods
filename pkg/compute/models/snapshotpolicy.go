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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/bitmap"
)

type SSnapshotPolicyManager struct {
	db.SVirtualResourceBaseManager
}

type SSnapshotPolicy struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	RetentionDays int `nullable:"false" list:"user" get:"user" create:"required"`

	// 0~6, 0 is Monday
	RepeatWeekdays uint8 `charset:"utf8" create:"required"`
	// 0~23
	TimePoints  uint32 `charset:"utf8" create:"required"`
	IsActivated bool   `list:"user" get:"user" create:"optional" default:"true"`
}

var SnapshotPolicyManager *SSnapshotPolicyManager

func init() {
	SnapshotPolicyManager = &SSnapshotPolicyManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SSnapshotPolicy{},
			"snapshotpolicies_tbl",
			"snapshotpolicy",
			"snapshotpolicies",
		),
	}
	SnapshotPolicyManager.SetVirtualObject(SnapshotPolicyManager)
}

func (manager *SSnapshotPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := &api.SSnapshotPolicyCreateInput{}
	err := data.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal input failed %s", err)
	}
	input.ProjectId = ownerId.GetProjectId()
	input.DomainId = ownerId.GetProjectDomainId()

	err = db.NewNameValidator(manager, ownerId, input.Name)
	if err != nil {
		return nil, err
	}

	cloudregionV := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", ownerId)
	err = cloudregionV.Validate(data)
	if err != nil {
		return nil, err
	}
	cloudregion := cloudregionV.Model.(*SCloudregion)
	input.CloudregionId = cloudregion.GetId()

	err = cloudregion.GetDriver().ValidateCreateSnapshotPolicyData(ctx, userCred, input, ownerId, data)
	if err != nil {
		return nil, err
	}
	internalInput := manager.sSnapshotPolicyCreateInputToInternal(input)
	data = internalInput.JSON(internalInput)
	return data, nil
}

func (manager *SSnapshotPolicyManager) sSnapshotPolicyCreateInputToInternal(input *api.SSnapshotPolicyCreateInput) *api.SSnapshotPolicyCreateInternalInput {
	ret := api.SSnapshotPolicyCreateInternalInput{
		Meta:          input.Meta,
		Name:          input.Name,
		ProjectId:     input.ProjectId,
		DomainId:      input.DomainId,
		ManagerId:     input.ManagerId,
		CloudregionId: input.CloudregionId,
		RetentionDays: input.RetentionDays,
	}

	ret.RepeatWeekdays = manager.RepeatWeekdaysParseIntArray(input.RepeatWeekdays)
	ret.TimePoints = manager.TimePointsParseIntArray(input.TimePoints)
	return &ret
}

func (manager *SSnapshotPolicyManager) sSnapshotPolicyCreateInputFromInternal(input *api.SSnapshotPolicyCreateInternalInput) *api.SSnapshotPolicyCreateInput {
	return nil
}

func (self *SSnapshotPolicy) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.StartCreateSnapshotPolicy(ctx, userCred, ownerId, query, data)
}

func (self *SSnapshotPolicy) StartCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyCreateTask", self, userCred, nil, "", "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SSnapshotPolicy) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	self.SetStatus(userCred, api.SNAPSHOT_POLICY_DELETING, "")
	return self.StartSnapshotPolicyDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SSnapshotPolicy) StartSnapshotPolicyDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SSnapshotPolicy) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	ret := self.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	ret.Update(self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query))

	// more
	weekdays := SnapshotPolicyManager.RepeatWeekdaysToIntArray(self.RepeatWeekdays)
	timePoints := SnapshotPolicyManager.TimePointsToIntArray(self.TimePoints)
	ret.Add(jsonutils.Marshal(weekdays), "repeat_weekdays")
	ret.Add(jsonutils.Marshal(timePoints), "time_points")
	return ret
}

func (self *SSnapshotPolicy) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()

	// more
	weekdays := SnapshotPolicyManager.RepeatWeekdaysToIntArray(self.RepeatWeekdays)
	timePoints := SnapshotPolicyManager.TimePointsToIntArray(self.TimePoints)
	ret.Add(jsonutils.Marshal(weekdays), "repeat_weekdays")
	ret.Add(jsonutils.Marshal(timePoints), "time_points")
	return ret, nil
}

func (self *SSnapshotPolicy) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for sp %s: %s", self.Name, err)
	}
	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to find region for sp %s", self.Name)
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SSnapshotPolicyManager) RepeatWeekdaysParseIntArray(nums []int) uint8 {
	return uint8(bitmap.IntArray2Uint(nums))
}

func (self *SSnapshotPolicyManager) RepeatWeekdaysToIntArray(n uint8) []int {
	return bitmap.Uint2IntArray(uint32(n))
}

func (self *SSnapshotPolicyManager) TimePointsParseIntArray(nums []int) uint32 {
	return bitmap.IntArray2Uint(nums)
}

func (self *SSnapshotPolicyManager) TimePointsToIntArray(n uint32) []int {
	return bitmap.Uint2IntArray(n)
}

func (self *SSnapshotPolicy) GenerateCreateSpParams() (*cloudprovider.SnapshotPolicyInput, error) {
	intWeekdays := SnapshotPolicyManager.RepeatWeekdaysToIntArray(self.RepeatWeekdays)
	intTimePoints := SnapshotPolicyManager.TimePointsToIntArray(self.TimePoints)

	return &cloudprovider.SnapshotPolicyInput{
		RetentionDays:  self.RetentionDays,
		RepeatWeekdays: intWeekdays,
		TimePoints:     intTimePoints,
		PolicyName:     self.Name,
	}, nil
}

func (manager *SSnapshotPolicyManager) SyncSnapshotPolicies(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, snapshots []cloudprovider.ICloudSnapshotPolicy, syncOwnerId mcclient.IIdentityProvider) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	syncResult := compare.SyncResult{}
	dbSnapshotPolicies, err := manager.getProviderSnapshotPolicies(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	removed := make([]SSnapshotPolicy, 0)
	commondb := make([]SSnapshotPolicy, 0)
	commonext := make([]cloudprovider.ICloudSnapshotPolicy, 0)
	added := make([]cloudprovider.ICloudSnapshotPolicy, 0)

	err = compare.CompareSets(dbSnapshotPolicies, snapshots, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudSnapshot(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudSnapshotPolicy(ctx, userCred, commonext[i], syncOwnerId, region)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		local, err := manager.newFromCloudSnapshotPolicy(ctx, userCred, added[i], region, syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (manager *SSnapshotPolicyManager) getProviderSnapshotPolicies(region *SCloudregion, provider *SCloudprovider) ([]SSnapshotPolicy, error) {
	if region == nil || provider == nil {
		return nil, fmt.Errorf("Region is nil or provider is nil")
	}
	snapshotPolicies := make([]SSnapshotPolicy, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id)
	err := db.FetchModelObjects(manager, q, &snapshotPolicies)
	if err != nil {
		return nil, err
	}
	return snapshotPolicies, nil
}

func (self *SSnapshotPolicy) syncRemoveCloudSnapshot(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		err = self.SetStatus(userCred, api.SNAPSHOT_POLICY_UNKNOWN, "sync to delete")
	} else {
		err = self.RealDelete(ctx, userCred)
	}
	return err
}

func (self *SSnapshotPolicy) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSnapshotPolicy) SyncWithCloudSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSnapshotPolicy, ownerId mcclient.IIdentityProvider, region *SCloudregion) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = ext.GetName()
		self.Status = ext.GetStatus()
		self.RetentionDays = ext.GetRetentionDays()

		arw, err := ext.GetRepeatWeekdays()
		if err != nil {
			return err
		}
		self.RepeatWeekdays = SnapshotPolicyManager.RepeatWeekdaysParseIntArray(arw)

		atp, err := ext.GetTimePoints()
		if err != nil {
			return err
		}
		self.TimePoints = SnapshotPolicyManager.TimePointsParseIntArray(atp)
		return nil
	})
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	SyncCloudProject(userCred, self, ownerId, ext, self.ManagerId)
	return err
}

func (manager *SSnapshotPolicyManager) newFromCloudSnapshotPolicy(
	ctx context.Context, userCred mcclient.TokenCredential,
	ext cloudprovider.ICloudSnapshotPolicy, region *SCloudregion,
	syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider,
) (*SSnapshotPolicy, error) {

	snapshotPolicy := SSnapshotPolicy{}
	snapshotPolicy.SetModelManager(manager, &snapshotPolicy)

	newName, err := db.GenerateName(manager, syncOwnerId, ext.GetName())
	if err != nil {
		return nil, err
	}
	snapshotPolicy.Name = newName
	snapshotPolicy.Status = ext.GetStatus()
	snapshotPolicy.ExternalId = ext.GetGlobalId()
	snapshotPolicy.ManagerId = provider.Id
	snapshotPolicy.CloudregionId = region.Id
	snapshotPolicy.RetentionDays = ext.GetRetentionDays()
	arw, err := ext.GetRepeatWeekdays()
	if err != nil {
		return nil, err
	}
	snapshotPolicy.RepeatWeekdays = SnapshotPolicyManager.RepeatWeekdaysParseIntArray(arw)
	atp, err := ext.GetTimePoints()
	if err != nil {
		return nil, err
	}
	snapshotPolicy.TimePoints = SnapshotPolicyManager.TimePointsParseIntArray(atp)

	err = manager.TableSpec().Insert(&snapshotPolicy)
	if err != nil {
		log.Errorf("newFromCloudEip fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &snapshotPolicy, syncOwnerId, ext, snapshotPolicy.ManagerId)
	db.OpsLog.LogEvent(&snapshotPolicy, db.ACT_CREATE, snapshotPolicy.GetShortDesc(ctx), userCred)
	return &snapshotPolicy, nil
}

func (self *SSnapshotPolicy) AllowPerformApplyToDisks(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "apply-to-disks")
}

func (self *SSnapshotPolicy) PerformApplyToDisks(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	diskIds, err := self.preCheck(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	return nil, self.StartApplySnapshotPolicyToDisks(ctx, userCred, diskIds)
}

func (self *SSnapshotPolicy) StartApplySnapshotPolicyToDisks(ctx context.Context, userCred mcclient.TokenCredential, diskIds []string) error {
	params := jsonutils.NewDict()
	params.Set("disk_ids", jsonutils.Marshal(diskIds))
	if task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyApplyTask", self, userCred, params, "", "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SSnapshotPolicy) preCheck(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) ([]string, error) {
	if self.Status != api.SNAPSHOT_POLICY_READY {
		return nil, httperrors.NewInvalidStatusError("Snapshot policy status %s can't do apply", self.Status)
	}
	jsonDiskIds, err := data.Get("disks")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disks")
	}
	ids, ok := jsonDiskIds.(*jsonutils.JSONArray)
	if !ok {
		return nil, httperrors.NewInputParameterError("disk_ids %s", jsonDiskIds)
	}
	diskIds := ids.GetStringArray()
	disks := make([]string, 0)
	err = DiskManager.Query("id").Equals("cloudregion_id", self.CloudregionId).
		Equals("manager_id", self.ManagerId).In("id", diskIds).All(&disks)
	if err != nil {
		return nil, httperrors.NewInternalServerError("Query disks error %s", err)
	}
	if len(disks) < len(diskIds) {
		notFoundDisks := make([]string, 0)
		for _, id := range diskIds {
			if !utils.IsInStringArray(id, disks) {
				notFoundDisks = append(notFoundDisks, id)
			}
		}
		return nil, httperrors.NewNotFoundError("Disks %v not found", notFoundDisks)
	}
	return diskIds, nil
}

func (manager *SSnapshotPolicyManager) GetSnapshotPoliciesAt(week, timePoint uint32) ([]string, error) {

	q := manager.Query("id")
	q = q.Filter(sqlchemy.Equals(sqlchemy.AND_Val("", q.Field("repeat_weekdays"), 1<<week), 1<<week))
	q = q.Filter(sqlchemy.Equals(sqlchemy.AND_Val("", q.Field("time_points"), 1<<timePoint), 1<<timePoint))
	q = q.Equals("is_activated", true)
	q.DebugQuery()

	sps := make([]SSnapshotPolicy, 0)
	err := q.All(&sps)
	if err != nil {
		return nil, err
	}
	if len(sps) > 0 {
		ret := make([]string, len(sps))
		for i := 0; i < len(sps); i++ {
			ret[i] = sps[i].Id
		}
		return ret, nil
	}
	return nil, nil
}

func (manager *SSnapshotPolicyManager) FetchSnapshotPolicyById(spId string) *SSnapshotPolicy {
	sp, err := manager.FetchById(spId)
	if err != nil {
		log.Errorf("FetchBId fail %s", err)
		return nil
	}
	return sp.(*SSnapshotPolicy)
}
