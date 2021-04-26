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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SFileSystemManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SZoneResourceBaseManager

	SDeletePreventableResourceBaseManager
}

var FileSystemManager *SFileSystemManager

func init() {
	FileSystemManager = &SFileSystemManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SFileSystem{},
			"file_systems_tbl",
			"file_system",
			"file_systems",
		),
	}
	FileSystemManager.SetVirtualObject(FileSystemManager)
}

type SFileSystem struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SBillingResourceBase
	SCloudregionResourceBase
	SZoneResourceBase

	SDeletePreventableResourceBase

	// 文件系统类型
	// enmu: extreme, standard, cpfs
	FileSystemType string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required"`

	// 存储类型
	// enmu: performance, capacity, standard, advance, advance_100, advance_200
	StorageType string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// 协议类型
	// enum: NFS, SMB, cpfs
	Protocol string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// 容量, 单位Gb
	Capacity int64 `nullable:"false" list:"user" create:"optional"`
	// 已使用容量, 单位Gb
	UsedCapacity int64 `nullable:"false" list:"user"`

	// 最多支持挂载点数量, -1代表无限制
	MountTargetCountLimit int `nullable:"false" list:"user" default:"-1"`
}

func (manager *SFileSystemManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SFileSystem) GetCloudproviderId() string {
	return self.ManagerId
}

func (manager *SFileSystemManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.FileSystemListInput,
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

func (man *SFileSystemManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.FileSystemCreateInput) (api.FileSystemCreateInput, error) {
	var err error
	if len(input.NetworkId) > 0 {
		net, err := validators.ValidateModel(userCred, NetworkManager, &input.NetworkId)
		if err != nil {
			return input, err
		}
		network := net.(*SNetwork)
		input.ManagerId = network.GetVpc().ManagerId
		if zone := network.GetZone(); zone != nil {
			input.ZoneId = zone.Id
			input.CloudregionId = zone.CloudregionId
		}
	}
	if len(input.ZoneId) == 0 {
		return input, httperrors.NewMissingParameterError("zone_id")
	}
	_zone, err := validators.ValidateModel(userCred, ZoneManager, &input.ZoneId)
	if err != nil {
		return input, err
	}
	zone := _zone.(*SZone)
	region := zone.GetRegion()
	input.CloudregionId = region.Id

	if len(input.ManagerId) == 0 {
		return input, httperrors.NewMissingParameterError("manager_id")
	}

	if len(input.Duration) > 0 {
		billingCycle, err := billing.ParseBillingCycle(input.Duration)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid duration %s", input.Duration)
		}

		if !utils.IsInStringArray(input.BillingType, []string{billing_api.BILLING_TYPE_PREPAID, billing_api.BILLING_TYPE_POSTPAID}) {
			input.BillingType = billing_api.BILLING_TYPE_PREPAID
		}

		if input.BillingType == billing_api.BILLING_TYPE_PREPAID {
			if !region.GetDriver().IsSupportedBillingCycle(billingCycle, man.KeywordPlural()) {
				return input, httperrors.NewInputParameterError("unsupported duration %s", input.Duration)
			}
		}
		tm := time.Time{}
		input.BillingCycle = billingCycle.String()
		input.ExpiredAt = billingCycle.EndAt(tm)
	}

	input.StatusInfrasResourceBaseCreateInput, err = man.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SFileSystem) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartCreateTask(ctx, userCred, jsonutils.GetAnyString(data, []string{"network_id"}), "")
}

func (self *SFileSystem) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, networkId string, parentTaskId string) error {
	var err = func() error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(networkId), "network_id")
		task, err := taskman.TaskManager.NewTask(ctx, "FileSystemCreateTask", self, userCred, params, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.NAS_STATUS_CREATE_FAILED, err.Error())
		return err
	}
	self.SetStatus(userCred, api.NAS_STATUS_CREATING, "")
	return nil
}

func (manager SFileSystemManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.FileSystemDetails {
	rows := make([]api.FileSystemDetails, len(objs))
	stdRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	mRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.FileSystemDetails{
			StatusInfrasResourceBaseDetails: stdRows[i],
			CloudregionResourceInfo:         regionRows[i],
			ManagedResourceInfo:             mRows[i],
		}
		nas := objs[i].(*SFileSystem)
		zoneIds[i] = nas.ZoneId
	}

	zoneMaps, err := db.FetchIdNameMap2(ZoneManager, zoneIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		rows[i].Zone, _ = zoneMaps[zoneIds[i]]
	}
	return rows
}

func (manager *SFileSystemManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SFileSystemManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SFileSystemManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.FileSystemListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (self *SCloudregion) GetFileSystems() ([]SFileSystem, error) {
	ret := []SFileSystem{}
	q := FileSystemManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(FileSystemManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncFileSystems(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, filesystems []cloudprovider.ICloudFileSystem) ([]SFileSystem, []cloudprovider.ICloudFileSystem, compare.SyncResult) {
	lockman.LockRawObject(ctx, self.Id, "filesystems")
	defer lockman.ReleaseRawObject(ctx, self.Id, "filesystems")

	result := compare.SyncResult{}

	localFSs := []SFileSystem{}
	remoteFSs := []cloudprovider.ICloudFileSystem{}

	dbFSs, err := self.GetFileSystems()
	if err != nil {
		result.Error(errors.Wrapf(err, "self.GetFileSystems"))
		return localFSs, remoteFSs, result
	}

	removed := make([]SFileSystem, 0)
	commondb := make([]SFileSystem, 0)
	commonext := make([]cloudprovider.ICloudFileSystem, 0)
	added := make([]cloudprovider.ICloudFileSystem, 0)
	err = compare.CompareSets(dbFSs, filesystems, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return localFSs, remoteFSs, result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudFileSystem(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		localFSs = append(localFSs, commondb[i])
		remoteFSs = append(remoteFSs, commonext[i])
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		newFs, err := self.newFromCloudFileSystem(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, newFs, added[i])
		localFSs = append(localFSs, *newFs)
		remoteFSs = append(remoteFSs, added[i])
		result.Add()
	}

	return localFSs, remoteFSs, result
}

func (self *SFileSystem) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	self.DeletePreventionOff(self, userCred)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.NAS_STATUS_UNKNOWN, "sync to delete")
	}

	return self.RealDelete(ctx, userCred)
}

func (self *SFileSystem) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SFileSystem) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "FileSystemDeleteTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.NAS_STATUS_DELETE_FAILED, err.Error())
		return nil
	}
	self.SetStatus(userCred, api.NAS_STATUS_DELETING, "")
	return nil
}

func (self *SFileSystem) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SFileSystem) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	mts, err := self.GetMountTargets()
	if err != nil {
		return errors.Wrapf(err, "GetMountTargets")
	}
	for i := range mts {
		err = mts[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "mount target %s real delete", mts[i].DomainName)
		}
	}
	return self.SInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SFileSystem) ValidateDeleteCondition(ctx context.Context) error {
	if self.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("FileSystem is locked, cannot delete")
	}
	return self.SStatusInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SFileSystem) SyncAllWithCloudFileSystem(ctx context.Context, userCred mcclient.TokenCredential, fs cloudprovider.ICloudFileSystem) error {
	syncFileSystemMountTargets(ctx, userCred, self, fs)
	return self.SyncWithCloudFileSystem(ctx, userCred, fs)
}

func (self *SFileSystem) SyncWithCloudFileSystem(ctx context.Context, userCred mcclient.TokenCredential, fs cloudprovider.ICloudFileSystem) error {
	_, err := db.Update(self, func() error {
		self.Status = fs.GetStatus()
		self.StorageType = fs.GetStorageType()
		self.Protocol = fs.GetProtocol()
		self.Capacity = fs.GetCapacityGb()
		self.UsedCapacity = fs.GetUsedCapacityGb()
		self.FileSystemType = fs.GetFileSystemType()
		self.MountTargetCountLimit = fs.GetMountTargetCountLimit()
		if zoneId := fs.GetZoneId(); len(zoneId) > 0 {
			region, err := self.GetRegion()
			if err != nil {
				return errors.Wrapf(err, "self.GetRegion")
			}
			self.ZoneId, _ = region.getZoneIdBySuffix(zoneId)
		}
		return nil
	})
	return errors.Wrapf(err, "db.Update")
}

func (self *SCloudregion) getZoneIdBySuffix(zoneId string) (string, error) {
	zones, err := self.GetZones()
	if err != nil {
		return "", errors.Wrapf(err, "region.GetZones")
	}
	for _, zone := range zones {
		if strings.HasSuffix(zone.ExternalId, zoneId) {
			return zone.Id, nil
		}
	}
	return "", errors.Wrapf(cloudprovider.ErrNotFound, zoneId)
}

func (self *SCloudregion) newFromCloudFileSystem(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, fs cloudprovider.ICloudFileSystem) (*SFileSystem, error) {
	nas := SFileSystem{}
	nas.SetModelManager(FileSystemManager, &nas)
	nas.ExternalId = fs.GetGlobalId()
	nas.CloudregionId = self.Id
	nas.ManagerId = provider.Id
	nas.Status = fs.GetStatus()
	nas.CreatedAt = fs.GetCreatedAt()
	nas.StorageType = fs.GetStorageType()
	nas.Protocol = fs.GetProtocol()
	nas.Capacity = fs.GetCapacityGb()
	nas.UsedCapacity = fs.GetCapacityGb()
	nas.FileSystemType = fs.GetFileSystemType()
	nas.MountTargetCountLimit = fs.GetMountTargetCountLimit()
	if zoneId := fs.GetZoneId(); len(zoneId) > 0 {
		nas.ZoneId, _ = self.getZoneIdBySuffix(zoneId)
	}
	return func() (*SFileSystem, error) {
		lockman.LockRawObject(ctx, FileSystemManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, FileSystemManager.Keyword(), "name")

		var err error
		nas.Name, err = db.GenerateName(ctx, FileSystemManager, userCred, fs.GetName())
		if err != nil {
			return nil, errors.Wrapf(err, "db.GenerateName")
		}

		return &nas, FileSystemManager.TableSpec().Insert(ctx, &nas)
	}()
}

func (self *SFileSystem) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

// 同步NAS状态
func (self *SFileSystem) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.FileSystemSyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Nas has %d task active, can't sync status", count)
	}

	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SFileSystem) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "FileSystemSyncstatusTask", parentTaskId)
}

func (self *SFileSystem) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		return nil, errors.Wrap(err, "CloudregionManager.FetchById")
	}
	return region.(*SCloudregion), nil
}

func (self *SFileSystem) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetDriver")
	}
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetRegion")
	}
	iRegion, err := provider.GetIRegionById(region.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "provider.GetIRegionById")
	}
	return iRegion, nil
}

func (self *SFileSystem) GetICloudFileSystem() (cloudprovider.ICloudFileSystem, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty externalId")
	}
	iRegion, err := self.GetIRegion()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIRegion")
	}
	return iRegion.GetICloudFileSystemById(self.ExternalId)
}

func (manager *SFileSystemManager) getExpiredPostpaids() ([]SFileSystem, error) {
	q := ListExpiredPostpaidResources(manager.Query(), options.Options.ExpiredPrepaidMaxCleanBatchSize)

	fs := make([]SFileSystem, 0)
	err := db.FetchModelObjects(manager, q, &fs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return fs, nil
}

func (self *SFileSystem) doExternalSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	iFs, err := self.GetICloudFileSystem()
	if err != nil {
		return errors.Wrapf(err, "GetICloudFileSystem")
	}
	return self.SyncWithCloudFileSystem(ctx, userCred, iFs)
}

func (manager *SFileSystemManager) DeleteExpiredPostpaids(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	fss, err := manager.getExpiredPostpaids()
	if err != nil {
		log.Errorf("FileSystem getExpiredPostpaids error: %v", err)
		return
	}
	for i := 0; i < len(fss); i += 1 {
		if len(fss[i].ExternalId) > 0 {
			err := fss[i].doExternalSync(ctx, userCred)
			if err == nil && fss[i].IsValidPostPaid() {
				continue
			}
		}
		fss[i].DeletePreventionOff(&fss[i], userCred)
		fss[i].StartDeleteTask(ctx, userCred, "")
	}
}
