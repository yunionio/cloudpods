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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDnsZoneManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
}

var DnsZoneManager *SDnsZoneManager

func init() {
	DnsZoneManager = &SDnsZoneManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SDnsZone{},
			"dnszones_tbl",
			"dns_zone",
			"dns_zones",
		),
	}
	DnsZoneManager.SetVirtualObject(DnsZoneManager)
}

type SDnsZone struct {
	db.SSharableVirtualResourceBase
	db.SEnabledResourceBase `default:"true" create:"optional" list:"user"`
	db.SExternalizedResourceBase
	SManagedResourceBase

	ZoneType    string `width:"32" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
	ProductType string `width:"32" charset:"ascii" nullable:"false" list:"domain" create:"domain_optional"`
}

// 创建
func (manager *SDnsZoneManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.DnsZoneCreateInput,
) (*api.DnsZoneCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	if !regutils.MatchDomainName(input.Name) {
		return nil, httperrors.NewInputParameterError("invalid domain name %s", input.Name)
	}
	if len(input.ZoneType) == 0 {
		return nil, httperrors.NewMissingParameterError("zone_type")
	}
	var provider *SCloudprovider = nil
	if len(input.CloudproviderId) > 0 {
		providerObj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return nil, err
		}
		provider = providerObj.(*SCloudprovider)
		input.ManagerId = provider.Id
		input.CloudproviderId = provider.Id
	}

	switch cloudprovider.TDnsZoneType(input.ZoneType) {
	case cloudprovider.PrivateZone:
		vpcIds := []string{}
		for i := range input.VpcIds {
			vpcObj, err := validators.ValidateModel(ctx, userCred, VpcManager, &input.VpcIds[i])
			if err != nil {
				return input, err
			}
			vpc := vpcObj.(*SVpc)
			if len(input.ManagerId) == 0 {
				input.ManagerId = vpc.ManagerId
				input.CloudproviderId = vpc.ManagerId
			}
			if vpc.ManagerId != input.ManagerId {
				return nil, httperrors.NewConflictError("conflict cloudprovider %s with vpc %s", input.ManagerId, vpc.Name)
			}
			if len(vpc.ManagerId) > 0 {
				factory, err := vpc.GetProviderFactory()
				if err != nil {
					return input, errors.Wrapf(err, "vpc.GetProviderFactory")
				}
				zoneTypes := factory.GetSupportedDnsZoneTypes()
				if isIn, _ := utils.InArray(cloudprovider.TDnsZoneType(input.ZoneType), zoneTypes); !isIn && len(zoneTypes) > 0 {
					return input, httperrors.NewNotSupportedError("Not support %s for vpc %s, supported %s", input.ZoneType, vpc.Name, zoneTypes)
				}
			}
			vpcIds = append(vpcIds, vpc.GetId())
		}
		input.VpcIds = vpcIds
	case cloudprovider.PublicZone:
		if len(input.CloudproviderId) == 0 {
			return nil, httperrors.NewMissingParameterError("cloudprovider_id")
		}
		input.ManagerId = provider.Id
		factory, err := provider.GetProviderFactory()
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetProviderFactory"))
		}
		zoneTypes := factory.GetSupportedDnsZoneTypes()
		if isIn, _ := utils.InArray(cloudprovider.TDnsZoneType(input.ZoneType), zoneTypes); !isIn && len(zoneTypes) > 0 {
			return input, httperrors.NewNotSupportedError("Not support %s for account %s, supported %s", input.ZoneType, provider.Name, zoneTypes)
		}
		if !strings.ContainsRune(input.Name, '.') {
			return input, httperrors.NewNotSupportedError("top level public domain name %s not support", input.Name)
		}
	default:
		return input, httperrors.NewInputParameterError("unknown zone type %s", input.ZoneType)
	}

	if !gotypes.IsNil(provider) {
		switch provider.Provider {
		case api.CLOUD_PROVIDER_AWS:
			if len(input.VpcIds) == 0 && input.ZoneType == string(cloudprovider.PrivateZone) {
				return nil, httperrors.NewMissingParameterError("vpc_ids")
			}
		}
	}

	quota := &SDomainQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{
			DomainId: ownerId.GetProjectDomainId(),
		},
		DnsZone: 1,
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return input, httperrors.NewOutOfQuotaError("%v", err)
	}
	return input, nil
}

func (self *SDnsZone) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	quota := &SDomainQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{
			DomainId: ownerId.GetProjectDomainId(),
		},
		DnsZone: 1,
	}
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("SDnsZone CancelPendingUsage fail %s", err)
	}
	self.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := &api.DnsZoneCreateInput{}
	data.Unmarshal(input)
	switch cloudprovider.TDnsZoneType(input.ZoneType) {
	case cloudprovider.PrivateZone:
		for _, vpcId := range input.VpcIds {
			self.AddVpc(ctx, vpcId)
		}
	}
	self.StartDnsZoneCreateTask(ctx, userCred, "")
}

func (self *SDnsZone) StartDnsZoneCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.DNS_ZONE_STATUS_CREATING, "")
	return task.ScheduleRun(nil)
}

// 列表
func (manager *SDnsZoneManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsZoneListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, err
	}

	if len(query.ZoneType) > 0 {
		q = q.Equals("zone_type", query.ZoneType)
	}

	if len(query.VpcId) > 0 {
		vpc, err := VpcManager.FetchByIdOrName(ctx, userCred, query.VpcId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("vpc", query.VpcId)
			}
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "VpcManager.FetchByIdOrName"))
		}
		sq := DnsZoneVpcManager.Query("dns_zone_id").Equals("vpc_id", vpc.GetId())
		q = q.In("id", sq.SubQuery())
	}
	return q, nil
}

// 解析详情
func (manager *SDnsZoneManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DnsZoneDetails {
	rows := make([]api.DnsZoneDetails, len(objs))
	enRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.DnsZoneDetails{
			SharableVirtualResourceDetails: enRows[i],
			ManagedResourceInfo:            managerRows[i],
		}
		zone := objs[i].(*SDnsZone)
		zoneIds[i] = zone.Id
	}

	q := DnsZoneVpcManager.Query().In("dns_zone_id", zoneIds)
	vpcs := []SDnsZoneVpc{}
	err := q.All(&vpcs)
	if err != nil {
		log.Errorf("query dns zone vpcs error: %v", err)
		return rows
	}
	vpcMap := map[string][]string{}
	for i := range vpcs {
		_, ok := vpcMap[vpcs[i].DnsZoneId]
		if !ok {
			vpcMap[vpcs[i].DnsZoneId] = []string{}
		}
		vpcMap[vpcs[i].DnsZoneId] = append(vpcMap[vpcs[i].DnsZoneId], vpcs[i].VpcId)
	}
	q = DnsRecordManager.Query().In("dns_zone_id", zoneIds)
	records := []SDnsRecord{}
	err = q.All(&records)
	if err != nil {
		log.Errorf("query dns zone records error: %v", err)
		return rows
	}

	recordMap := map[string][]SDnsRecord{}
	for i := range records {
		_, ok := recordMap[records[i].DnsZoneId]
		if !ok {
			recordMap[records[i].DnsZoneId] = []SDnsRecord{}
		}
		recordMap[records[i].DnsZoneId] = append(recordMap[records[i].DnsZoneId], records[i])
	}

	for i := range rows {
		vpcs, _ := vpcMap[zoneIds[i]]
		rows[i].VpcCount = len(vpcs)
		records, _ := recordMap[zoneIds[i]]
		rows[i].DnsRecordCount = len(records)
	}

	return rows
}

func (self *SDnsZone) RemoveVpc(ctx context.Context, vpcId string) error {
	q := DnsZoneVpcManager.Query().Equals("dns_zone_id", self.Id).Equals("vpc_id", vpcId)
	zvs := []SDnsZoneVpc{}
	err := db.FetchModelObjects(DnsZoneVpcManager, q, &zvs)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range zvs {
		err = zvs[i].Delete(ctx, nil)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

func (self *SDnsZone) AddVpc(ctx context.Context, vpcId string) error {
	zv := &SDnsZoneVpc{}
	zv.SetModelManager(DnsZoneVpcManager, zv)
	zv.VpcId = vpcId
	zv.DnsZoneId = self.Id
	return DnsZoneVpcManager.TableSpec().Insert(ctx, zv)
}

func (self *SDnsZone) GetVpcs() ([]SVpc, error) {
	sq := DnsZoneVpcManager.Query("vpc_id").Equals("dns_zone_id", self.Id)
	q := VpcManager.Query().In("id", sq.SubQuery())
	vpcs := []SVpc{}
	err := db.FetchModelObjects(VpcManager, q, &vpcs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return vpcs, nil
}

func (self *SDnsZone) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDnsZoneDeleteTask(ctx, userCred, "")
}

func (self *SDnsZone) StartDnsZoneDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.DNS_ZONE_STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SDnsZone) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SDnsZone) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	dnsVpcs := DnsZoneVpcManager.Query("row_id").Equals("dns_zone_id", self.Id)
	records := DnsRecordManager.Query("id").Equals("dns_zone_id", self.Id)

	pairs := []purgePair{
		{manager: DnsZoneVpcManager, key: "row_id", q: dnsVpcs},
		{manager: DnsRecordManager, key: "id", q: records},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}

	return self.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SDnsZone) GetDetailsExports(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	records, err := self.GetDnsRecordSets()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDnsRecordSets")
	}
	result := "$ORIGIN " + self.Name + ".\n"
	lines := []string{}
	for _, record := range records {
		lines = append(lines, record.ToZoneLine())
	}
	result += strings.Join(lines, "\n")
	rr := make(map[string]string)
	rr["zone"] = result
	return jsonutils.Marshal(rr), nil
}

func (self *SDnsZone) GetDnsRecordSets() ([]SDnsRecord, error) {
	records := []SDnsRecord{}
	q := DnsRecordManager.Query().Equals("dns_zone_id", self.Id)
	err := db.FetchModelObjects(DnsRecordManager, q, &records)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return records, nil
}

func (self *SCloudprovider) GetDnsZones() ([]SDnsZone, error) {
	q := DnsZoneManager.Query().Equals("manager_id", self.Id)
	ret := []SDnsZone{}
	return ret, db.FetchModelObjects(DnsZoneManager, q, &ret)
}

func (self *SCloudprovider) SyncDnsZones(ctx context.Context, userCred mcclient.TokenCredential, zones []cloudprovider.ICloudDnsZone, xor bool) ([]SDnsZone, []cloudprovider.ICloudDnsZone, compare.SyncResult) {
	lockman.LockRawObject(ctx, DnsZoneManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, DnsZoneManager.Keyword(), self.Id)

	local := make([]SDnsZone, 0)
	remote := make([]cloudprovider.ICloudDnsZone, 0)
	result := compare.SyncResult{}

	dbZones, err := self.GetDnsZones()
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	removed := make([]SDnsZone, 0)
	commondb := make([]SDnsZone, 0)
	commonext := make([]cloudprovider.ICloudDnsZone, 0)
	added := make([]cloudprovider.ICloudDnsZone, 0)

	err = compare.CompareSets(dbZones, zones, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveDnsZone(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].syncWithDnsZone(ctx, userCred, self, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			local = append(local, commondb[i])
			remote = append(remote, commonext[i])
			result.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		zone, err := self.newFromCloudDnsZone(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		local = append(local, *zone)
		remote = append(remote, added[i])
		result.Add()
	}

	return local, remote, result
}

func (self *SDnsZone) syncRemoveDnsZone(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SDnsZone) GetProvider(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	if len(self.ManagerId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty manager id")
	}
	provider := self.GetCloudprovider()
	if provider == nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "failed to found provider")
	}
	return provider.GetProvider(ctx)
}

func (self *SDnsZone) GetICloudDnsZone(ctx context.Context) (cloudprovider.ICloudDnsZone, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	provider, err := self.GetProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetICloudDnsZoneById(self.ExternalId)
}

func (self *SDnsZone) syncWithDnsZone(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudDnsZone) error {
	_, err := db.Update(self, func() error {
		self.Status = ext.GetStatus()
		self.ProductType = string(ext.GetDnsProductType())
		return nil
	})
	if err != nil {
		return err
	}

	privider := self.GetCloudprovider()
	if privider != nil {
		if account, _ := provider.GetCloudaccount(); account != nil {
			syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
		}
		SyncCloudProject(ctx, userCred, self, provider.GetOwnerId(), ext, provider)
	}

	return nil
}

func (self *SCloudprovider) newFromCloudDnsZone(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsZone) (*SDnsZone, error) {
	zone := &SDnsZone{}
	zone.Name = ext.GetName()
	zone.ExternalId = ext.GetGlobalId()
	zone.ManagerId = self.Id
	zone.Status = ext.GetStatus()
	zone.Enabled = tristate.True
	zone.ZoneType = string(ext.GetZoneType())
	zone.ProductType = string(ext.GetDnsProductType())
	zone.SetModelManager(DnsZoneManager, zone)
	err := DnsZoneManager.TableSpec().Insert(ctx, zone)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncVirtualResourceMetadata(ctx, userCred, zone, ext, false)
	SyncCloudProject(ctx, userCred, zone, self.GetOwnerId(), ext, self)

	return zone, nil
}

func (self *SDnsZone) GetDnsRecords() ([]SDnsRecord, error) {
	q := DnsRecordManager.Query().Equals("dns_zone_id", self.Id)
	ret := []SDnsRecord{}
	return ret, db.FetchModelObjects(DnsRecordManager, q, &ret)
}

// 同步状态
func (self *SDnsZone) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneSyncStatusInput) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "DnsZoneSyncstatusTask", "")
}

// 添加VPC
func (self *SDnsZone) PerformAddVpcs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneAddVpcsInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DNS_ZONE_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("dns zone can not uncache in status %s", self.Status)
	}
	if cloudprovider.TDnsZoneType(self.ZoneType) != cloudprovider.PrivateZone {
		return nil, httperrors.NewUnsupportOperationError("Only %s support cache for account", cloudprovider.PrivateZone)
	}
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetVpcs"))
	}
	localVpcIds := []string{}
	for _, vpc := range vpcs {
		localVpcIds = append(localVpcIds, vpc.Id)
	}

	if len(input.VpcIds) == 0 {
		return nil, httperrors.NewMissingParameterError("vpc_ids")
	}

	for i := range input.VpcIds {
		vpcObj, err := validators.ValidateModel(ctx, userCred, VpcManager, &input.VpcIds[i])
		if err != nil {
			return nil, err
		}
		vpc := vpcObj.(*SVpc)
		if utils.IsInStringArray(vpc.GetId(), localVpcIds) {
			return nil, httperrors.NewConflictError("vpc %s has already in this dns zone", input.VpcIds[i])
		}
		if vpc.ManagerId != self.ManagerId {
			return nil, httperrors.NewConflictError("vpc %s not same with dns zone account", input.VpcIds[i])
		}
	}
	return nil, self.StartDnsZoneAddVpcsTask(ctx, userCred, input.VpcIds, "")
}

func (self *SDnsZone) StartDnsZoneAddVpcsTask(ctx context.Context, userCred mcclient.TokenCredential, vpcIds []string, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("vpc_ids", jsonutils.NewStringArray(vpcIds))
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneAddVpcsTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

func (self *SDnsZone) StartDnsZoneRemoveVpcsTask(ctx context.Context, userCred mcclient.TokenCredential, vpcIds []string, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("vpc_ids", jsonutils.NewStringArray(vpcIds))
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneRemoveVpcsTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

// 移除VPC
func (self *SDnsZone) PerformRemoveVpcs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneRemoveVpcsInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DNS_ZONE_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("dns zone can not uncache in status %s", self.Status)
	}
	if cloudprovider.TDnsZoneType(self.ZoneType) != cloudprovider.PrivateZone {
		return nil, httperrors.NewUnsupportOperationError("Only %s support cache for account", cloudprovider.PrivateZone)
	}
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetVpcs"))
	}
	vpcIds := []string{}
	for _, vpc := range vpcs {
		vpcIds = append(vpcIds, vpc.Id)
	}
	for _, vpcId := range input.VpcIds {
		if !utils.IsInStringArray(vpcId, vpcIds) {
			return nil, httperrors.NewResourceNotFoundError("vpc %s not in dns zone", vpcId)
		}
	}
	return nil, self.StartDnsZoneRemoveVpcsTask(ctx, userCred, input.VpcIds, "")
}

func (manager *SDnsZoneManager) GetPropertyCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(cloudprovider.GetDnsCapabilities()), nil
}

func (self *SDnsZone) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZonePurgeInput) (jsonutils.JSONObject, error) {
	return nil, self.RealDelete(ctx, userCred)
}

func (manager *SDnsZoneManager) totalCount(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider) int {
	q := manager.Query()
	switch scope {
	case rbacscope.ScopeProject, rbacscope.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}
	cnt, _ := q.CountWithError()
	return cnt
}

func (dzone *SDnsZone) GetUsages() []db.IUsage {
	if dzone.Deleted {
		return nil
	}
	usage := SDomainQuota{DnsZone: 1}
	usage.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: dzone.DomainId})
	return []db.IUsage{
		&usage,
	}
}

func (manager *SDnsZoneManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SDnsZoneManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsZoneListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}
