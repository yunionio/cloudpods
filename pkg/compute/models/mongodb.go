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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=mongodb
// +onecloud:swagger-gen-model-plural=mongodbs
type SMongoDBManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDeletePreventableResourceBaseManager

	SCloudregionResourceBaseManager
	SZoneResourceBaseManager
	SManagedResourceBaseManager
	SVpcResourceBaseManager
}

var MongoDBManager *SMongoDBManager

func init() {
	MongoDBManager = &SMongoDBManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SMongoDB{},
			"mongodbs_tbl",
			"mongodb",
			"mongodbs",
		),
	}
	MongoDBManager.SetVirtualObject(MongoDBManager)
}

type SMongoDB struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SBillingResourceBase

	SCloudregionResourceBase
	SZoneResourceBase
	SDeletePreventableResourceBase

	// CPU数量
	// example: 1
	VcpuCount int `nullable:"false" default:"1" list:"user" create:"optional"`
	// 内存大小
	// example: 1024
	VmemSizeMb int `nullable:"false" list:"user" create:"required"`
	// 存储大小, 单位Mb
	// example: 10240
	DiskSizeMb int `nullable:"false" list:"user" create:"required"`
	// 端口
	// example: 3306
	Port int `nullable:"false" list:"user" create:"optional"`
	// 实例类型
	// example: ha
	Category string `nullable:"false" list:"user" create:"optional"`

	// 分片数量
	// example: 3
	ReplicationNum int `nullable:"false" default:"0" list:"user" create:"optional"`

	// 最大连接数
	MaxConnections int `nullable:"true" list:"user" create:"optional"`
	Iops           int `nullable:"true" list:"user" create:"optional"`

	// 实例IP地址
	IpAddr string `nullable:"false" list:"user"`

	// 引擎
	// example: MySQL
	Engine string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// 引擎版本
	// example: 5.7
	EngineVersion string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// 套餐名称
	// example: mysql.x4.large.2c
	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`

	// 维护时间
	MaintainTime string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// 虚拟私有网络Id
	// example: ed20d84e-3158-41b1-870c-1725e412e8b6
	VpcId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`

	// 所属网络ID
	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" json:"network_id"`

	// 连接地址
	NetworkAddress string `width:"256" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (manager *SMongoDBManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

// MongoDB实例列表
func (man *SMongoDBManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.MongoDBListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SDeletePreventableResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DeletePreventableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDeletePreventableResourceBaseManager.ListItemFilter")
	}
	q, err = man.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	q, err = man.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}

	if query.VcpuCount > 0 {
		q = q.Equals("vcpu_count", query.VcpuCount)
	}
	if query.VmemSizeMb > 0 {
		q = q.Equals("vmem_size_mb", query.VmemSizeMb)
	}
	if len(query.Category) > 0 {
		q = q.Equals("category", query.Category)
	}
	if len(query.Engine) > 0 {
		q = q.Equals("engine", query.Engine)
	}
	if len(query.EngineVersion) > 0 {
		q = q.Equals("engine_version", query.EngineVersion)
	}
	if len(query.InstanceType) > 0 {
		q = q.Equals("instance_type", query.InstanceType)
	}

	return q, nil
}

func (man *SMongoDBManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.MongoDBListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SMongoDBManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SMongoDBManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.MongoDBCreateInput) (*api.MongoDBCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implemented")
}

func (man *SMongoDBManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.MongoDBCreateInput) (api.MongoDBCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SMongoDBManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.MongoDBDetails {
	rows := make([]api.MongoDBDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	vpcIds := make([]string, len(rows))
	netIds := make([]string, len(rows))
	zoneIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.MongoDBDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regRows[i],
		}
		instance := objs[i].(*SMongoDB)
		vpcIds[i] = instance.VpcId
		netIds[i] = instance.NetworkId
		zoneIds[i] = instance.ZoneId
	}

	vpcs := make(map[string]SVpc)

	err := db.FetchStandaloneObjectsByIds(VpcManager, vpcIds, &vpcs)
	if err != nil {
		log.Errorf("db.FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	netMaps, err := db.FetchIdNameMap2(NetworkManager, netIds)
	if err != nil {
		return rows
	}

	zoneMaps, err := db.FetchIdNameMap2(ZoneManager, zoneIds)
	if err != nil {
		return rows
	}

	for i := range rows {
		if vpc, ok := vpcs[vpcIds[i]]; ok {
			rows[i].Vpc = vpc.Name
			rows[i].VpcExtId = vpc.ExternalId
		}
		rows[i].Network, _ = netMaps[netIds[i]]
		rows[i].Zone, _ = zoneMaps[zoneIds[i]]
	}

	return rows
}

func (self *SMongoDB) GetIMongoDB(ctx context.Context) (cloudprovider.ICloudMongoDB, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iregion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIRegion")
	}
	iMongoDB, err := iregion.GetICloudMongoDBById(self.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudMongoDBById(%s)", self.ExternalId)
	}
	return iMongoDB, nil
}

// 同步MongoDB实例状态
func (self *SMongoDB) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("MongoDB has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "MongoDBSyncstatusTask", "")
}

func (self *SMongoDB) SetAutoRenew(autoRenew bool) error {
	_, err := db.Update(self, func() error {
		self.AutoRenew = autoRenew
		return nil
	})
	return err
}

func (self *SMongoDB) SaveRenewInfo(
	ctx context.Context, userCred mcclient.TokenCredential,
	bc *billing.SBillingCycle, expireAt *time.Time, billingType string,
) error {
	_, err := db.Update(self, func() error {
		if billingType == "" {
			billingType = billing_api.BILLING_TYPE_PREPAID
		}
		if self.BillingType == "" {
			self.BillingType = billingType
		}
		if expireAt != nil && !expireAt.IsZero() {
			self.ExpiredAt = *expireAt
		} else {
			self.BillingCycle = bc.String()
			self.ExpiredAt = bc.EndAt(self.ExpiredAt)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, self.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SMongoDB) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("mongodb delete do nothing")
	return nil
}

func (self *SMongoDB) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SMongoDB) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SMongoDB) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "MongoDBDeleteTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(ctx, userCred, api.MONGO_DB_STATUS_DELETE_FAILED, err.Error())
		return err
	}
	return self.SetStatus(ctx, userCred, api.MONGO_DB_STATUS_DELETING, "")
}

func (self *SCloudregion) GetMongoDBs(managerId string) ([]SMongoDB, error) {
	q := MongoDBManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	dbs := []SMongoDB{}
	err := db.FetchModelObjects(MongoDBManager, q, &dbs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return dbs, nil
}

func (self *SCloudregion) SyncMongoDBs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	cloudMongoDBs []cloudprovider.ICloudMongoDB,
	xor bool,
) ([]SMongoDB, []cloudprovider.ICloudMongoDB, compare.SyncResult) {
	lockman.LockRawObject(ctx, MongoDBManager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, self.Id))
	defer lockman.ReleaseRawObject(ctx, MongoDBManager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, self.Id))

	localMongoDBs := []SMongoDB{}
	remoteMongoDBs := []cloudprovider.ICloudMongoDB{}
	result := compare.SyncResult{}

	dbInstances, err := self.GetMongoDBs(provider.Id)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	removed := make([]SMongoDB, 0)
	commondb := make([]SMongoDB, 0)
	commonext := make([]cloudprovider.ICloudMongoDB, 0)
	added := make([]cloudprovider.ICloudMongoDB, 0)
	err = compare.CompareSets(dbInstances, cloudMongoDBs, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudMongoDB(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		for i := 0; i < len(commondb); i++ {
			err := commondb[i].SyncWithCloudMongoDB(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			localMongoDBs = append(localMongoDBs, commondb[i])
			remoteMongoDBs = append(remoteMongoDBs, commonext[i])
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		instance, err := self.newFromCloudMongoDB(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		localMongoDBs = append(localMongoDBs, *instance)
		remoteMongoDBs = append(remoteMongoDBs, added[i])
		result.Add()
	}
	return localMongoDBs, remoteMongoDBs, result
}

func (self *SMongoDB) syncRemoveCloudMongoDB(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.RealDelete(ctx, userCred)
	if err != nil {
		return err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionSyncDelete,
	})
	return nil
}

func (self *SMongoDB) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("MongoDB is locked, cannot delete")
	}
	return self.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SMongoDB) SyncAllWithCloudMongoDB(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudMongoDB) error {
	err := self.SyncWithCloudMongoDB(ctx, userCred, ext)
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudMongoDB")
	}
	return nil
}

func (self *SMongoDB) SyncWithCloudMongoDB(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudMongoDB) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, ext.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}

		self.IpAddr = ext.GetIpAddr()
		self.VcpuCount = ext.GetVcpuCount()
		self.VmemSizeMb = ext.GetVmemSizeMb()
		self.DiskSizeMb = ext.GetDiskSizeMb()
		self.ReplicationNum = ext.GetReplicationNum()
		self.Engine = ext.GetEngine()
		self.EngineVersion = ext.GetEngineVersion()
		self.Category = ext.GetCategory()
		self.InstanceType = ext.GetInstanceType()
		self.MaintainTime = ext.GetMaintainTime()
		self.Status = ext.GetStatus()
		self.Port = ext.GetPort()
		if iops := ext.GetIops(); iops > 0 {
			self.Iops = iops
		}
		self.MaxConnections = ext.GetMaxConnections()
		self.NetworkAddress = ext.GetNetworkAddress()

		if vpcId := ext.GetVpcId(); len(vpcId) > 0 {
			vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", self.ManagerId)
			})
			if err != nil {
				log.Errorf("FetchVpcId(%s) error: %v", vpcId, err)
			} else {
				self.VpcId = vpc.GetId()
			}
		}

		if networkId := ext.GetNetworkId(); len(networkId) > 0 {
			network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				wire := WireManager.Query().SubQuery()
				vpc := VpcManager.Query().SubQuery()
				return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
					Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
					Filter(sqlchemy.Equals(vpc.Field("manager_id"), self.ManagerId))
			})
			if err == nil {
				self.NetworkId = network.GetId()
			}
		}

		if zoneId := ext.GetZoneId(); len(zoneId) > 0 {
			zone, err := self.GetZoneBySuffix(zoneId)
			if err != nil {
				log.Errorf("find zone %s error: %v", zoneId, err)
			} else {
				self.ZoneId = zone.Id
			}
		}

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionSyncUpdate,
		})
	}
	if account := self.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}
	if provider := self.GetCloudprovider(); provider != nil {
		SyncCloudProject(ctx, userCred, self, provider.GetOwnerId(), ext, provider)
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SCloudregion) newFromCloudMongoDB(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudMongoDB) (*SMongoDB, error) {
	ins := SMongoDB{}
	ins.SetModelManager(MongoDBManager, &ins)

	ins.ExternalId = ext.GetGlobalId()
	ins.CloudregionId = self.Id
	ins.ManagerId = provider.Id
	ins.Status = ext.GetStatus()
	ins.IpAddr = ext.GetIpAddr()
	ins.VcpuCount = ext.GetVcpuCount()
	ins.VmemSizeMb = ext.GetVmemSizeMb()
	ins.DiskSizeMb = ext.GetDiskSizeMb()
	ins.Engine = ext.GetEngine()
	ins.EngineVersion = ext.GetEngineVersion()
	ins.Category = ext.GetCategory()
	ins.InstanceType = ext.GetInstanceType()
	ins.MaintainTime = ext.GetMaintainTime()
	ins.Port = ext.GetPort()
	ins.ReplicationNum = ext.GetReplicationNum()
	ins.Iops = ext.GetIops()
	ins.MaxConnections = ext.GetMaxConnections()
	ins.NetworkAddress = ext.GetNetworkAddress()

	if zoneId := ext.GetZoneId(); len(zoneId) > 0 {
		zone, err := self.GetZoneBySuffix(zoneId)
		if err == nil {
			ins.ZoneId = zone.Id
		}
	}

	createdAt := ext.GetCreatedAt()
	if !createdAt.IsZero() {
		ins.CreatedAt = createdAt
	}

	ins.BillingType = ext.GetBillingType()
	if ins.BillingType == billing_api.BILLING_TYPE_PREPAID {
		expiredAt := ext.GetExpiredAt()
		if !expiredAt.IsZero() {
			ins.ExpiredAt = expiredAt
		}
		ins.AutoRenew = ext.IsAutoRenew()
	}

	if vpcId := ext.GetVpcId(); len(vpcId) > 0 {
		vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", provider.Id)
		})
		if err != nil {
			log.Errorf("FetchVpcId(%s) error: %v", vpcId, err)
		} else {
			ins.VpcId = vpc.GetId()
		}
	}

	if networkId := ext.GetNetworkId(); len(networkId) > 0 {
		network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			wire := WireManager.Query().SubQuery()
			vpc := VpcManager.Query().SubQuery()
			return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
				Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpc.Field("manager_id"), provider.Id))
		})
		if err != nil {
			return nil, errors.Wrapf(err, "ext.FetchNetworkId")
		}
		ins.NetworkId = network.GetId()
	}

	var err error
	err = func() error {
		lockman.LockRawObject(ctx, MongoDBManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, MongoDBManager.Keyword(), "name")

		ins.Name, err = db.GenerateName(ctx, MongoDBManager, provider.GetOwnerId(), ext.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}
		return MongoDBManager.TableSpec().Insert(ctx, &ins)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudMongoDB.Insert")
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &ins,
		Action: notifyclient.ActionSyncCreate,
	})

	syncVirtualResourceMetadata(ctx, userCred, &ins, ext, false)
	SyncCloudProject(ctx, userCred, &ins, provider.GetOwnerId(), ext, provider)
	db.OpsLog.LogEvent(&ins, db.ACT_CREATE, ins.GetShortDesc(ctx), userCred)

	return &ins, nil
}

type SMongoDBCountStat struct {
	TotalMongodbCount int
	TotalCpuCount     int
	TotalMemSizeMb    int
}

func (man *SMongoDBManager) TotalCount(
	ctx context.Context,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	providers []string, brands []string, cloudEnv string,
	policyResult rbacutils.SPolicyResult,
) (SMongoDBCountStat, error) {
	mgq := man.Query()

	mgq = scopeOwnerIdFilter(mgq, scope, ownerId)
	mgq = CloudProviderFilter(mgq, mgq.Field("manager_id"), providers, brands, cloudEnv)
	mgq = RangeObjectsFilter(mgq, rangeObjs, mgq.Field("cloudregion_id"), nil, mgq.Field("manager_id"), nil, nil)
	mgq = db.ObjectIdQueryWithPolicyResult(ctx, mgq, man, policyResult)

	sq := mgq.SubQuery()
	q := sq.Query(sqlchemy.COUNT("total_mongodb_count"),
		sqlchemy.SUM("total_cpu_count", sq.Field("vcpu_count")),
		sqlchemy.SUM("total_mem_size_mb", sq.Field("vmem_size_mb")))

	stat := SMongoDBCountStat{}
	row := q.Row()
	err := q.Row2Struct(row, &stat)
	return stat, err
}

func (self *SMongoDB) GetQuotaKeys() quotas.IQuotaKeys {
	region, _ := self.GetRegion()
	return fetchRegionalQuotaKeys(
		rbacscope.ScopeProject,
		self.GetOwnerId(),
		region,
		self.GetCloudprovider(),
	)
}

func (self *SMongoDB) GetUsages() []db.IUsage {
	if self.PendingDeleted || self.Deleted {
		return nil
	}
	usage := SRegionQuota{Rds: 1}
	keys := self.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (self *SMongoDB) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, err
	}
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetDriver")
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (manager *SMongoDBManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
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

	if keys.Contains("vpc") {
		q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"vpc"}))
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (self *SMongoDB) PerformPostpaidExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PostpaidExpireInput) (jsonutils.JSONObject, error) {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return nil, httperrors.NewBadRequestError("self billing type is %s", self.BillingType)
	}

	bc, err := ParseBillingCycleInput(&self.SBillingResourceBase, input)
	if err != nil {
		return nil, err
	}

	err = self.SaveRenewInfo(ctx, userCred, bc, nil, billing_api.BILLING_TYPE_POSTPAID)
	return nil, err
}

func (self *SMongoDB) PerformCancelExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := self.CancelExpireTime(ctx, userCred); err != nil {
		return nil, err
	}

	return nil, nil
}

func (self *SMongoDB) CancelExpireTime(ctx context.Context, userCred mcclient.TokenCredential) error {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return httperrors.NewBadRequestError("self billing type %s not support cancel expire", self.BillingType)
	}

	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set expired_at = NULL and billing_cycle = NULL where id = ?",
			MongoDBManager.TableSpec().Name(),
		), self.Id,
	)
	if err != nil {
		return errors.Wrap(err, "self cancel expire time")
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, "self cancel expire time", userCred)
	return nil
}

func (self *SMongoDB) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.MongoDBRemoteUpdateInput) (jsonutils.JSONObject, error) {
	err := self.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRemoteUpdateTask")
	}
	return nil, nil
}

func (self *SMongoDB) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewBool(replaceTags), "replace_tags")
	task, err := taskman.TaskManager.NewTask(ctx, "MongoDBRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_UPDATE_TAGS, "StartRemoteUpdateTask")
	return task.ScheduleRun(nil)
}

func (self *SMongoDB) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(self.ExternalId) == 0 || options.Options.KeepTagLocalization {
		return
	}
	if account := self.GetCloudaccount(); account != nil && account.ReadOnly {
		return
	}
	err := self.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}

// 获取备份列表
func (self *SMongoDB) GetDetailsBackups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*cloudprovider.SMongoDBBackups, error) {
	if self.Status != api.MONGO_DB_STATUS_RUNNING {
		return nil, httperrors.NewInvalidStatusError("invalid mongodb status %s for query backups", self.Status)
	}
	ret := &cloudprovider.SMongoDBBackups{}
	iMongoDB, err := self.GetIMongoDB(ctx)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetIMongoDB"))
	}
	ret.Data, err = iMongoDB.GetIBackups()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetIBackups"))
	}
	ret.Total = len(ret.Data)
	return ret, nil
}
