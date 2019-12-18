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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/bitmap"
	"yunion.io/x/onecloud/pkg/util/validate"
)

type SSnapshotPolicyManager struct {
	db.SVirtualResourceBaseManager
}

type SSnapshotPolicy struct {
	db.SVirtualResourceBase
	//db.SExternalizedResourceBase
	//
	//SManagedResourceBase
	//SCloudregionResourceBase

	RetentionDays int `nullable:"false" list:"user" get:"user" create:"required"`

	// 1~7, 1 is Monday
	RepeatWeekdays uint8 `charset:"utf8" create:"required"`
	// 0~23
	TimePoints  uint32            `charset:"utf8" create:"required"`
	IsActivated tristate.TriState `list:"user" get:"user" create:"optional" default:"true"`
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

func (manager *SSnapshotPolicyManager) ValidateListConditions(ctx context.Context, userCred mcclient.TokenCredential,
	query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	input := &api.SSnapshotPolicyCreateInput{}
	err := query.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal input failed %s", err)
	}
	if query.Contains("repeat_weekdays") {
		query.Remove("repeat_weekdays")
		query.Add(jsonutils.NewInt(int64(manager.RepeatWeekdaysParseIntArray(input.RepeatWeekdays))), "repeat_weekdays")
	}
	if query.Contains("time_points") {
		query.Remove("time_points")
		query.Add(jsonutils.NewInt(int64(manager.RepeatWeekdaysParseIntArray(input.RepeatWeekdays))), "time_points")
	}
	return query, nil
}

func (sp *SSnapshotPolicy) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

// ==================================================== fetch ==========================================================
func (manager *SSnapshotPolicyManager) GetSnapshotPoliciesAt(week, timePoint uint32) ([]string, error) {

	q := manager.Query("id")
	q = q.Filter(sqlchemy.Equals(sqlchemy.AND_Val("", q.Field("repeat_weekdays"), 1<<week), 1<<week))
	q = q.Filter(sqlchemy.Equals(sqlchemy.AND_Val("", q.Field("time_points"), 1<<timePoint), 1<<timePoint))
	q = q.Equals("is_activated", true)

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

func (manager *SSnapshotPolicyManager) FetchSnapshotPolicyById(spId string) (*SSnapshotPolicy, error) {
	sp, err := manager.FetchById(spId)
	if err != nil {
		return nil, err
	}
	return sp.(*SSnapshotPolicy), nil
}

func (manager *SSnapshotPolicyManager) FetchAllByIds(spIds []string) ([]SSnapshotPolicy, error) {
	if spIds == nil || len(spIds) == 0 {
		return []SSnapshotPolicy{}, nil
	}
	q := manager.Query().In("id", spIds)
	sps := make([]SSnapshotPolicy, 0, 1)
	err := db.FetchModelObjects(manager, q, &sps)
	if err != nil {
		return nil, err
	}
	return sps, nil
}

// ==================================================== create =========================================================

func (manager *SSnapshotPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	input := &api.SSnapshotPolicyCreateInput{}
	err := data.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal input failed %s", err)
	}
	input.ProjectId = ownerId.GetProjectId()
	input.DomainId = ownerId.GetProjectDomainId()

	err = db.NewNameValidator(manager, ownerId, input.Name, "")
	if err != nil {
		return nil, err
	}

	if input.RetentionDays < -1 || input.RetentionDays == 0 || input.RetentionDays > options.Options.RetentionDaysLimit {
		return nil, httperrors.NewInputParameterError("Retention days must in 1~%d or -1", options.Options.RetentionDaysLimit)
	}

	if len(input.RepeatWeekdays) == 0 {
		return nil, httperrors.NewMissingParameterError("repeat_weekdays")
	}

	if len(input.RepeatWeekdays) > options.Options.RepeatWeekdaysLimit {
		return nil, httperrors.NewInputParameterError("repeat_weekdays only contains %d days at most",
			options.Options.RepeatWeekdaysLimit)
	}
	input.RepeatWeekdays, err = validate.DaysCheck(input.RepeatWeekdays, 1, 7)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}

	if len(input.TimePoints) == 0 {
		return nil, httperrors.NewMissingParameterError("time_points")
	}
	if len(input.TimePoints) > options.Options.TimePointsLimit {
		return nil, httperrors.NewInputParameterError("time_points only contains %d points at most", options.Options.TimePointsLimit)
	}
	input.TimePoints, err = validate.DaysCheck(input.TimePoints, 0, 23)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}

	internalInput := manager.sSnapshotPolicyCreateInputToInternal(input)
	data = internalInput.JSON(internalInput)
	return data, nil
}

// ==================================================== update =========================================================

func (sp *SSnapshotPolicy) AllowPerformUpdate(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	// no fo now
	return false
	//return sp.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, sp, "update")
}

func (sp *SSnapshotPolicy) PerformUpdate(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	//check param
	input := &api.SSnapshotPolicyCreateInput{}
	err := data.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshel input failed %s", err)
	}
	err = sp.UpdateParamCheck(input)
	if err != nil {
		return nil, err
	}
	return nil, sp.StartSnapshotPolicyUpdateTask(ctx, userCred, input)
}

func (sp *SSnapshotPolicy) StartSnapshotPolicyUpdateTask(ctx context.Context, userCred mcclient.TokenCredential,
	input *api.SSnapshotPolicyCreateInput) error {

	params := jsonutils.NewDict()
	params.Add(jsonutils.Marshal(input), "input")
	sp.SetStatus(userCred, api.SNAPSHOT_POLICY_UPDATING, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyUpdateTask", sp, userCred, params,
		"", "", nil); err == nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

// UpdateParamCheck check if update parameters are correct and need to update
func (sp *SSnapshotPolicy) UpdateParamCheck(input *api.SSnapshotPolicyCreateInput) error {
	var err error
	updateNum := 0

	if input.RetentionDays != 0 {
		if input.RetentionDays < -1 || input.RetentionDays > 65535 {
			return httperrors.NewInputParameterError("Retention days must in 1~65535 or -1")
		}
		if input.RetentionDays != sp.RetentionDays {
			updateNum++
		}
	}

	if input.RepeatWeekdays != nil && len(input.RepeatWeekdays) != 0 {
		input.RepeatWeekdays, err = validate.DaysCheck(input.RepeatWeekdays, 1, 7)
		if err != nil {
			return httperrors.NewInputParameterError(err.Error())
		}
		if sp.RepeatWeekdays != SnapshotPolicyManager.RepeatWeekdaysParseIntArray(input.RepeatWeekdays) {
			updateNum++
		}
	}

	if input.TimePoints != nil && len(input.TimePoints) != 0 {
		input.TimePoints, err = validate.DaysCheck(input.TimePoints, 0, 23)
		if err != nil {
			return httperrors.NewInputParameterError(err.Error())
		}
		if sp.TimePoints != SnapshotPolicyManager.TimePointsParseIntArray(input.TimePoints) {
			updateNum++
		}
	}

	if updateNum == 0 {
		return httperrors.NewInputParameterError("Do not need to update")
	}
	return nil
}

// ==================================================== delete =========================================================

func (sp *SSnapshotPolicy) DetachAfterDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := SnapshotPolicyDiskManager.SyncDetachBySnapshotpolicy(ctx, userCred, nil, sp)
	if err != nil {
		return errors.Wrap(err, "detach after delete failed")
	}
	return nil
}

func (sp *SSnapshotPolicy) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.
	JSONObject, data jsonutils.JSONObject) error {

	// check if sp bind to some disks
	sds, err := SnapshotPolicyDiskManager.FetchAllBySnapshotpolicyID(ctx, userCred, sp.GetId())
	if err != nil {
		return errors.Wrap(err, "fetch bind info failed")
	}
	if len(sds) != 0 {
		return httperrors.NewBadRequestError("Couldn't delete snapshot policy binding to disks")
	}
	sp.SetStatus(userCred, api.SNAPSHOT_POLICY_DELETING, "")
	return sp.StartSnapshotPolicyDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (sp *SSnapshotPolicy) StartSnapshotPolicyDeleteTask(ctx context.Context, userCred mcclient.TokenCredential,
	params *jsonutils.JSONDict, parentTaskId string) error {

	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyDeleteTask", sp, userCred, params,
		parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (sp *SSnapshotPolicy) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) *jsonutils.JSONDict {
	extraDict := sp.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	ret, _ := sp.getMoreDetails(ctx, userCred, extraDict)
	return ret
}

func (sp *SSnapshotPolicy) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extraDict, err := sp.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return sp.getMoreDetails(ctx, userCred, extraDict)
}

func (sp *SSnapshotPolicy) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential,
	extraDict *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	ret := extraDict
	// more
	weekdays := SnapshotPolicyManager.RepeatWeekdaysToIntArray(sp.RepeatWeekdays)
	timePoints := SnapshotPolicyManager.TimePointsToIntArray(sp.TimePoints)
	ret.Add(jsonutils.Marshal(weekdays), "repeat_weekdays")
	ret.Add(jsonutils.Marshal(timePoints), "time_points")
	count, err := SnapshotPolicyDiskManager.FetchDiskCountBySPID(sp.Id)
	if err != nil {
		return ret, err
	}
	ret.Add(jsonutils.NewInt(int64(count)), "binding_disk_count")
	return ret, nil
}

// ==================================================== sync ===========================================================
func (manager *SSnapshotPolicyManager) SyncSnapshotPolicies(ctx context.Context, userCred mcclient.TokenCredential,
	provider *SCloudprovider, region *SCloudregion, cloudSPs []cloudprovider.ICloudSnapshotPolicy,
	syncOwnerId mcclient.IIdentityProvider) compare.SyncResult {

	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	syncResult := compare.SyncResult{}

	// Fetch allsnapshotpolicy caches
	spCaches, err := SnapshotPolicyCacheManager.FetchAllByRegionProvider(region.GetId(), provider.GetId())
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	spIdSet, spIds := make(map[string]struct{}), make([]string, 0, 2)
	for _, spCache := range spCaches {
		if _, ok := spIdSet[spCache.SnapshotpolicyId]; !ok {
			spIds = append(spIds, spCache.SnapshotpolicyId)
			spIdSet[spCache.SnapshotpolicyId] = struct{}{}
		}
	}

	// Fetch allsnapshotpolicy of caches above
	snapshotPolicies, err := manager.FetchAllByIds(spIds)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	// structure two sets (externalID, snapshotpolicyCache), (snapshotPolicyID, snapshotPolicy)
	spSet, spCacheSet := make(map[string]*SSnapshotPolicy), make(map[string]*SSnapshotPolicyCache)
	for i := range snapshotPolicies {
		spSet[snapshotPolicies[i].GetId()] = &snapshotPolicies[i]
	}
	for i := range spCaches {
		externalId := spCaches[i].ExternalId
		if len(externalId) != 0 {
			spCacheSet[spCaches[i].ExternalId] = &spCaches[i]
		}
	}

	// start sync
	// add forsnapshotpolicy and cache
	// delete for snapshotpolicy cache
	added := make([]cloudprovider.ICloudSnapshotPolicy, 0, 1)
	commonext := make([]cloudprovider.ICloudSnapshotPolicy, 0, 1)
	commondb := make([]*SSnapshotPolicyCache, 0, 1)
	removed := make([]*SSnapshotPolicyCache, 0, 1)
	for _, cloudSP := range cloudSPs {
		spCache, ok := spCacheSet[cloudSP.GetGlobalId()]
		if !ok {
			added = append(added, cloudSP)
			continue
		}
		snapshotPolicy := spSet[spCache.SnapshotpolicyId]
		if !snapshotPolicy.Equals(cloudSP) {
			removed = append(removed, spCache)
			added = append(added, cloudSP)
		} else {
			commondb = append(commondb, spCache)
			commonext = append(commonext, cloudSP)
		}
		delete(spCacheSet, cloudSP.GetGlobalId())
	}

	for _, v := range spCacheSet {
		removed = append(removed, v)
	}

	for i := range removed {
		// changesnapshotpolicy cache
		err := removed[i].RealDetele(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		}
	}

	syncResult = manager.allNewFromCloudSnapshotPolicy(ctx, userCred, added, region, syncOwnerId, provider, syncResult)

	for i := range commondb {
		_, err = db.Update(commondb[i], func() error {
			commondb[i].Status = api.SNAPSHOT_POLICY_CACHE_STATUS_READY
			if len(commonext[i].GetName()) == 0 {
				commondb[i].Name = commonext[i].GetId()
			} else {
				commondb[i].Name = commonext[i].GetName()
			}
			return nil
		})
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	return syncResult
}

func (manager *SSnapshotPolicyManager) allNewFromCloudSnapshotPolicy(
	ctx context.Context, userCred mcclient.TokenCredential, added []cloudprovider.ICloudSnapshotPolicy,
	region *SCloudregion, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider,
	syncResult compare.SyncResult) compare.SyncResult {

	var snapshotpolicyCluster map[uint64][]*SSnapshotPolicy

	if len(added) > 5 {
		// the number of added is large
		// fetch all snapshotpolicy
		q := SnapshotPolicyManager.Query()
		allSnapshotPolicies := make([]SSnapshotPolicy, 0, 10)
		err := q.All(&allSnapshotPolicies)
		if err != nil {
			syncResult.Error(err)
			return syncResult
		}
		// cluster snapshotpolicy
		snapshotpolicyCluster := make(map[uint64][]*SSnapshotPolicy)
		for i := range allSnapshotPolicies {
			key := allSnapshotPolicies[i].Key()
			list, ok := snapshotpolicyCluster[key]
			if !ok {
				list = make([]*SSnapshotPolicy, 0, 1)
			}
			list = append(list, &allSnapshotPolicies[i])
			// sliceHeader change
			snapshotpolicyCluster[key] = list
		}
	}

	for i := range added {
		local, err := manager.newFromCloudSnapshotPolicy(ctx, userCred, snapshotpolicyCluster, added[i], region,
			syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (manager *SSnapshotPolicyManager) newFromCloudSnapshotPolicy(
	ctx context.Context, userCred mcclient.TokenCredential, snapshotpolicyCluster map[uint64][]*SSnapshotPolicy,
	ext cloudprovider.ICloudSnapshotPolicy, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider,
) (*SSnapshotPolicy, error) {

	snapshotPolicyTmp := SSnapshotPolicy{}
	snapshotPolicyTmp.RetentionDays = ext.GetRetentionDays()
	arw, err := ext.GetRepeatWeekdays()
	if err != nil {
		return nil, err
	}
	snapshotPolicyTmp.RepeatWeekdays = SnapshotPolicyManager.RepeatWeekdaysParseIntArray(arw)
	atp, err := ext.GetTimePoints()
	if err != nil {
		return nil, err
	}
	snapshotPolicyTmp.TimePoints = SnapshotPolicyManager.TimePointsParseIntArray(atp)
	snapshotPolicyTmp.IsActivated = tristate.NewFromBool(ext.IsActivated())

	extProjectId := SnapshotPolicyManager.FetchProjectId(ctx, userCred, syncOwnerId, ext, provider.GetId())

	var snapshotPolicy *SSnapshotPolicy

	// find suitable snapshotpolicy
	if snapshotpolicyCluster == nil {
		q := manager.Query().Equals("repeat_weekdays", snapshotPolicyTmp.RepeatWeekdays).Equals("time_points",
			snapshotPolicyTmp.TimePoints).Equals("retention_days", snapshotPolicyTmp.RetentionDays).Equals(
			"is_activated", snapshotPolicyTmp.IsActivated.Bool()).Equals("tenant_id", extProjectId)
		count, err := q.CountWithError()
		if err != nil {
			return nil, err
		}
		if count > 0 {
			snapshotPolicy = &SSnapshotPolicy{}
			err = q.First(snapshotPolicy)
			if err != nil {
				return nil, err
			}
			snapshotPolicy.SetModelManager(manager, snapshotPolicy)
		}
	} else {
		extkey := snapshotPolicyTmp.Key()
		if list, ok := snapshotpolicyCluster[extkey]; ok {
			// find first snapshotpolicy enough to rebase
			for _, sp := range list {
				if sp.ProjectId == extProjectId {
					snapshotPolicy = sp
					break
				}
			}
		}
	}

	// no such suitable snapshotpolicy
	if snapshotPolicy == nil {
		snapshotPolicyTmp.SetModelManager(manager, &snapshotPolicyTmp)
		newName, err := db.GenerateName(manager, syncOwnerId, ext.GetName())
		if err != nil {
			return nil, err
		}
		snapshotPolicyTmp.Name = newName
		snapshotPolicyTmp.Status = ext.GetStatus()

		err = manager.TableSpec().Insert(&snapshotPolicyTmp)
		if err != nil {
			log.Errorf("newFromCloudEip fail %s", err)
			return nil, err
		}
		// sync project
		SyncCloudProject(userCred, &snapshotPolicyTmp, syncOwnerId, ext, provider.GetId())
		// update snapshotpolicyCluster
		if snapshotpolicyCluster != nil {
			key := snapshotPolicyTmp.Key()
			list, ok := snapshotpolicyCluster[key]
			if !ok {
				list = make([]*SSnapshotPolicy, 0)
			}
			list = append(list, &snapshotPolicyTmp)
			snapshotpolicyCluster[key] = list
		}
		snapshotPolicy = &snapshotPolicyTmp
	}

	// add cache
	_, err = SnapshotPolicyCacheManager.NewCacheWithExternalId(ctx, userCred, snapshotPolicy.GetId(),
		ext.GetGlobalId(), region.GetId(), provider.GetId(), ext.GetName())
	if err != nil {
		//snapshotpolicy has been exist so that created is successful although cache created is fail.
		// disk will be sync aftersnapshotpolicy sync, cache must be right so that this sync is fail
		log.Errorf("snapshotpolicy %s created successfully but corresponding cache created fail", snapshotPolicy.GetId())
		return nil, errors.Wrapf(err, "snapshotpolicy %s created successfully but corresponding cache created fail",
			snapshotPolicy.GetId())
	}

	db.OpsLog.LogEvent(snapshotPolicy, db.ACT_CREATE, snapshotPolicy.GetShortDesc(ctx), userCred)
	return snapshotPolicy, nil
}

func (spm *SSnapshotPolicyManager) FetchProjectId(ctx context.Context, userCred mcclient.TokenCredential,
	syncOwnerId mcclient.IIdentityProvider, cloudSP cloudprovider.ICloudSnapshotPolicy, managerId string) string {
	var newOwnerId mcclient.IIdentityProvider
	if extProjectId := cloudSP.GetProjectId(); len(extProjectId) > 0 {
		extProject, err := ExternalProjectManager.GetProject(extProjectId, managerId)
		if err != nil {
			log.Errorln(err)
		} else {
			newOwnerId = extProject.GetOwnerId()
		}
	}
	if newOwnerId == nil && syncOwnerId != nil && len(syncOwnerId.GetProjectId()) > 0 {
		newOwnerId = syncOwnerId
	}
	if newOwnerId == nil {
		newOwnerId = userCred
	}
	return newOwnerId.GetProjectId()
}

func (sp *SSnapshotPolicy) Key() uint64 {
	var key uint64
	key |= (uint64(sp.RepeatWeekdays) << 56) | (uint64(sp.TimePoints) << 24)
	// that sp.RetentionDays is -1 means permanent retention, sp.RetentionDays+1 must be less than 2^23
	r := sp.RetentionDays + 1&(1<<24-1)
	key |= uint64(r) << 1
	if sp.IsActivated.IsTrue() {
		key |= 1
	}
	return key
}

func (sp *SSnapshotPolicy) Equals(cloudSP cloudprovider.ICloudSnapshotPolicy) bool {
	rws, err := cloudSP.GetRepeatWeekdays()
	if err != nil {
		return false
	}
	tps, err := cloudSP.GetTimePoints()
	if err != nil {
		return false
	}
	repeatWeekdays := SnapshotPolicyManager.RepeatWeekdaysParseIntArray(rws)
	timePoints := SnapshotPolicyManager.TimePointsParseIntArray(tps)

	return sp.RetentionDays == cloudSP.GetRetentionDays() && sp.RepeatWeekdays == repeatWeekdays && sp.
		TimePoints == timePoints && sp.IsActivated.Bool() == cloudSP.IsActivated()
}

func (manager *SSnapshotPolicyManager) getProviderSnapshotPolicies(region *SCloudregion, provider *SCloudprovider) ([]SSnapshotPolicy, error) {
	if region == nil && provider == nil {
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

func (sp *SSnapshotPolicy) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (sp *SSnapshotPolicy) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, sp)
}

// ==================================================== utils ==========================================================

func (manager *SSnapshotPolicyManager) sSnapshotPolicyCreateInputToInternal(input *api.SSnapshotPolicyCreateInput,
) *api.SSnapshotPolicyCreateInternalInput {
	ret := api.SSnapshotPolicyCreateInternalInput{
		Meta:          input.Meta,
		Name:          input.Name,
		ProjectId:     input.ProjectId,
		DomainId:      input.DomainId,
		RetentionDays: input.RetentionDays,
	}

	ret.RepeatWeekdays = manager.RepeatWeekdaysParseIntArray(input.RepeatWeekdays)
	ret.TimePoints = manager.TimePointsParseIntArray(input.TimePoints)
	return &ret
}

func (manager *SSnapshotPolicyManager) sSnapshotPolicyCreateInputFromInternal(input *api.
	SSnapshotPolicyCreateInternalInput) *api.SSnapshotPolicyCreateInput {
	return nil
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

func (sp *SSnapshotPolicy) GenerateCreateSpParams() *cloudprovider.SnapshotPolicyInput {
	intWeekdays := SnapshotPolicyManager.RepeatWeekdaysToIntArray(sp.RepeatWeekdays)
	intTimePoints := SnapshotPolicyManager.TimePointsToIntArray(sp.TimePoints)

	return &cloudprovider.SnapshotPolicyInput{
		RetentionDays:  sp.RetentionDays,
		RepeatWeekdays: intWeekdays,
		TimePoints:     intTimePoints,
		PolicyName:     sp.Name,
	}
}

// ==================================================== action =========================================================
func (sp *SSnapshotPolicy) AllowPerformCache(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {

	return sp.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, sp, "cache")
}

func (sp *SSnapshotPolicy) PerformCache(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	regionId := jsonutils.GetAnyString(data, []string{"region_id", "cloudregion_id"})
	if len(regionId) == 0 {
		return nil, httperrors.NewMissingParameterError("region_id or cloudregion_id")
	}
	providerId, err := data.GetString("provider_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("provider_id")
	}
	_, err = SnapshotPolicyCacheManager.NewCache(ctx, userCred, sp.Id, regionId, providerId)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (sp *SSnapshotPolicy) AllowPerformBindDisks(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {

	return sp.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, sp, "bind-disks")
}

func (sp *SSnapshotPolicy) PerformBindDisks(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	disks := jsonutils.GetArrayOfPrefix(data, "disk")
	if len(disks) == 0 {
		return nil, httperrors.NewMissingParameterError("disk.0 disk.1 ... ")
	}

	//database
	diskSlice := make([]*SDisk, len(disks))
	for i := range disks {
		diskId, _ := disks[i].GetString()
		disk := DiskManager.FetchDiskById(diskId)
		if disk == nil {
			return nil, httperrors.NewInputParameterError("no such disk %s", diskId)
		}
		disk.SetModelManager(DiskManager, disk)
		diskSlice[i] = disk
	}

	taskDisk := make([]*SDisk, 0, len(diskSlice))
	taskSpd := make([]*SSnapshotPolicyDisk, 0, len(diskSlice))
	for _, disk := range diskSlice {
		spd, err := SnapshotPolicyDiskManager.newSnapshotpolicyDisk(ctx, userCred, sp, disk)
		if err == ErrExistSD {
			if spd.Status == "init" {
				taskDisk = append(taskDisk, disk)
				taskSpd = append(taskSpd, spd)
			}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("oper for database error")
		}
		taskDisk = append(taskDisk, disk)
		taskSpd = append(taskSpd, spd)
	}

	for i := range taskDisk {
		// field 'need_detach' is not needed, because the the subject is snapshot policy not disk
		taskdata := jsonutils.NewDict()
		taskdata.Add(jsonutils.Marshal(taskSpd[i]), "snapshotPolicyDisk")
		taskdata.Add(jsonutils.Marshal(sp), "snapshotPolicy")
		if task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyApplyTask", taskDisk[i], userCred, nil,
			"", "", nil); err != nil {
			continue
		} else {
			task.ScheduleRun(taskdata)
		}
	}

	return nil, nil
}

func (sp *SSnapshotPolicy) AllowPerformUnbindDisks(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {

	return sp.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, sp, "bind-disks")
}

func (sp *SSnapshotPolicy) PerformUnbindDisks(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	disks := jsonutils.GetArrayOfPrefix(data, "disk")
	if len(disks) == 0 {
		return nil, httperrors.NewMissingParameterError("disk.0 disk.1 ... ")
	}

	diskSlice := make([]*SDisk, len(disks))
	for i := range disks {
		diskId, _ := disks[i].GetString()
		disk := DiskManager.FetchDiskById(diskId)
		if disk == nil {
			return nil, httperrors.NewInputParameterError("no such disk %s", diskId)
		}
		disk.SetModelManager(DiskManager, disk)
		diskSlice[i] = disk
	}

	taskDisk := make([]*SDisk, 0, len(diskSlice))
	taskSpd := make([]*SSnapshotPolicyDisk, 0, len(diskSlice))
	for _, disk := range diskSlice {
		spd, err := SnapshotPolicyDiskManager.FetchBySnapshotPolicyDisk(sp.Id, disk.GetId())
		if err != nil {
			continue
		}
		if spd == nil {
			continue
		}
		taskSpd = append(taskSpd, spd)
		taskDisk = append(taskDisk, disk)
	}

	for i := range taskDisk {
		taskdata := jsonutils.NewDict()
		taskdata.Add(jsonutils.NewString(sp.Id), "snapshot_policy_id")
		taskdata.Add(jsonutils.Marshal(taskSpd[i]), "snapshotPolicyDisk")
		taskSpd[i].SetStatus(userCred, api.SNAPSHOT_POLICY_DISK_DELETING, "")
		if task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyCancelTask", taskDisk[i], userCred, nil, "", "",
			nil); err != nil {
			continue
		} else {
			task.ScheduleRun(taskdata)
		}
	}
	return nil, nil
}
