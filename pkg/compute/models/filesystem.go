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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
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

type SFileSystemManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SZoneResourceBaseManager

	SDeletePreventableResourceBaseManager
}

var FileSystemManager *SFileSystemManager

func init() {
	FileSystemManager = &SFileSystemManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SFileSystem{},
			"file_systems_tbl",
			"file_system",
			"file_systems",
		),
	}
	FileSystemManager.SetVirtualObject(FileSystemManager)
}

type SFileSystem struct {
	db.SSharableVirtualResourceBase
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
	// enum: ["NFS", "SMB", "cpfs"]
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

func (fileSystem *SFileSystem) GetCloudproviderId() string {
	return fileSystem.ManagerId
}

func (manager *SFileSystemManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.FileSystemListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
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
		net, err := validators.ValidateModel(ctx, userCred, NetworkManager, &input.NetworkId)
		if err != nil {
			return input, err
		}
		network := net.(*SNetwork)
		vpc, _ := network.GetVpc()
		input.ManagerId = vpc.ManagerId
		if zone, _ := network.GetZone(); zone != nil {
			input.ZoneId = zone.Id
			input.CloudregionId = zone.CloudregionId
		}
	}
	if len(input.ZoneId) == 0 {
		return input, httperrors.NewMissingParameterError("zone_id")
	}
	_zone, err := validators.ValidateModel(ctx, userCred, ZoneManager, &input.ZoneId)
	if err != nil {
		return input, err
	}
	zone := _zone.(*SZone)
	region, _ := zone.GetRegion()
	input.CloudregionId = region.Id

	if len(input.ManagerId) == 0 {
		sq := CloudproviderManager.Query().Equals("provider", api.CLOUD_PROVIDER_CEPHFS).SubQuery()
		q := CloudproviderRegionManager.Query().Equals("cloudregion_id", input.CloudregionId)
		q = q.Join(sq, sqlchemy.Equals(q.Field("cloudprovider_id"), sq.Field("id")))
		cprgs := []SCloudproviderregion{}
		err = q.All(&cprgs)
		if err != nil {
			return input, err
		}
		if len(cprgs) == 1 {
			input.ManagerId = cprgs[0].CloudproviderId
		} else {
			return input, httperrors.NewMissingParameterError("manager_id")
		}
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

	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (fileSystem *SFileSystem) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	fileSystem.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	fileSystem.StartCreateTask(ctx, userCred, jsonutils.GetAnyString(data, []string{"network_id"}), "")
}

func (fileSystem *SFileSystem) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, networkId string, parentTaskId string) error {
	var err = func() error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(networkId), "network_id")
		task, err := taskman.TaskManager.NewTask(ctx, "FileSystemCreateTask", fileSystem, userCred, params, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		fileSystem.SetStatus(ctx, userCred, api.NAS_STATUS_CREATE_FAILED, err.Error())
		return err
	}
	fileSystem.SetStatus(ctx, userCred, api.NAS_STATUS_CREATING, "")
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
	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	mRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.FileSystemDetails{
			SharableVirtualResourceDetails: virtRows[i],
			CloudregionResourceInfo:        regionRows[i],
			ManagedResourceInfo:            mRows[i],
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
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SFileSystemManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
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

func (region *SCloudregion) GetFileSystems(managerId string) ([]SFileSystem, error) {
	ret := []SFileSystem{}
	q := FileSystemManager.Query().Equals("cloudregion_id", region.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	err := db.FetchModelObjects(FileSystemManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (region *SCloudregion) SyncFileSystems(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	filesystems []cloudprovider.ICloudFileSystem,
	xor bool,
) ([]SFileSystem, []cloudprovider.ICloudFileSystem, compare.SyncResult) {
	lockman.LockRawObject(ctx, region.Id, FileSystemManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, region.Id, FileSystemManager.Keyword())

	result := compare.SyncResult{}

	localFSs := []SFileSystem{}
	remoteFSs := []cloudprovider.ICloudFileSystem{}

	dbFSs, err := region.GetFileSystems(provider.Id)
	if err != nil {
		result.Error(errors.Wrapf(err, "GetFileSystems"))
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
		if !xor {
			err = commondb[i].SyncWithCloudFileSystem(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
		}
		localFSs = append(localFSs, commondb[i])
		remoteFSs = append(remoteFSs, commonext[i])
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		newFs, err := region.newFromCloudFileSystem(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, newFs, added[i], false)
		localFSs = append(localFSs, *newFs)
		remoteFSs = append(remoteFSs, added[i])
		result.Add()
	}

	return localFSs, remoteFSs, result
}

func (fileSystem *SFileSystem) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, fileSystem)
	defer lockman.ReleaseObject(ctx, fileSystem)

	fileSystem.DeletePreventionOff(fileSystem, userCred)

	err := fileSystem.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		return fileSystem.SetStatus(ctx, userCred, api.NAS_STATUS_UNKNOWN, "sync to delete")
	}

	err = fileSystem.RealDelete(ctx, userCred)
	if err != nil {
		return err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    fileSystem,
		Action: notifyclient.ActionSyncDelete,
	})
	return nil
}

func (fileSystem *SFileSystem) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	return fileSystem.StartDeleteTask(ctx, userCred, "")
}

func (fileSystem *SFileSystem) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "FileSystemDeleteTask", fileSystem, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		fileSystem.SetStatus(ctx, userCred, api.NAS_STATUS_DELETE_FAILED, err.Error())
		return nil
	}
	fileSystem.SetStatus(ctx, userCred, api.NAS_STATUS_DELETING, "")
	return nil
}

func (fileSystem *SFileSystem) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (fileSystem *SFileSystem) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	mts, err := fileSystem.GetMountTargets()
	if err != nil {
		return errors.Wrapf(err, "GetMountTargets")
	}
	for i := range mts {
		err = mts[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "mount target %s real delete", mts[i].DomainName)
		}
	}
	return fileSystem.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (fileSystem *SFileSystem) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if fileSystem.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("FileSystem is locked, cannot delete")
	}
	return fileSystem.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (fileSystem *SFileSystem) SyncAllWithCloudFileSystem(ctx context.Context, userCred mcclient.TokenCredential, fs cloudprovider.ICloudFileSystem) error {
	syncFileSystemMountTargets(ctx, userCred, fileSystem, fs, false)
	return fileSystem.SyncWithCloudFileSystem(ctx, userCred, fs)
}

func (fileSystem *SFileSystem) SyncWithCloudFileSystem(ctx context.Context, userCred mcclient.TokenCredential, fs cloudprovider.ICloudFileSystem) error {
	diff, err := db.Update(fileSystem, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(fileSystem, fs.GetName())
			if len(newName) > 0 {
				fileSystem.Name = newName
			}
		}

		fileSystem.Status = fs.GetStatus()
		fileSystem.StorageType = fs.GetStorageType()
		fileSystem.Protocol = fs.GetProtocol()
		fileSystem.Capacity = fs.GetCapacityGb()
		fileSystem.UsedCapacity = fs.GetUsedCapacityGb()
		fileSystem.FileSystemType = fs.GetFileSystemType()
		fileSystem.MountTargetCountLimit = fs.GetMountTargetCountLimit()
		if zoneId := fs.GetZoneId(); len(zoneId) > 0 {
			region, err := fileSystem.GetRegion()
			if err != nil {
				return errors.Wrapf(err, "fileSystem.GetRegion")
			}
			fileSystem.ZoneId, _ = region.getZoneIdBySuffix(zoneId)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    fileSystem,
			Action: notifyclient.ActionSyncUpdate,
		})
	}
	if account := fileSystem.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, fileSystem, fs, account.ReadOnly)
	}

	if provider := fileSystem.GetCloudprovider(); provider != nil {
		SyncCloudProject(ctx, userCred, fileSystem, provider.GetOwnerId(), fs, provider)
	}

	return nil
}

func (region *SCloudregion) getZoneIdBySuffix(zoneId string) (string, error) {
	zones, err := region.GetZones()
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

func (region *SCloudregion) newFromCloudFileSystem(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, fs cloudprovider.ICloudFileSystem) (*SFileSystem, error) {
	nas := SFileSystem{}
	nas.SetModelManager(FileSystemManager, &nas)
	nas.ExternalId = fs.GetGlobalId()
	nas.CloudregionId = region.Id
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
		nas.ZoneId, _ = region.getZoneIdBySuffix(zoneId)
	}
	fileSystem, err := func() (*SFileSystem, error) {
		lockman.LockRawObject(ctx, FileSystemManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, FileSystemManager.Keyword(), "name")

		var err error
		nas.Name, err = db.GenerateName(ctx, FileSystemManager, userCred, fs.GetName())
		if err != nil {
			return nil, errors.Wrapf(err, "db.GenerateName")
		}

		return &nas, FileSystemManager.TableSpec().Insert(ctx, &nas)
	}()
	if err != nil {
		return nil, err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &nas,
		Action: notifyclient.ActionSyncCreate,
	})

	if account, _ := provider.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, fileSystem, fs, account.ReadOnly)
	}

	SyncCloudProject(ctx, userCred, fileSystem, provider.GetOwnerId(), fs, provider)

	return fileSystem, nil
}

// 同步NAS状态
func (fileSystem *SFileSystem) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.FileSystemSyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(fileSystem, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Nas has %d task active, can't sync status", count)
	}

	return nil, fileSystem.StartSyncstatus(ctx, userCred, "")
}

func (fileSystem *SFileSystem) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, fileSystem, "FileSystemSyncstatusTask", parentTaskId)
}

// 设置容量大小(CephFS)
func (fileSystem *SFileSystem) PerformSetQuota(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.FileSystemSetQuotaInput) (jsonutils.JSONObject, error) {
	if input.MaxFiles == nil && input.MaxGb == nil {
		return nil, httperrors.NewMissingParameterError("max_gb")
	}
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(fileSystem, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Nas has %d task active, can't sync status", count)
	}

	return nil, fileSystem.StartSetQuotaTask(ctx, userCred, input)
}

func (fileSystem *SFileSystem) StartSetQuotaTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.FileSystemSetQuotaInput) error {
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "FileSystemSetQuotaTask", fileSystem, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	fileSystem.SetStatus(ctx, userCred, api.NAS_STATUS_EXTENDING, "set quota")
	return task.ScheduleRun(nil)
}

func (fileSystem *SFileSystem) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	provider, err := fileSystem.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "fileSystem.GetDriver")
	}
	if provider.GetFactory().IsOnPremise() {
		return provider.GetOnPremiseIRegion()
	}
	region, err := fileSystem.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "fileSystem.GetRegion")
	}
	iRegion, err := provider.GetIRegionById(region.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "provider.GetIRegionById")
	}
	return iRegion, nil
}

func (fileSystem *SFileSystem) GetICloudFileSystem(ctx context.Context) (cloudprovider.ICloudFileSystem, error) {
	if len(fileSystem.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty externalId")
	}
	iRegion, err := fileSystem.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "fileSystem.GetIRegion")
	}
	return iRegion.GetICloudFileSystemById(fileSystem.ExternalId)
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

func (fileSystem *SFileSystem) doExternalSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	iFs, err := fileSystem.GetICloudFileSystem(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetICloudFileSystem")
	}
	return fileSystem.SyncWithCloudFileSystem(ctx, userCred, iFs)
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

func (fileSystem *SFileSystem) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.FileSystemRemoteUpdateInput) (jsonutils.JSONObject, error) {
	return nil, fileSystem.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
}

func (fileSystem *SFileSystem) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	if replaceTags {
		data.Add(jsonutils.JSONTrue, "replace_tags")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "FileSystemRemoteUpdateTask", fileSystem, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	fileSystem.SetStatus(ctx, userCred, api.NAS_UPDATE_TAGS, "StartRemoteUpdateTask")
	return task.ScheduleRun(nil)
}

func (fileSystem *SFileSystem) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(fileSystem.ExternalId) == 0 || options.Options.KeepTagLocalization {
		return
	}
	if account := fileSystem.GetCloudaccount(); account != nil && account.ReadOnly {
		return
	}
	fileSystem.StartRemoteUpdateTask(ctx, userCred, true, "")
}

func (fileSystem *SFileSystem) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := fileSystem.SSharableVirtualResourceBase.GetShortDesc(ctx)
	region, _ := fileSystem.GetRegion()
	provider := fileSystem.GetCloudprovider()
	info := MakeCloudProviderInfo(region, nil, provider)
	desc.Set("file_system_type", jsonutils.NewString(fileSystem.FileSystemType))
	desc.Set("storage_type", jsonutils.NewString(fileSystem.StorageType))
	desc.Set("protocol", jsonutils.NewString(fileSystem.Protocol))
	desc.Set("capacity", jsonutils.NewInt(fileSystem.Capacity))
	desc.Set("used_capacity", jsonutils.NewInt(fileSystem.UsedCapacity))
	desc.Update(jsonutils.Marshal(&info))
	return desc
}
