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

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDnsZoneCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDnsZoneResourceBaseManager
}

var DnsZoneCacheManager *SDnsZoneCacheManager

func init() {
	DnsZoneCacheManager = &SDnsZoneCacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SDnsZoneCache{},
			"dns_zone_caches_tbl",
			"dns_zonecache",
			"dns_zonecaches",
		),
	}
	DnsZoneCacheManager.SetVirtualObject(DnsZoneCacheManager)

}

type SDnsZoneCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SDnsZoneResourceBase

	// 归属云账号ID
	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	ProductType    string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SDnsZoneCacheManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DnsZoneCacheCreateInput) (api.DnsZoneCacheCreateInput, error) {
	return input, httperrors.NewNotSupportedError("Not support")
}

func (manager *SDnsZoneCacheManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (manager *SDnsZoneCacheManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	dnsZoneId, _ := data.GetString("dns_zone_id")
	if len(dnsZoneId) > 0 {
		dnsZone, err := db.FetchById(DnsZoneManager, dnsZoneId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(DnsZoneManager, %s)", dnsZoneId)
		}
		return dnsZone.(*SDnsZone).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (self *SDnsZoneCache) GetOwnerId() mcclient.IIdentityProvider {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil
	}
	return dnsZone.GetOwnerId()
}

func (manager *SDnsZoneCacheManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	sq := DnsZoneManager.Query("id")
	sq = db.SharableManagerFilterByOwner(DnsZoneManager, sq, userCred, scope)
	return q.In("dns_zone_id", sq.SubQuery())
}

func (manager *SDnsZoneCacheManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, manager)
}

func (manager *SDnsZoneCacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsZoneCacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SDnsZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DnsZoneFilterListBase)
	if err != nil {
		return nil, err
	}

	if len(query.CloudaccountId) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(userCred, query.CloudaccountId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudaccount", query.CloudaccountId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("cloudaccount_id", account.GetId())
	}

	return q, nil
}

func (manager *SDnsZoneCacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DnsZoneCacheDetails {
	rows := make([]api.DnsZoneCacheDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	accountIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.DnsZoneCacheDetails{
			StatusStandaloneResourceDetails: stdRows[i],
		}
		cache := objs[i].(*SDnsZoneCache)
		accountIds[i] = cache.CloudaccountId
	}
	accounts := make(map[string]SCloudaccount)
	err := db.FetchStandaloneObjectsByIds(CloudaccountManager, accountIds, &accounts)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds (%s) fail %s",
			CloudaccountManager.KeywordPlural(), err)
		return rows
	}

	for i := range rows {
		if account, ok := accounts[accountIds[i]]; ok {
			rows[i].Account = account.Name
			rows[i].Brand = account.Brand
			rows[i].Provider = account.Provider
		}
	}

	return rows
}

func (self *SDnsZoneCache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDnsZoneCacheDeleteTask(ctx, userCred, "")
}

func (self *SDnsZoneCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SDnsZoneCache) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return errors.Wrapf(err, "GetVpcs")
	}
	if dnsZone, _ := self.GetDnsZone(); dnsZone != nil {
		for _, vpc := range vpcs {
			dnsZone.RemoveVpc(ctx, vpc.Id)
		}
	}
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SDnsZoneCache) GetDnsZone() (*SDnsZone, error) {
	dnsZone, err := DnsZoneManager.FetchById(self.DnsZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "DnsZoneManager.FetchById(%s)", self.DnsZoneId)
	}
	return dnsZone.(*SDnsZone), nil
}

func (self *SDnsZoneCache) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrapf(err, "GetDnsZone for %s", self.Name)
		}
		return self.RealDelete(ctx, userCred)
	}
	if cloudprovider.TDnsZoneType(dnsZone.ZoneType) == cloudprovider.PublicZone {
		return self.RealDelete(ctx, userCred)
	}

	err = dnsZone.RealDelete(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "dnsZone.RealDelete for %s(%s)", dnsZone.Name, dnsZone.Id)
	}

	return self.RealDelete(ctx, userCred)
}

func (self *SDnsZoneCache) GetVpcs() ([]SVpc, error) {
	providerQ := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.CloudaccountId)
	vpcQ := DnsZoneVpcManager.Query("vpc_id").Equals("dns_zone_id", self.DnsZoneId)
	q := VpcManager.Query().In("manager_id", providerQ.SubQuery()).In("id", vpcQ.SubQuery())
	vpcs := []SVpc{}
	err := db.FetchModelObjects(VpcManager, q, &vpcs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return vpcs, nil
}

func (self *SDnsZoneCache) SyncWithCloudDnsZone(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsZone) error {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return errors.Wrapf(err, "GetDnsZone")
	}
	_, err = db.Update(self, func() error {
		self.ExternalId = ext.GetGlobalId()
		self.Status = ext.GetStatus()
		self.Name = ext.GetName()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	vpcs, err := self.GetVpcs()
	if err != nil {
		return errors.Wrapf(err, "GetVpcs")
	}
	if dnsZone.ZoneType == string(cloudprovider.PrivateZone) {
		externalIds, err := ext.GetICloudVpcIds()
		if err != nil {
			return errors.Wrapf(err, "GetICloudVpcIds for %s", dnsZone.Name)
		}
		vpcIds := []string{}
		for _, externalId := range externalIds {
			vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, externalId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				sq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.CloudaccountId)
				return q.In("manager_id", sq.SubQuery())
			})
			if err != nil {
				return errors.Wrapf(err, "vpc.FetchByExternalIdAndManagerId(%s)", externalId)
			}
			vpcIds = append(vpcIds, vpc.GetId())
		}
		localVpcs := set.New(set.ThreadSafe)
		for i := range vpcs {
			localVpcs.Add(vpcs[i].Id)
		}
		remoteVpcs := set.New(set.ThreadSafe)
		for _, vpcId := range vpcIds {
			remoteVpcs.Add(vpcId)
		}
		for _, del := range set.Difference(localVpcs, remoteVpcs).List() {
			dnsZone.RemoveVpc(ctx, del.(string))
		}
		for _, add := range set.Difference(remoteVpcs, localVpcs).List() {
			dnsZone.AddVpc(ctx, add.(string))
		}
	}
	return nil
}

func (self *SDnsZoneCache) SyncVpcForCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	iDnsZone, err := self.GetICloudDnsZone()
	if err != nil {
		return errors.Wrapf(err, "GetICloudDnsZone")
	}
	vpcs, err := self.GetVpcs()
	if err != nil {
		return errors.Wrapf(err, "GetVpcs")
	}

	externalIds, err := iDnsZone.GetICloudVpcIds()
	if err != nil {
		return errors.Wrapf(err, "GetICloudVpcIds for %s", self.Name)
	}
	vpcIds := []string{}
	for _, externalId := range externalIds {
		vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, externalId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.CloudaccountId)
			return q.In("manager_id", sq.SubQuery())
		})
		if err != nil {
			return errors.Wrapf(err, "vpc.FetchByExternalIdAndManagerId(%s)", externalId)
		}
		vpcIds = append(vpcIds, vpc.GetId())
	}
	localVpcs := set.New(set.ThreadSafe)
	for i := range vpcs {
		localVpcs.Add(vpcs[i].Id)
	}
	remoteVpcs := set.New(set.ThreadSafe)
	for _, vpcId := range vpcIds {
		remoteVpcs.Add(vpcId)
	}
	for _, del := range set.Difference(remoteVpcs, localVpcs).List() {
		_vpc, err := VpcManager.FetchById(del.(string))
		if err != nil {
			return errors.Wrapf(err, "VpcManager.FetchById(%s)", del.(string))
		}
		vpc := _vpc.(*SVpc)
		iVpc, err := vpc.GetIVpc()
		if err != nil {
			return errors.Wrapf(err, "GetIVpc")
		}
		opts := cloudprovider.SPrivateZoneVpc{
			Id:       iVpc.GetGlobalId(),
			RegionId: iVpc.GetRegion().GetId(),
		}
		err = iDnsZone.RemoveVpc(&opts)
		if err != nil {
			return errors.Wrapf(err, "iDnsZone.RemoveVpc")
		}
	}
	for _, add := range set.Difference(localVpcs, remoteVpcs).List() {
		_vpc, err := VpcManager.FetchById(add.(string))
		if err != nil {
			return errors.Wrapf(err, "VpcManager.FetchById(%s)", add.(string))
		}
		vpc := _vpc.(*SVpc)
		iVpc, err := vpc.GetIVpc()
		if err != nil {
			return errors.Wrapf(err, "GetIVpc")
		}
		opts := cloudprovider.SPrivateZoneVpc{
			Id:       iVpc.GetGlobalId(),
			RegionId: iVpc.GetRegion().GetId(),
		}
		err = iDnsZone.AddVpc(&opts)
		if err != nil {
			return errors.Wrapf(err, "iDnsZone.AddVpc")
		}
	}
	return nil
}

func (self *SDnsZoneCache) GetCloudaccount() (*SCloudaccount, error) {
	account, err := CloudaccountManager.FetchById(self.CloudaccountId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudaccountManager.FetchById(%s)", self.CloudaccountId)
	}
	return account.(*SCloudaccount), nil
}

func (self *SDnsZoneCache) GetProvider() (cloudprovider.ICloudProvider, error) {
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudaccount")
	}
	return account.GetProvider()
}

func (self *SDnsZoneCache) GetICloudDnsZone() (cloudprovider.ICloudDnsZone, error) {
	provider, err := self.GetProvider()
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	return provider.GetICloudDnsZoneById(self.ExternalId)
}

func (self *SDnsZoneCache) StartDnsZoneCacheDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return errors.Wrapf(err, "GetDnsZone")
	}
	dnsZone.SetStatus(userCred, api.DNS_ZONE_STATUS_UNCACHING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneCacheDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_CACHE_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDnsZoneCache) StartDnsZoneCacheCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return errors.Wrapf(err, "GetDnsZone")
	}
	dnsZone.SetStatus(userCred, api.DNS_ZONE_STATUS_CACHING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneCacheCreateTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_CACHE_STATUS_CREATING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDnsZoneCache) GetRecordSetsByDnsType(dnsTypes []cloudprovider.TDnsType) ([]SDnsRecordSet, error) {
	records := []SDnsRecordSet{}
	q := DnsRecordSetManager.Query().Equals("dns_zone_id", self.DnsZoneId).In("dns_type", dnsTypes)
	err := db.FetchModelObjects(DnsRecordSetManager, q, &records)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return records, nil
}

func (self *SDnsZoneCache) GetRecordSets() ([]cloudprovider.DnsRecordSet, error) {
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudaccount")
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		return nil, errors.Wrapf(err, "GetProviderFactory")
	}
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDnsZone")
	}

	_dnsTypes := factory.GetSupportedDnsTypes()
	dnsTypes, ok := _dnsTypes[cloudprovider.TDnsZoneType(dnsZone.ZoneType)]
	if !ok {
		return nil, fmt.Errorf("invalid zone type %s for %s(%s)", dnsZone.ZoneType, account.Name, account.Provider)
	}
	records, err := self.GetRecordSetsByDnsType(dnsTypes)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRecordSetsByDnsType")
	}
	ttlrange := factory.GetTTLRange(cloudprovider.TDnsZoneType(dnsZone.ZoneType), cloudprovider.TDnsProductType(self.ProductType))
	ret := []cloudprovider.DnsRecordSet{}
	for i := range records {
		record := cloudprovider.DnsRecordSet{
			Id:         records[i].Id,
			DnsName:    records[i].Name,
			DnsValue:   records[i].DnsValue,
			DnsType:    cloudprovider.TDnsType(records[i].DnsType),
			Enabled:    records[i].Enabled.Bool(),
			Status:     records[i].Status,
			Ttl:        ttlrange.GetSuppportedTTL(records[i].TTL),
			MxPriority: records[i].MxPriority,
		}

		record.PolicyType, record.PolicyValue, record.PolicyOptions, err = records[i].GetDefaultDnsTrafficPolicy(account.Provider)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDefaultDnsTrafficPolicy(%s)", account.Provider)
		}
		ret = append(ret, record)
	}

	return ret, nil
}

func (self *SDnsZoneCache) SyncRecordSets(ctx context.Context, userCred mcclient.TokenCredential) error {
	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccount")
	}

	iDnsZone, err := self.GetICloudDnsZone()
	if err != nil {
		return errors.Wrapf(err, "GetICloudDnsZone")
	}
	iRecordSets, err := iDnsZone.GetIDnsRecordSets()
	if err != nil {
		return errors.Wrapf(err, "GetIDnsRecordSets")
	}
	dbRecordSets, err := self.GetRecordSets()
	if err != nil {
		return errors.Wrapf(err, "GetRecordSets")
	}

	var skipSoaAndNs = func(records []cloudprovider.DnsRecordSet) []cloudprovider.DnsRecordSet {
		ret := []cloudprovider.DnsRecordSet{}
		for i := range records {
			if isIn, _ := utils.InArray(records[i].DnsType, []cloudprovider.TDnsType{cloudprovider.DnsTypeSOA, cloudprovider.DnsTypeNS}); isIn {
				continue
			}
			ret = append(ret, records[i])
		}
		return ret
	}

	add, del, update := []cloudprovider.DnsRecordSet{}, []cloudprovider.DnsRecordSet{}, []cloudprovider.DnsRecordSet{}
	common, _add, _del, _update := cloudprovider.CompareDnsRecordSet(iRecordSets, dbRecordSets, false)
	add, del, update = skipSoaAndNs(_add), skipSoaAndNs(_del), skipSoaAndNs(_update)
	for i := range update {
		_record, err := DnsRecordSetManager.FetchById(update[i].Id)
		if err != nil {
			return errors.Wrapf(err, "DnsRecordSetManager.FetchById(%s)", update[i].Id)
		}
		record := _record.(*SDnsRecordSet)
		update[i].MxPriority = record.MxPriority
		update[i].Enabled = record.GetEnabled()
	}

	log.Infof("sync %s records for %s(%s) common: %d add: %d del: %d update: %d", self.Name, account.Name, account.Provider, len(common), len(add), len(del), len(update))
	return iDnsZone.SyncDnsRecordSets(common, add, del, update)
}

func (manager *SDnsZoneCacheManager) removeCaches(ctx context.Context, userCred mcclient.TokenCredential, accountId string) error {
	q := manager.Query().Equals("cloudaccount_id", accountId)
	caches := []SDnsZoneCache{}
	err := db.FetchModelObjects(manager, q, &caches)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range caches {
		err = caches[i].syncRemove(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "syncRemove cache %s(%s)", caches[i].Name, caches[i].Id)
		}
	}
	return nil
}
