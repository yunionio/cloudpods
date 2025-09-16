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
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
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

	db.SEncryptedResource

	BackupStorageId string `width:"36" charset:"ascii" nullable:"true" create:"required" list:"user" index:"true"`

	GuestId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" index:"true"`
	// 云主机配置 corresponds to api.ServerCreateInput
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

// +onecloud:swagger-gen-model-singular=instancebackup
// +onecloud:swagger-gen-model-plural=instancebackups
type SInstanceBackupManager struct {
	db.SVirtualResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	db.SMultiArchResourceBaseManager
	db.SEncryptedResourceManager
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
		guestObj, err := GuestManager.FetchByIdOrName(ctx, userCred, guestStr)
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

	if db.NeedOrderQuery([]string{query.OrderByGuest}) {
		gQ := GuestManager.Query()
		gSQ := gQ.AppendField(gQ.Field("name").Label("guest_name"), gQ.Field("id")).SubQuery()
		q = q.LeftJoin(gSQ, sqlchemy.Equals(gSQ.Field("id"), q.Field("guest_id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(gSQ.Field("guest_name"))
		q = db.OrderByFields(q, []string{query.OrderByGuest}, []sqlchemy.IQueryField{q.Field("guest_name")})
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
	backups, _ := self.GetBackups()
	out.DiskBackups = []api.SSimpleBackup{}
	for i := 0; i < len(backups); i++ {
		out.DiskBackups = append(out.DiskBackups, api.SSimpleBackup{
			Id:           backups[i].Id,
			Name:         backups[i].Name,
			SizeMb:       backups[i].SizeMb,
			DiskSizeMb:   backups[i].DiskSizeMb,
			DiskType:     backups[i].DiskType,
			Status:       backups[i].Status,
			EncryptKeyId: backups[i].EncryptKeyId,
			CreatedAt:    backups[i].CreatedAt,
		})
	}
	out.Size = self.SizeMb * 1024 * 1024
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
	encRows := manager.SEncryptedResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.InstanceBackupDetails{
			VirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:    manRows[i],

			EncryptedResourceDetails: encRows[i],
		}
		rows[i] = objs[i].(*SInstanceBackup).getMoreDetails(userCred, rows[i])
	}

	return rows
}

func (self *SInstanceBackup) StartCreateInstanceBackupTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.SetStatus(ctx, userCred, api.INSTANCE_BACKUP_STATUS_CREATING, "")
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

	// inherit encrypt_key_id from guest
	instanceBackup.EncryptKeyId = guest.EncryptKeyId

	host, _ := guest.GetHost()
	instanceBackup.ManagerId = host.ManagerId
	zone, _ := host.GetZone()
	instanceBackup.CloudregionId = zone.CloudregionId

	createInput := guest.ToCreateInput(ctx, userCred)
	createInput.ProjectId = guest.ProjectId
	createInput.ProjectDomainId = guest.DomainId
	for i := 0; i < len(createInput.Networks); i++ {
		createInput.Networks[i].Mac = ""
		createInput.Networks[i].Address = ""
		createInput.Networks[i].Address6 = ""
	}
	instanceBackup.ServerConfig = jsonutils.Marshal(createInput)
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

	createInput := new(api.ServerCreateInput)
	createInput.ServerConfigs = new(api.ServerConfigs)
	if err := self.ServerConfig.Unmarshal(createInput); err != nil {
		return nil, errors.Wrap(err, "unmarshal sched input")
	}

	isjs := make([]SInstanceBackupJoint, 0)
	err := InstanceBackupJointManager.Query().Equals("instance_backup_id", self.Id).Asc("disk_index").All(&isjs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch instance backups")
	}

	for i := 0; i < len(createInput.Disks); i++ {
		index := createInput.Disks[i].Index
		if index < len(isjs) {
			createInput.Disks[i].BackupId = isjs[index].DiskBackupId
			createInput.Disks[i].ImageId = ""
			createInput.Disks[i].SnapshotId = ""
			if i < len(sourceInput.Disks) {
				createInput.Disks[i].Backend = sourceInput.Disks[i].Backend
			}
		}
	}

	sourceInput.Disks = createInput.Disks
	if sourceInput.VmemSize == 0 {
		sourceInput.VmemSize = createInput.VmemSize
	}
	if sourceInput.VcpuCount == 0 {
		sourceInput.VcpuCount = createInput.VcpuCount
	}
	if len(sourceInput.OsArch) == 0 {
		sourceInput.OsArch = createInput.OsArch
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
		sourceInput.Networks = createInput.Networks
	}
	if sourceInput.Vga == "" {
		sourceInput.Vga = createInput.Vga
	}
	if sourceInput.Vdi == "" {
		sourceInput.Vdi = createInput.Vdi
	}
	if sourceInput.Vdi == "" {
		sourceInput.Vdi = createInput.Vdi
	}
	if sourceInput.Bios == "" {
		sourceInput.Bios = createInput.Bios
	}
	if sourceInput.BootOrder == "" {
		sourceInput.BootOrder = createInput.BootOrder
	}
	if sourceInput.ShutdownBehavior == "" {
		sourceInput.ShutdownBehavior = createInput.ShutdownBehavior
	}
	if sourceInput.IsolatedDevices == nil {
		sourceInput.IsolatedDevices = createInput.IsolatedDevices
	}
	if self.IsEncrypted() {
		if sourceInput.EncryptKeyId != nil && *sourceInput.EncryptKeyId != self.EncryptKeyId {
			return nil, errors.Wrap(httperrors.ErrConflict, "encrypt_key_id conflict with instance_backup's encrypt_key_id")
		}
		sourceInput.EncryptKeyId = &self.EncryptKeyId
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
	forceDelete := jsonutils.QueryBoolean(query, "force", false)
	return self.StartInstanceBackupDeleteTask(ctx, userCred, "", forceDelete)
}

func (self *SInstanceBackup) StartInstanceBackupDeleteTask(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, forceDelete bool) error {
	params := jsonutils.NewDict()
	if forceDelete {
		params.Set("force_delete", jsonutils.JSONTrue)
	}
	task, err := taskman.TaskManager.NewTask(ctx, "InstanceBackupDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	self.SetStatus(ctx, userCred, api.INSTANCE_BACKUP_STATUS_DELETING, "InstanceBackupDeleteTask")
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
	self.SetStatus(ctx, userCred, api.INSTANCE_BACKUP_STATUS_RECOVERY, "")
	params := jsonutils.NewDict()
	if serverName != "" {
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
	self.SetStatus(ctx, userCred, api.INSTANCE_BACKUP_STATUS_PACK, "")
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

func (manager *SInstanceBackupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.InstanceBackupManagerCreateFromPackageInput) (api.InstanceBackupManagerCreateFromPackageInput, error) {
	if input.PackageName == "" {
		return input, httperrors.NewMissingParameterError("miss package_name")
	}
	_, err := BackupStorageManager.FetchById(input.BackupStorageId)
	if err != nil {
		return input, httperrors.NewInputParameterError("unable to fetch backupStorage %s", input.BackupStorageId)
	}
	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
	}

	return input, nil
}

func (manager *SInstanceBackupManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
	packageName, _ := data[0].GetString("package_name")
	params := jsonutils.NewDict()
	params.Set("package_name", jsonutils.NewString(packageName))
	for i := range items {
		ib := items[i].(*SInstanceBackup)
		task, err := taskman.TaskManager.NewTask(ctx, "InstanceBackupUnpackTask", ib, userCred, params, "", "", nil)
		if err != nil {
			log.Errorf("InstanceBackupUnpackTask fail %s", err)
		} else {
			task.ScheduleRun(nil)
		}
	}
}

func (self *SInstanceBackup) PackMetadata(ctx context.Context, userCred mcclient.TokenCredential) (*api.InstanceBackupPackMetadata, error) {
	allMetadata, err := self.GetAllMetadata(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetAllMetadata")
	}

	metadata := &api.InstanceBackupPackMetadata{
		OsArch:         self.OsArch,
		ServerConfig:   self.ServerConfig,
		ServerMetadata: self.ServerMetadata,
		SecGroups:      self.SecGroups,
		KeypairId:      self.KeypairId,
		OsType:         self.OsType,
		InstanceType:   self.InstanceType,
		SizeMb:         self.SizeMb,

		EncryptKeyId: self.EncryptKeyId,
		Metadata:     allMetadata,
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

/*func (manager *SInstanceBackupManager) CreateInstanceBackupFromPackage(ctx context.Context, owner mcclient.TokenCredential, backupStorageId, name string) (*SInstanceBackup, error) {
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
}*/

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
	if len(metadata.Metadata) > 0 {
		// check class metadata
		allOwner := db.AllMetadataOwner(metadata.Metadata)
		err := db.RequireSameClass(ctx, allOwner, ib)
		if err != nil {
			return nil, errors.Wrap(err, "db.IsInSameClass")
		}
		meta := make(map[string]interface{})
		for k, v := range metadata.Metadata {
			meta[k] = v
		}
		ib.SetAllMetadata(ctx, meta, userCred)
	}
	if len(metadata.EncryptKeyId) > 0 {
		session := auth.GetSession(ctx, userCred, consts.GetRegion())
		_, err := identity_modules.Credentials.GetEncryptKey(session, metadata.EncryptKeyId)
		if err != nil {
			return nil, errors.Wrap(err, "GetEncryptKey")
		}
	}
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

		ib.EncryptKeyId = metadata.EncryptKeyId
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ib, nil
}

func (self *SInstanceBackup) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	if len(self.GuestId) > 0 {
		// use disk's ownerId instead of default ownerId
		guestObj, err := GuestManager.FetchById(self.GuestId)
		if err != nil {
			return errors.Wrap(err, "GuestManager.FetchById")
		}
		ownerId = guestObj.(*SGuest).GetOwnerId()
	}
	return self.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}
