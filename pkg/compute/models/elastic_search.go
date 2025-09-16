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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SElasticSearchManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDeletePreventableResourceBaseManager

	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
	SVpcResourceBaseManager
}

var ElasticSearchManager *SElasticSearchManager

func init() {
	ElasticSearchManager = &SElasticSearchManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SElasticSearch{},
			"elastic_searchs_tbl",
			"elastic_search",
			"elastic_searchs",
		),
	}
	ElasticSearchManager.SetVirtualObject(ElasticSearchManager)
	notifyclient.AddNotifyDBHookResources(ElasticSearchManager.KeywordPlural())
}

type SElasticSearch struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SBillingResourceBase

	SCloudregionResourceBase
	SDeletePreventableResourceBase

	// 版本
	Version string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`

	// 套餐名称
	// example: elasticsearch.sn2ne.xlarge
	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`

	// CPU数量
	// example: 1
	VcpuCount int `nullable:"false" default:"1" list:"user" create:"optional"`
	// 内存大小
	// example: 1024
	VmemSizeGb int `nullable:"false" list:"user" create:"optional"`

	// 存储类型
	// example: local_ssd
	StorageType string `nullable:"false" list:"user" create:"required"`
	// 存储大小
	// example: 1024
	DiskSizeGb int `nullable:"false" list:"user" create:"required"`
	// 实例类型
	// example: ha
	Category string `nullable:"false" list:"user" create:"optional"`

	VpcId     string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"vpc_id"`
	NetworkId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"network_id"`

	// 可用区Id
	ZoneId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"zone_id"`
	// 是否是多可用区部署
	IsMultiAz bool `nullable:"false" default:"false" list:"user" update:"user" create:"optional"`
}

func (manager *SElasticSearchManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

// ElasticSearch实例列表
func (man *SElasticSearchManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticSearchListInput,
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

	return q, nil
}

func (man *SElasticSearchManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticSearchListInput,
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

func (man *SElasticSearchManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (man *SElasticSearchManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ElasticSearchCreateInput) (api.ElasticSearchCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SElasticSearchManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticSearchDetails {
	rows := make([]api.ElasticSearchDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	vpcIds := make([]string, len(objs))
	netIds := make([]string, len(objs))
	zoneIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ElasticSearchDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regRows[i],
		}
		es := objs[i].(*SElasticSearch)
		vpcIds[i] = es.VpcId
		netIds[i] = es.NetworkId
		zoneIds[i] = es.ZoneId
	}
	vpcMaps, err := db.FetchIdNameMap2(VpcManager, vpcIds)
	if err != nil {
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
		rows[i].Vpc, _ = vpcMaps[vpcIds[i]]
		rows[i].Network, _ = netMaps[netIds[i]]
		rows[i].Zone, _ = zoneMaps[zoneIds[i]]
	}
	return rows
}

func (self *SCloudregion) GetElasticSearchs(managerId string) ([]SElasticSearch, error) {
	q := ElasticSearchManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SElasticSearch{}
	err := db.FetchModelObjects(ElasticSearchManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncElasticSearchs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudElasticSearch,
	xor bool,
) compare.SyncResult {
	// 加锁防止重入
	lockman.LockRawObject(ctx, ElasticSearchManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))
	defer lockman.ReleaseRawObject(ctx, ElasticSearchManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))

	result := compare.SyncResult{}

	dbEss, err := self.GetElasticSearchs(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SElasticSearch, 0)
	commondb := make([]SElasticSearch, 0)
	commonext := make([]cloudprovider.ICloudElasticSearch, 0)
	added := make([]cloudprovider.ICloudElasticSearch, 0)
	// 本地和云上资源列表进行比对
	err = compare.CompareSets(dbEss, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	// 删除云上没有的资源
	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticSearch(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		// 和云上资源属性进行同步
		for i := 0; i < len(commondb); i++ {
			err := commondb[i].SyncWithCloudElasticSearch(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	// 创建本地没有的云上资源
	for i := 0; i < len(added); i++ {
		_, err := self.newFromCloudElasticSearch(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

type SEsCountStat struct {
	TotalEsCount   int
	TotalCpuCount  int
	TotalMemSizeGb int
}

func (man *SElasticSearchManager) TotalCount(
	ctx context.Context,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	providers []string, brands []string, cloudEnv string,
	policyResult rbacutils.SPolicyResult,
) (SEsCountStat, error) {
	esq := man.Query()
	esq = scopeOwnerIdFilter(esq, scope, ownerId)
	esq = CloudProviderFilter(esq, esq.Field("manager_id"), providers, brands, cloudEnv)
	esq = RangeObjectsFilter(esq, rangeObjs, esq.Field("cloudregion_id"), nil, esq.Field("manager_id"), nil, nil)
	esq = db.ObjectIdQueryWithPolicyResult(ctx, esq, man, policyResult)

	sq := esq.SubQuery()
	q := sq.Query(sqlchemy.COUNT("total_es_count"),
		sqlchemy.SUM("total_cpu_count", sq.Field("vcpu_count")),
		sqlchemy.SUM("total_mem_size_gb", sq.Field("vmem_size_gb")))

	stat := SEsCountStat{}
	row := q.Row()
	err := q.Row2Struct(row, &stat)
	return stat, err
}

// 判断资源是否可以删除
func (self *SElasticSearch) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("ElasticSearch is locked, cannot delete")
	}
	return self.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SElasticSearch) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SElasticSearch) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SElasticSearch) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SElasticSearch) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticSearchDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	self.SetStatus(ctx, userCred, api.ELASTIC_SEARCH_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticSearch) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetDriver")
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (self *SElasticSearch) GetIElasticSearch(ctx context.Context) (cloudprovider.ICloudElasticSearch, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty externalId")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetIRegion")
	}
	return iRegion.GetIElasticSearchById(self.ExternalId)
}

func (self *SElasticSearch) syncRemoveCloudElasticSearch(ctx context.Context, userCred mcclient.TokenCredential) error {
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

// 同步资源属性
func (self *SElasticSearch) SyncWithCloudElasticSearch(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudElasticSearch) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, ext.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}

		self.Status = ext.GetStatus()
		self.Version = ext.GetVersion()
		self.StorageType = ext.GetStorageType()
		self.DiskSizeGb = ext.GetDiskSizeGb()
		self.Category = ext.GetCategory()
		self.InstanceType = ext.GetInstanceType()
		self.VcpuCount = ext.GetVcpuCount()
		self.VmemSizeGb = ext.GetVmemSizeGb()
		self.IsMultiAz = ext.IsMultiAz()

		self.BillingType = ext.GetBillingType()
		if self.BillingType == billing_api.BILLING_TYPE_PREPAID {
			if expiredAt := ext.GetExpiredAt(); !expiredAt.IsZero() {
				self.ExpiredAt = expiredAt
			}
			self.AutoRenew = ext.IsAutoRenew()
		}

		if networkId := ext.GetNetworkId(); len(networkId) > 0 && len(self.NetworkId) == 0 {
			_network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				wire := WireManager.Query().SubQuery()
				vpc := VpcManager.Query().SubQuery()
				return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
					Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
					Filter(sqlchemy.Equals(vpc.Field("manager_id"), self.ManagerId))
			})
			if err != nil {
				log.Errorf("failed to found network for elastic search %s by externalId: %s", self.Name, networkId)
			} else {
				network := _network.(*SNetwork)
				self.NetworkId = network.Id
				vpc, _ := network.GetVpc()
				self.VpcId = vpc.Id
				if zone, _ := network.GetZone(); zone != nil {
					self.ZoneId = zone.Id
				}
			}
		}
		if vpcId := ext.GetVpcId(); len(vpcId) > 0 && len(self.VpcId) == 0 {
			vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", self.ManagerId)
			})
			if err == nil {
				self.VpcId = vpc.GetId()
			}
		}

		if len(self.ZoneId) == 0 {
			zoneId := ext.GetZoneId()
			if len(zoneId) > 0 {
				region, err := self.GetRegion()
				if err != nil {
					return errors.Wrapf(err, "GetRegion")
				}
				zones, err := region.GetZones()
				if err != nil {
					return errors.Wrapf(err, "GetZone")
				}
				for _, zone := range zones {
					if strings.HasSuffix(zone.ExternalId, zoneId) {
						self.ZoneId = zone.Id
						break
					}
				}
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

func (self *SCloudregion) newFromCloudElasticSearch(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudElasticSearch) (*SElasticSearch, error) {
	es := SElasticSearch{}
	es.SetModelManager(ElasticSearchManager, &es)

	es.ExternalId = ext.GetGlobalId()
	es.CloudregionId = self.Id
	es.ManagerId = provider.Id
	es.IsEmulated = ext.IsEmulated()
	es.Status = ext.GetStatus()
	es.Version = ext.GetVersion()
	es.StorageType = ext.GetStorageType()
	es.DiskSizeGb = ext.GetDiskSizeGb()
	es.Category = ext.GetCategory()
	es.InstanceType = ext.GetInstanceType()
	es.VcpuCount = ext.GetVcpuCount()
	es.VmemSizeGb = ext.GetVmemSizeGb()
	es.IsMultiAz = ext.IsMultiAz()

	if networkId := ext.GetNetworkId(); len(networkId) > 0 {
		_network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			wire := WireManager.Query().SubQuery()
			vpc := VpcManager.Query().SubQuery()
			return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
				Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpc.Field("manager_id"), provider.Id))
		})
		if err != nil {
			log.Errorf("failed to found network for elastic search %s by externalId: %s", es.Name, networkId)
		} else {
			network := _network.(*SNetwork)
			es.NetworkId = network.Id
			vpc, _ := network.GetVpc()
			es.VpcId = vpc.Id
			if zone, _ := network.GetZone(); zone != nil {
				es.ZoneId = zone.Id
			}
		}
	}
	if len(es.VpcId) == 0 {
		if vpcId := ext.GetVpcId(); len(vpcId) > 0 {
			vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", provider.Id)
			})
			if err == nil {
				es.VpcId = vpc.GetId()
			}
		}
	}

	if len(es.ZoneId) == 0 {
		zoneId := ext.GetZoneId()
		if len(zoneId) > 0 {
			zones, _ := self.GetZones()
			for _, zone := range zones {
				if strings.HasSuffix(zone.ExternalId, zoneId) {
					es.ZoneId = zone.Id
					break
				}
			}
		}
	}

	if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
		es.CreatedAt = createdAt
	}

	es.BillingType = ext.GetBillingType()
	if es.BillingType == billing_api.BILLING_TYPE_PREPAID {
		if expired := ext.GetExpiredAt(); !expired.IsZero() {
			es.ExpiredAt = expired
		}
		es.AutoRenew = ext.IsAutoRenew()
	}

	var err error
	err = func() error {
		// 这里加锁是为了防止名称重复
		lockman.LockRawObject(ctx, ElasticSearchManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, ElasticSearchManager.Keyword(), "name")

		es.Name, err = db.GenerateName(ctx, ElasticSearchManager, provider.GetOwnerId(), ext.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}
		return ElasticSearchManager.TableSpec().Insert(ctx, &es)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticSearch.Insert")
	}

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &es,
		Action: notifyclient.ActionSyncCreate,
	})
	// 同步标签
	syncVirtualResourceMetadata(ctx, userCred, &es, ext, false)
	// 同步项目归属
	SyncCloudProject(ctx, userCred, &es, provider.GetOwnerId(), ext, provider)

	db.OpsLog.LogEvent(&es, db.ACT_CREATE, es.GetShortDesc(ctx), userCred)

	return &es, nil
}

func (manager *SElasticSearchManager) ListItemExportKeys(ctx context.Context,
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

	return q, nil
}

// 同步ElasticSearch实例状态
func (self *SElasticSearch) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("ElasticSearch has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "ElasticSearchSyncstatusTask", "")
}

func (self *SElasticSearch) GetDetailsAccessInfo(ctx context.Context, userCred mcclient.TokenCredential, input api.ElasticSearchAccessInfoInput) (*cloudprovider.ElasticSearchAccessInfo, error) {
	iEs, err := self.GetIElasticSearch(ctx)
	if err != nil {
		return nil, err
	}
	return iEs.GetAccessInfo()
}

func (es *SElasticSearch) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	es.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
}

func (self *SElasticSearch) StartSElasticSearchSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "ElasticSearchSyncstatusTask", parentTaskId)
}

func (self *SElasticSearch) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	if replaceTags {
		data.Add(jsonutils.JSONTrue, "replace_tags")
	}
	if task, err := taskman.TaskManager.NewTask(ctx, "ElasticSearchRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		return errors.Wrap(err, "Start ElasticSearchRemoteUpdateTask")
	} else {
		self.SetStatus(ctx, userCred, api.ELASTIC_SEARCH_UPDATE_TAGS, "StartRemoteUpdateTask")
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SElasticSearch) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
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
