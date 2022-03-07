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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	InstanceBackupManager = &SInstanceBackupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SInstanceBackup{},
			"instancebackups_tbl",
			"instancebackup",
			"instancebackups",
		),
	}
	InstanceBackupManager.SetVirtualObject(InstanceBackupManager)
}

type SInstanceBackup struct {
	db.SVirtualResourceBase

	SManagedResourceBase
	SCloudregionResourceBase
	db.SMultiArchResourceBase

	BackupStorageId string `width:"36" charset:"ascii" nullable:"true" create:"required" list:"user" index:"true"`

	GuestId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	// 云主机配置
	ServerConfig jsonutils.JSONObject `nullable:"true" list:"user"`
	// 云主机标签
	ServerMetadata jsonutils.JSONObject `nullable:"true" list:"user"`
	// 安全组
	SecGroups jsonutils.JSONObject `nullable:"true" list:"user"`
	// 秘钥Id
	KeypairId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
	// 操作系统类型
	OsType string `width:"36" charset:"ascii" nullable:"true" list:"user"`
	// 套餐名称
	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`
	// 主机备份容量和
	SizeMb int `nullable:"false" list:"user"`
}

type SInstanceBackupManager struct {
	db.SVirtualResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	db.SMultiArchResourceBaseManager
}

var InstanceBackupManager *SInstanceBackupManager

func (manager *SInstanceBackupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.InstanceBackupListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SMultiArchResourceBaseManager.ListItemFilter(ctx, q, userCred, query.MultiArchResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SMultiArchResourceBaseManager.ListItemFilter")
	}

	guestStr := query.ServerId
	if len(guestStr) > 0 {
		guestObj, err := GuestManager.FetchByIdOrName(userCred, guestStr)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("guests", guestStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("guest_id", guestObj.GetId())
	}

	if len(query.OsType) > 0 {
		q = q.In("os_type", query.OsType)
	}

	return q, nil
}

func (manager *SInstanceBackupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.InstanceBackupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SInstanceBackupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SInstanceBackupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SInstanceBackup) GetGuest() (*SGuest, error) {
	if len(self.GuestId) == 0 {
		return nil, errors.ErrNotFound
	}
	guest := GuestManager.FetchGuestById(self.GuestId)
	if guest == nil {
		return nil, errors.ErrNotFound
	}
	return guest, nil
}

func (self *SInstanceBackup) GetBackupStorage() (*SBackupStorage, error) {
	ibs, err := BackupStorageManager.FetchById(self.BackupStorageId)
	if err != nil {
		return nil, err
	}
	return ibs.(*SBackupStorage), nil
}

func (self *SInstanceBackup) getMoreDetails(userCred mcclient.TokenCredential, out api.InstanceBackupDetails) api.InstanceBackupDetails {
	guest := GuestManager.FetchGuestById(self.GuestId)
	if guest != nil {
		out.Guest = guest.Name
		out.GuestStatus = guest.Status
	}
	backupStorage, _ := self.GetBackupStorage()
	if backupStorage != nil {
		out.BackupStorageName = backupStorage.GetName()
	}
	return out
}

func (manager *SInstanceBackupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.InstanceBackupDetails {
	rows := make([]api.InstanceBackupDetails, len(objs))

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.InstanceBackupDetails{
			VirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:    manRows[i],
		}
		rows[i] = objs[i].(*SInstanceBackup).getMoreDetails(userCred, rows[i])
	}

	return rows
}

func (self *SInstanceBackup) StartCreateInstanceBackupTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.SetStatus(userCred, api.INSTANCE_BACKUP_STATUS_CREATING, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "InstanceBackupCreateTask", self, userCred, nil, parentTaskId, "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (manager *SInstanceBackupManager) fillInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, instanceBackup *SInstanceBackup) {
	instanceBackup.SetModelManager(manager, instanceBackup)
	instanceBackup.ProjectId = guest.ProjectId
	instanceBackup.DomainId = guest.DomainId
	instanceBackup.GuestId = guest.Id
	guestSchedInput := guest.ToSchedDesc()

	host, _ := guest.GetHost()
	instanceBackup.ManagerId = host.ManagerId
	zone, _ := host.GetZone()
	instanceBackup.CloudregionId = zone.CloudregionId

	guestSchedInput.HostId = ""
	guestSchedInput.Project = ""
	guestSchedInput.Domain = ""
	for i := 0; i < len(guestSchedInput.Networks); i++ {
		guestSchedInput.Networks[i].Mac = ""
		guestSchedInput.Networks[i].Address = ""
		guestSchedInput.Networks[i].Address6 = ""
	}
	instanceBackup.ServerConfig = jsonutils.Marshal(guestSchedInput.ServerConfig)
	if len(guest.KeypairId) > 0 {
		instanceBackup.KeypairId = guest.KeypairId
	}
	serverMetadata := jsonutils.NewDict()
	if loginAccount := guest.GetMetadata(ctx, "login_account", nil); len(loginAccount) > 0 {
		loginKey := guest.GetMetadata(ctx, "login_key", nil)
		if len(guest.KeypairId) == 0 && len(loginKey) > 0 {
			passwd, e := utils.DescryptAESBase64(guest.Id, loginKey)
			if e == nil {
				serverMetadata.Set("login_account", jsonutils.NewString(loginAccount))
				serverMetadata.Set("passwd", jsonutils.NewString(passwd))
			}
		} else {
			serverMetadata.Set("login_key", jsonutils.NewString(loginKey))
			serverMetadata.Set("login_account", jsonutils.NewString(loginAccount))
		}
	}
	if osArch := guest.GetMetadata(ctx, "os_arch", nil); len(osArch) > 0 {
		serverMetadata.Set("os_arch", jsonutils.NewString(osArch))
	}
	if osDist := guest.GetMetadata(ctx, "os_distribution", nil); len(osDist) > 0 {
		serverMetadata.Set("os_distribution", jsonutils.NewString(osDist))
	}
	if osName := guest.GetMetadata(ctx, "os_name", nil); len(osName) > 0 {
		serverMetadata.Set("os_name", jsonutils.NewString(osName))
	}
	if osVersion := guest.GetMetadata(ctx, "os_version", nil); len(osVersion) > 0 {
		serverMetadata.Set("os_version", jsonutils.NewString(osVersion))
	}
	secs, _ := guest.GetSecgroups()
	if len(secs) > 0 {
		secIds := make([]string, len(secs))
		for i := 0; i < len(secs); i++ {
			secIds[i] = secs[i].Id
		}
		instanceBackup.SecGroups = jsonutils.Marshal(secIds)
	}
	instanceBackup.OsType = guest.OsType
	instanceBackup.OsArch = guest.OsArch
	instanceBackup.ServerMetadata = serverMetadata
	instanceBackup.InstanceType = guest.InstanceType
}

func (manager *SInstanceBackupManager) CreateInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, name, backupStorageId string) (*SInstanceBackup, error) {
	instanceBackup := &SInstanceBackup{}
	instanceBackup.SetModelManager(manager, instanceBackup)
	instanceBackup.Name = name
	instanceBackup.BackupStorageId = backupStorageId
	manager.fillInstanceBackup(ctx, userCred, guest, instanceBackup)
	// compute size of instanceBackup
	//instanceBackup.SizeMb = guest.getDiskSize()
	err := manager.TableSpec().Insert(ctx, instanceBackup)
	if err != nil {
		return nil, err
	}
	return instanceBackup, nil
}

func (self *SInstanceBackup) ToInstanceCreateInput(sourceInput *api.ServerCreateInput) (*api.ServerCreateInput, error) {

	serverConfig := new(schedapi.ServerConfig)
	if err := self.ServerConfig.Unmarshal(serverConfig); err != nil {
		return nil, errors.Wrap(err, "unmarshal sched input")
	}

	isjs := make([]SInstanceBackupJoint, 0)
	err := InstanceBackupJointManager.Query().Equals("instance_backup_id", self.Id).Asc("disk_index").All(&isjs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch instance backups")
	}

	for i := 0; i < len(serverConfig.Disks); i++ {
		index := serverConfig.Disks[i].Index
		if index < len(isjs) {
			serverConfig.Disks[i].BackupId = isjs[index].DiskBackupId
			serverConfig.Disks[i].ImageId = ""
			serverConfig.Disks[i].SnapshotId = ""
		}
	}

	sourceInput.Disks = serverConfig.Disks
	if sourceInput.VmemSize == 0 {
		sourceInput.VmemSize = serverConfig.Memory
	}
	if sourceInput.VcpuCount == 0 {
		sourceInput.VcpuCount = serverConfig.Ncpu
	}
	if len(self.KeypairId) > 0 {
		sourceInput.KeypairId = self.KeypairId
	}
	if self.SecGroups != nil {
		secGroups := make([]string, 0)
		inputSecgs := make([]string, 0)
		self.SecGroups.Unmarshal(&secGroups)
		for i := 0; i < len(secGroups); i++ {
			_, err := SecurityGroupManager.FetchSecgroupById(secGroups[i])
			if err == nil {
				inputSecgs = append(inputSecgs, secGroups[i])
			}
		}
		sourceInput.Secgroups = inputSecgs
	}
	sourceInput.OsType = self.OsType
	sourceInput.InstanceType = self.InstanceType
	if len(sourceInput.Networks) == 0 {
		sourceInput.Networks = serverConfig.Networks
	}
	return sourceInput, nil
}

func (self *SInstanceBackup) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.Status == api.INSTANCE_SNAPSHOT_START_DELETE || self.Status == api.INSTANCE_SNAPSHOT_RESET {
		return httperrors.NewForbiddenError("can't delete instance snapshot with wrong status")
	}
	return nil
}

func (self *SInstanceBackup) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	return self.StartInstanceBackupDeleteTask(ctx, userCred, "")
}

func (self *SInstanceBackup) StartInstanceBackupDeleteTask(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {

	task, err := taskman.TaskManager.NewTask(ctx, "InstanceBackupDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	self.SetStatus(userCred, api.INSTANCE_BACKUP_STATUS_DELETING, "InstanceBackupDeleteTask")
	task.ScheduleRun(nil)
	return nil
}

func (self *SInstanceBackup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SInstanceBackup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SInstanceBackup) GetRegionDriver() IRegionDriver {
	provider := self.GetProviderName()
	return GetRegionDriver(provider)
}

func (self *SInstanceBackup) GetBackups() ([]SDiskBackup, error) {
	isjq := InstanceBackupJointManager.Query().SubQuery()
	backups := make([]SDiskBackup, 0)
	dq := DiskBackupManager.Query()
	q := dq.Join(isjq, sqlchemy.Equals(dq.Field("id"), isjq.Field("disk_backup_id"))).Filter(
		sqlchemy.Equals(isjq.Field("instance_backup_id"), self.GetId())).Asc(isjq.Field("disk_index"))
	err := db.FetchModelObjects(DiskBackupManager, q, &backups)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return backups, nil
}

func (self *SInstanceBackup) PerformRecovery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InstanceBackupRecoveryInput) (jsonutils.JSONObject, error) {
	return nil, self.StartRecoveryTask(ctx, userCred, "", input.Name)
}

func (self *SInstanceBackup) StartRecoveryTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, serverName string) error {
	self.SetStatus(userCred, api.INSTANCE_BACKUP_STATUS_RECOVERY, "")
	var params *jsonutils.JSONDict
	if serverName != "" {
		params = jsonutils.NewDict()
		params.Set("server_name", jsonutils.NewString(serverName))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "InstanceBackupRecoveryTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SInstanceBackup) PerformPack(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InstanceBackupPackInput) (jsonutils.JSONObject, error) {
	if input.PackageName == "" {
		return nil, httperrors.NewMissingParameterError("miss package_name")
	}
	self.SetStatus(userCred, api.INSTANCE_BACKUP_STATUS_PACK, "")
	params := jsonutils.NewDict()
	params.Set("package_name", jsonutils.NewString(input.PackageName))
	task, err := taskman.TaskManager.NewTask(ctx, "InstanceBackupPackTask", self, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (self *SInstanceBackupManager) PerformCreateFromPackage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InstanceBackupManagerCreateFromPackageInput) (jsonutils.JSONObject, error) {
	if input.Name == "" {
		return nil, httperrors.NewMissingParameterError("miss name")
	}
	if input.PackageName == "" {
		return nil, httperrors.NewMissingParameterError("miss package_name")
	}
	_, err := BackupStorageManager.FetchById(input.BackupStorageId)
	if err != nil {
		return nil, httperrors.NewInputParameterError("unable to fetch backupStorage %s", input.BackupStorageId)
	}
	name, err := db.GenerateName(ctx, self, userCred, input.Name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate name")
	}
	ib, err := self.CreateInstanceBackupFromPackage(ctx, userCred, input.BackupStorageId, name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create instanceBackup")
	}
	params := jsonutils.NewDict()
	params.Set("package_name", jsonutils.NewString(input.PackageName))
	task, err := taskman.TaskManager.NewTask(ctx, "InstanceBackupUnpackTask", ib, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (self *SInstanceBackup) PackMetadata() (*api.InstanceBackupPackMetadata, error) {
	metadata := &api.InstanceBackupPackMetadata{
		OsArch:         self.OsArch,
		ServerConfig:   self.ServerConfig,
		ServerMetadata: self.ServerMetadata,
		SecGroups:      self.SecGroups,
		KeypairId:      self.KeypairId,
		OsType:         self.OsType,
		InstanceType:   self.InstanceType,
		SizeMb:         self.SizeMb,
	}
	dbs, err := self.GetBackups()
	if err != nil {
		return nil, err
	}
	for i := range dbs {
		mt := dbs[i].PackMetadata()
		metadata.DiskMetadatas = append(metadata.DiskMetadatas, *mt)
	}
	return metadata, nil
}

func (manager *SInstanceBackupManager) CreateInstanceBackupFromPackage(ctx context.Context, owner mcclient.TokenCredential, backupStorageId, name string) (*SInstanceBackup, error) {
	ib := &SInstanceBackup{}
	ib.SetModelManager(manager, ib)
	ib.ProjectId = owner.GetProjectId()
	ib.DomainId = owner.GetProjectDomainId()
	ib.Name = name
	ib.BackupStorageId = backupStorageId
	ib.CloudregionId = "default"
	ib.Status = api.INSTANCE_BACKUP_STATUS_CREATING_FROM_PACKAGE
	err := manager.TableSpec().Insert(ctx, ib)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert instance backup")
	}
	return ib, nil
}

func (ib *SInstanceBackup) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InstanceBackupManagerSyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(ib, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("InstanceBackup has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, ib, "InstanceBackupSyncstatusTask", "")
}

func (ib *SInstanceBackup) FillFromPackMetadata(ctx context.Context, userCred mcclient.TokenCredential, diskBackupIds []string, metadata *api.InstanceBackupPackMetadata) (*SInstanceBackup, error) {
	for i, backupId := range diskBackupIds {
		_, err := DiskBackupManager.CreateFromPackMetadata(ctx, userCred, ib.BackupStorageId, backupId, fmt.Sprintf("%s_disk_%d", ib.Name, i), &metadata.DiskMetadatas[i])
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create diskbackup %s", backupId)
		}
		err = InstanceBackupJointManager.CreateJoint(ctx, ib.GetId(), backupId, int8(i))
		if err != nil {
			return nil, errors.Wrapf(err, "unable to CreateJoint for instanceBackup %s and diskBackup %s", ib.GetId(), backupId)
		}
	}
	_, err := db.Update(ib, func() error {
		ib.OsArch = metadata.OsArch
		ib.ServerConfig = metadata.ServerConfig
		ib.ServerMetadata = metadata.ServerMetadata
		ib.SecGroups = metadata.SecGroups
		ib.KeypairId = metadata.KeypairId
		ib.OsType = metadata.OsType
		ib.InstanceType = metadata.InstanceType
		ib.SizeMb = metadata.SizeMb
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ib, nil
}
