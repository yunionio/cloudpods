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
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SMountTargetManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SVpcResourceBaseManager
	SNetworkResourceBaseManager
	SAccessGroupResourceBaseManager
}

var MountTargetManager *SMountTargetManager

func init() {
	MountTargetManager = &SMountTargetManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SMountTarget{},
			"mount_targets_tbl",
			"mount_target",
			"mount_targets",
		),
	}
	MountTargetManager.SetVirtualObject(MountTargetManager)
}

type SMountTarget struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SVpcResourceBase
	SNetworkResourceBase
	SAccessGroupResourceBase

	NetworkType  string `width:"8" charset:"ascii" nullable:"false" create:"required" index:"true" list:"user" default:"vpc"`
	DomainName   string `charset:"utf8" nullable:"true" create:"optional" list:"user"`
	FileSystemId string `width:"36" charset:"ascii" nullable:"false" create:"required" index:"true" list:"user"`
}

func (manager *SMountTargetManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (self *SMountTarget) GetFileSystem() (*SFileSystem, error) {
	fs, err := FileSystemManager.FetchById(self.FileSystemId)
	if err != nil {
		return nil, errors.Wrapf(err, "FileSystemManager.FetchById(%s)", self.FileSystemId)
	}
	return fs.(*SFileSystem), nil
}

func (manager *SMountTargetManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.MountTargetCreateInput) (api.MountTargetCreateInput, error) {
	if len(input.FileSystemId) == 0 {
		return input, httperrors.NewMissingParameterError("file_system_id")
	}
	_fs, err := validators.ValidateModel(userCred, FileSystemManager, &input.FileSystemId)
	if err != nil {
		return input, err
	}
	fs := _fs.(*SFileSystem)
	if fs.MountTargetCountLimit > -1 {
		mts, err := fs.GetMountTargets()
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrapf(err, "fs.GetMountTargets"))
		}
		if len(mts) > fs.MountTargetCountLimit {
			return input, httperrors.NewOutOfLimitError("Mount target reached the upper limit")
		}
	}
	if len(input.NetworkType) == 0 {
		input.NetworkType = api.NETWORK_TYPE_VPC
	}
	if !utils.IsInStringArray(input.NetworkType, []string{api.NETWORK_TYPE_VPC, api.NETWORK_TYPE_CLASSIC}) {
		return input, httperrors.NewInputParameterError("invalid network type %s", input.NetworkType)
	}
	if input.NetworkType == api.NETWORK_TYPE_VPC {
		if len(input.NetworkId) == 0 {
			return input, httperrors.NewMissingParameterError("network_id")
		}
		_network, err := validators.ValidateModel(userCred, NetworkManager, &input.NetworkId)
		if err != nil {
			return input, err
		}
		network := _network.(*SNetwork)
		vpc, _ := network.GetVpc()
		if vpc == nil {
			return input, httperrors.NewGeneralError(fmt.Errorf("failed to found vpc for network %s", input.NetworkId))
		}
		if vpc.ManagerId != fs.ManagerId {
			return input, httperrors.NewConflictError("network and filesystem do not belong to the same account")
		}
		if vpc.CloudregionId != fs.CloudregionId {
			return input, httperrors.NewConflictError("network and filesystem are not in the same region")
		}
		input.VpcId = vpc.Id
	}
	if len(input.AccessGroupId) == 0 {
		return input, httperrors.NewMissingParameterError("access_group_id")
	}
	_, err = validators.ValidateModel(userCred, AccessGroupManager, &input.AccessGroupId)
	if err != nil {
		return input, err
	}
	input.StatusStandaloneResourceCreateInput, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SMountTarget) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if self.NetworkType == api.NETWORK_TYPE_CLASSIC {
		db.Update(self, func() error {
			self.VpcId = ""
			self.NetworkId = ""
			return nil
		})
	}
	self.StartCreateTask(ctx, userCred, "")
}

func (self *SMountTarget) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "MountTargetCreateTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.MOUNT_TARGET_STATUS_CREATE_FAILED, err.Error())
		return nil
	}
	self.SetStatus(userCred, api.MOUNT_TARGET_STATUS_CREATING, "")
	return nil
}

func (manager *SMountTargetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.MountTargetListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SAccessGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.AccessGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SAccessGroupResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SVpcResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SNetworkResourceBaseManager.ListItemFilter")
	}
	if len(query.FileSystemId) > 0 {
		_, err := validators.ValidateModel(userCred, FileSystemManager, &query.FileSystemId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("file_system_id", query.FileSystemId)
	}
	return q, nil
}

func (manager *SMountTargetManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.MountTargetListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SAccessGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.AccessGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SAccessGroupResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SVpcResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SMountTargetManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SAccessGroupResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SNetworkResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SMountTarget) GetOwnerId() mcclient.IIdentityProvider {
	fs, err := self.GetFileSystem()
	if err != nil {
		return &db.SOwnerId{}
	}
	return &db.SOwnerId{DomainId: fs.DomainId}
}

func (manager *SMountTargetManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		sq := FileSystemManager.Query("id")
		if scope == rbacutils.ScopeDomain && len(userCred.GetProjectDomainId()) > 0 {
			sq = sq.Equals("domain_id", userCred.GetProjectDomainId())
			return q.In("file_system_id", sq)
		}
	}
	return q
}

func (manager *SMountTargetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.MountTargetDetails {
	rows := make([]api.MountTargetDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	netRows := manager.SNetworkResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	acRows := manager.SAccessGroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	fsIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.MountTargetDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			VpcResourceInfo:                 vpcRows[i],
			NetworkResourceInfo:             netRows[i],
			AccessGroupResourceInfo:         acRows[i],
		}
		mount := objs[i].(*SMountTarget)
		fsIds[i] = mount.FileSystemId
	}

	fsMaps, err := db.FetchIdNameMap2(FileSystemManager, fsIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		rows[i].FileSystem, _ = fsMaps[fsIds[i]]
	}

	return rows
}

func (manager *SMountTargetManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrapf(err, "SNetworkResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SAccessGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrapf(err, "SAccessGroupResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (self *SMountTarget) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	fs, err := self.GetFileSystem()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrapf(err, "GetFileSystem"))
	}
	region, err := fs.GetRegion()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrapf(err, "GetRegion"))
	}
	if utils.IsInStringArray(region.Provider, []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
		return httperrors.NewNotSupportedError("not allow to delete")
	}
	return self.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SMountTarget) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SMountTarget) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

// 删除挂载点
func (self *SMountTarget) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query api.ServerDeleteInput, input api.NatgatewayDeleteInput) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SMountTarget) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "MountTargetDeleteTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.MOUNT_TARGET_STATUS_DELETE_FAILED, err.Error())
		return err
	}
	return self.SetStatus(userCred, api.MOUNT_TARGET_STATUS_DELETING, "")
}

func (self *SFileSystem) GetMountTargets() ([]SMountTarget, error) {
	mounts := []SMountTarget{}
	q := MountTargetManager.Query().Equals("file_system_id", self.Id)
	err := db.FetchModelObjects(MountTargetManager, q, &mounts)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return mounts, nil
}

func (self *SFileSystem) SyncMountTargets(ctx context.Context, userCred mcclient.TokenCredential, extMounts []cloudprovider.ICloudMountTarget) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, MountTargetManager.KeywordPlural())
	lockman.ReleaseRawObject(ctx, self.Id, MountTargetManager.KeywordPlural())

	result := compare.SyncResult{}

	dbMounts, err := self.GetMountTargets()
	if err != nil {
		result.Error(errors.Wrapf(err, "self.GetMountTargets"))
		return result
	}

	removed := make([]SMountTarget, 0)
	commondb := make([]SMountTarget, 0)
	commonext := make([]cloudprovider.ICloudMountTarget, 0)
	added := make([]cloudprovider.ICloudMountTarget, 0)
	err = compare.CompareSets(dbMounts, extMounts, &removed, &commondb, &commonext, &added)
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
		err = commondb[i].SyncWithMountTarget(ctx, userCred, self.ManagerId, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		err := self.newFromCloudMountTarget(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SMountTarget) SyncWithMountTarget(ctx context.Context, userCred mcclient.TokenCredential, managerId string, m cloudprovider.ICloudMountTarget) error {
	_, err := db.Update(self, func() error {
		self.Status = m.GetStatus()
		self.Name = m.GetName()
		self.DomainName = m.GetDomainName()
		self.ExternalId = m.GetGlobalId()
		if groupId := m.GetAccessGroupId(); len(groupId) > 0 {
			_cache, _ := db.FetchByExternalIdAndManagerId(AccessGroupCacheManager, groupId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", managerId)
			})
			if _cache != nil {
				cache := _cache.(*SAccessGroupCache)
				self.AccessGroupId = cache.AccessGroupId
			}
		}
		return nil
	})
	return errors.Wrapf(err, "db.Update")
}

func (self *SFileSystem) newFromCloudMountTarget(ctx context.Context, userCred mcclient.TokenCredential, m cloudprovider.ICloudMountTarget) error {
	mount := &SMountTarget{}
	mount.SetModelManager(MountTargetManager, mount)
	mount.FileSystemId = self.Id
	mount.Name = m.GetName()
	mount.Status = m.GetStatus()
	mount.ExternalId = m.GetGlobalId()
	mount.DomainName = m.GetDomainName()
	mount.NetworkType = m.GetNetworkType()
	if mount.NetworkType == api.NETWORK_TYPE_VPC {
		if vpcId := m.GetVpcId(); len(vpcId) > 0 {
			vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", self.ManagerId)
			})
			if err != nil {
				log.Errorf("failed to found vpc for mount point %s by externalId: %s", mount.Name, vpcId)
			} else {
				mount.VpcId = vpc.GetId()
			}
		}
		if networkId := m.GetNetworkId(); len(networkId) > 0 {
			network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				wire := WireManager.Query().SubQuery()
				vpc := VpcManager.Query().SubQuery()
				return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
					Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
					Filter(sqlchemy.Equals(vpc.Field("manager_id"), self.ManagerId))
			})
			if err != nil {
				log.Errorf("failed to found network for mount point %s by externalId: %s", mount.Name, networkId)
			} else {
				mount.NetworkId = network.GetId()
			}
		}
	}

	return MountTargetManager.TableSpec().Insert(ctx, mount)
}

func (self *SMountTarget) GetNetwork() (*SNetwork, error) {
	network, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		return nil, errors.Wrapf(err, "NetworkManager.FetchById(%s)", self.NetworkId)
	}
	return network.(*SNetwork), nil
}

func (self *SMountTarget) GetVpc() (*SVpc, error) {
	vpc, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "VpcManager.FetchById(%s)", self.VpcId)
	}
	return vpc.(*SVpc), nil
}

// 同步挂载点状态
func (self *SMountTarget) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.MountTargetSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SMountTarget) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "MountTargetSyncstatusTask", parentTaskId)
}
