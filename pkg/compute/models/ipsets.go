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
	"net"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=ipset
// +onecloud:swagger-gen-model-plural=ipsets
type SIpSetManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var IpSetManager *SIpSetManager

func init() {
	IpSetManager = &SIpSetManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SIpSet{},
			"ipsets_tbl",
			"ipset",
			"ipsets",
		),
	}
	IpSetManager.SetVirtualObject(IpSetManager)
}

type SIpSet struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// IP集合类型
	IpSetType api.TIpSetType `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	// IP/CIDR 列表，逗号分隔
	Data string `width:"2048" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
}

func (manager *SIpSetManager) InitializeData() error {
	q := manager.Query().IsNullOrEmpty("manager_id").Equals("status", "init")
	ipsets := []SIpSet{}
	err := db.FetchModelObjects(manager, q, &ipsets)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range ipsets {
		_, err := db.Update(&ipsets[i], func() error {
			ipsets[i].Status = apis.STATUS_AVAILABLE
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "Update ipset %s", ipsets[i].Id)
		}
	}
	return nil
}

func ipSetRegionListFilter(ctx context.Context, q *sqlchemy.SQuery, query api.RegionalFilterListInput) (*sqlchemy.SQuery, error) {
	regionIds := []string{}
	for _, region := range query.CloudregionId {
		if len(region) == 0 {
			continue
		}
		regionObj, err := ValidateCloudregionId(ctx, nil, region)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateCloudregionId")
		}
		regionIds = append(regionIds, regionObj.GetId())
	}
	if len(regionIds) > 0 {
		providerSubq := CloudproviderRegionManager.Query("cloudprovider_id").In("cloudregion_id", regionIds).Distinct().SubQuery()
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("cloudregion_id"), regionIds),
			sqlchemy.AND(
				sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")),
				sqlchemy.In(q.Field("manager_id"), providerSubq),
			),
		))
	}
	if len(query.City) > 0 {
		subq := CloudregionManager.Query("id").Equals("city", query.City).SubQuery()
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("cloudregion_id"), subq),
			sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")),
		))
	}
	return q, nil
}

// IP集合列表
func (manager *SIpSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IpSetListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = ipSetRegionListFilter(ctx, q, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "ipSetRegionListFilter")
	}
	if len(query.IpSetType) > 0 {
		types := make([]string, 0, len(query.IpSetType))
		for _, t := range query.IpSetType {
			types = append(types, string(t))
		}
		q = q.In("ip_set_type", types)
	}
	if len(query.Ip) > 0 {
		q = q.Contains("data", query.Ip)
	}
	return q, nil
}

func (manager *SIpSetManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IpSetListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SIpSetManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SIpSetManager) QueryDistinctExtraFields(q *sqlchemy.SQuery, resource string, fields []string) (*sqlchemy.SQuery, error) {
	q, err := manager.SManagedResourceBaseManager.QueryDistinctExtraFields(q, resource, fields)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

type sIpSetUsageCount struct {
	Cidr               string
	SecurityGroupCount int
}

func (manager *SIpSetManager) TotalResourceCount(ipSetIds []string) (map[string]int, error) {
	ret := map[string]int{}
	if len(ipSetIds) == 0 {
		return ret, nil
	}
	sq := SecurityGroupRuleManager.Query("cidr", "secgroup_id").
		Equals("target_type", api.SecurityGroupRuleTargetTypeIpSet).
		In("cidr", ipSetIds).
		Distinct().SubQuery()
	q := sq.Query(
		sq.Field("cidr"),
		sqlchemy.COUNT("security_group_count"),
	).GroupBy(sq.Field("cidr"))
	counts := []sIpSetUsageCount{}
	err := q.All(&counts)
	if err != nil {
		return nil, errors.Wrapf(err, "q.All")
	}
	for i := range counts {
		ret[counts[i].Cidr] = counts[i].SecurityGroupCount
	}
	return ret, nil
}

func (manager *SIpSetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.IpSetDetails {
	rows := make([]api.IpSetDetails, len(objs))
	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	ipSetIds := make([]string, len(objs))
	for i := range rows {
		ipSetIds[i] = objs[i].(*SIpSet).Id
		rows[i] = api.IpSetDetails{
			SharableVirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:            managerRows[i],
			CloudregionResourceInfo:        regionRows[i],
		}
	}
	counts, err := manager.TotalResourceCount(ipSetIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}
	for i := range rows {
		rows[i].SecurityGroupCount = counts[ipSetIds[i]]
	}
	return rows
}

func (manager *SIpSetManager) ListItemExportKeys(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
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

func (manager *SIpSetManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.IpSetCreateInput,
) (*api.IpSetCreateInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	input.Data = normalizeIpSetData(input.IpSetType, input.Data)
	if len(input.Data) == 0 {
		return nil, httperrors.NewInputParameterError("Data is empty")
	}

	var err error
	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	if len(input.CloudproviderId) > 0 {
		providerObj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return nil, err
		}
		input.ManagerId = input.CloudproviderId
		provider := providerObj.(*SCloudprovider)
		if len(input.CloudregionId) > 0 {
			regionObj, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
			if err != nil {
				return nil, err
			}
			region := regionObj.(*SCloudregion)
			if provider.Provider != region.Provider {
				return nil, httperrors.NewConflictError("conflict region %s and cloudprovider %s", region.Name, provider.Name)
			}
		} else if provider.Provider != api.CLOUD_PROVIDER_QCLOUD {
			return nil, httperrors.NewMissingParameterError("cloudregion_id")
		} else {
			input.CloudregionId = ""
		}
		return input, nil
	}
	if len(input.CloudregionId) == 0 {
		input.CloudregionId = api.DEFAULT_REGION_ID
	}
	regionObj, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return nil, err
	}
	_ = regionObj.(*SCloudregion)
	return input, nil
}

func (ipset *SIpSet) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.IpSetUpdateInput,
) (*api.IpSetUpdateInput, error) {
	if input.Data != nil {
		normalized := normalizeIpSetData(ipset.IpSetType, *input.Data)
		if len(normalized) == 0 {
			return nil, httperrors.NewInputParameterError("Data is empty")
		}
		input.Data = &normalized
	}

	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = ipset.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (ipset *SIpSet) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	ipset.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if ipset.IsManaged() {
		regionId, _ := data.GetString("cloudregion_id")
		if len(regionId) == 0 {
			provider := ipset.GetCloudprovider()
			if provider != nil && provider.Provider == api.CLOUD_PROVIDER_QCLOUD && len(ipset.CloudregionId) > 0 {
				_, err := db.Update(ipset, func() error {
					ipset.CloudregionId = ""
					return nil
				})
				if err != nil {
					log.Errorf("clear cloudregion_id for ipset %s: %v", ipset.Id, err)
				}
			}
		}
		ipset.StartCreateTask(ctx, userCred, "")
		return
	}
	ipset.SetStatus(ctx, userCred, apis.STATUS_AVAILABLE, "")
	ipset.DoSync(ctx, userCred)
}

func (ipset *SIpSet) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	ipset.SSharableVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	if ipset.IsManaged() {
		ipset.StartUpdateTask(ctx, userCred, "")
		return
	}
	ipset.DoSync(ctx, userCred)
}

func (ipset *SIpSet) GetSecurityGroups() ([]SSecurityGroup, error) {
	sq := SecurityGroupRuleManager.Query("secgroup_id").
		Equals("target_type", api.SecurityGroupRuleTargetTypeIpSet).
		Equals("cidr", ipset.Id).
		Distinct().SubQuery()
	q := SecurityGroupManager.Query().In("id", sq)
	ret := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (ipset *SIpSet) DoSync(ctx context.Context, userCred mcclient.TokenCredential) {
	secgroups, err := ipset.GetSecurityGroups()
	if err != nil {
		log.Errorf("GetSecurityGroups error: %s", err.Error())
		return
	}
	for i := range secgroups {
		if len(secgroups[i].ManagerId) == 0 {
			secgroups[i].DoSync(ctx, userCred)
		}
	}
}

func normalizeIpSetData(ipSetType api.TIpSetType, data string) string {
	parts := strings.Split(data, ",")
	normalized := make([]string, 0, len(parts))
	for i := range parts {
		cidr := strings.TrimSpace(parts[i])
		if len(cidr) == 0 {
			continue
		}
		switch ipSetType {
		case api.IpSetTypeIpv4CidrList:
			if pref, err := netutils.NewIPV4Prefix(cidr); err == nil {
				normalized = append(normalized, pref.String())
			}
		case api.IpSetTypeIpv6CidrList:
			if pref, err := netutils.NewIPV6Prefix(cidr); err == nil {
				normalized = append(normalized, pref.String())
			}
		}
	}
	return strings.Join(normalized, ",")
}

func (ipset *SIpSet) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if info != nil && info.Contains("security_group_count") {
		securityGroupCount, _ := info.Int("security_group_count")
		if securityGroupCount > 0 {
			return errors.Wrapf(httperrors.ErrNotEmpty, "ipset is being used by %d security groups", securityGroupCount)
		}
	} else if ipset.GetSecurityGroupCount() > 0 {
		return errors.Wrapf(httperrors.ErrNotEmpty, "ipset is being used by %d security groups", ipset.GetSecurityGroupCount())
	}
	err := ipset.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, info)
	if err != nil {
		return errors.Wrapf(err, "SSharableVirtualResourceBase.ValidateDeleteCondition")
	}
	return nil
}

func (ipset *SIpSet) GetSecurityGroupCount() int {
	counts, err := IpSetManager.TotalResourceCount([]string{ipset.Id})
	if err != nil {
		return 0
	}
	return counts[ipset.Id]
}

func (ipset *SIpSet) getIpNets() []*net.IPNet {
	return netutils2.Str2IPNets(ipset.Data)
}

func (ipset *SIpSet) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	if ipset.IsManaged() && len(ipset.CloudregionId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "account level ipset")
	}
	region, err := ipset.SCloudregionResourceBase.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := ipset.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (ipset *SIpSet) GetICloudIpSet(ctx context.Context) (cloudprovider.ICloudIpSet, error) {
	if len(ipset.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	if ipset.IsManaged() && len(ipset.CloudregionId) == 0 {
		provider := ipset.GetCloudprovider()
		if provider == nil {
			return nil, errors.Errorf("empty cloudprovider")
		}
		driver, err := provider.GetProvider(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetProvider")
		}
		return driver.GetIIpSetById(ipset.ExternalId)
	}
	iRegion, err := ipset.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	return iRegion.GetIIpSetById(ipset.ExternalId)
}

func (ipset *SIpSet) GetAddresses() []string {
	if len(ipset.Data) == 0 {
		return []string{}
	}
	parts := strings.Split(ipset.Data, ",")
	ret := make([]string, 0, len(parts))
	for i := range parts {
		addr := strings.TrimSpace(parts[i])
		if len(addr) > 0 {
			ret = append(ret, addr)
		}
	}
	return ret
}

func (ipset *SIpSet) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "IpSetCreateTask", ipset, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	ipset.SetStatus(ctx, userCred, apis.STATUS_CREATING, "")
	return task.ScheduleRun(nil)
}

func (ipset *SIpSet) StartUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "IpSetUpdateTask", ipset, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	ipset.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

func (ipset *SIpSet) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "IpSetDeleteTask", ipset, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	ipset.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (ipset *SIpSet) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if ipset.IsManaged() {
		return nil
	}
	return ipset.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (ipset *SIpSet) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return ipset.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (ipset *SIpSet) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if ipset.IsManaged() {
		return ipset.StartDeleteTask(ctx, userCred, "")
	}
	return nil
}

func (ipset *SIpSet) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !ipset.IsManaged() {
		return nil, ipset.SetStatus(ctx, userCred, apis.STATUS_AVAILABLE, "")
	}
	return nil, StartResourceSyncStatusTask(ctx, userCred, ipset, "IpSetSyncstatusTask", "")
}

func (self *SCloudregion) GetIpSets(managerId string) ([]SIpSet, error) {
	q := IpSetManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SIpSet{}
	err := db.FetchModelObjects(IpSetManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncIpSets(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudIpSet,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, IpSetManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, IpSetManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbIpSets, err := self.GetIpSets(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SIpSet, 0)
	commondb := make([]SIpSet, 0)
	commonext := make([]cloudprovider.ICloudIpSet, 0)
	added := make([]cloudprovider.ICloudIpSet, 0)
	err = compare.CompareSets(dbIpSets, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		for i := 0; i < len(commondb); i++ {
			err := commondb[i].SyncWithCloudIpSet(ctx, userCred, commonext[i], provider, self.Id)
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		err = self.newFromCloudIpSet(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (provider *SCloudprovider) GetIpSets() ([]SIpSet, error) {
	q := IpSetManager.Query().Equals("manager_id", provider.Id)
	ret := []SIpSet{}
	err := db.FetchModelObjects(IpSetManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (provider *SCloudprovider) SyncIpSets(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	exts []cloudprovider.ICloudIpSet,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, IpSetManager.Keyword(), provider.Id)
	defer lockman.ReleaseRawObject(ctx, IpSetManager.Keyword(), provider.Id)

	result := compare.SyncResult{}

	dbIpSets, err := provider.GetIpSets()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SIpSet, 0)
	commondb := make([]SIpSet, 0)
	commonext := make([]cloudprovider.ICloudIpSet, 0)
	added := make([]cloudprovider.ICloudIpSet, 0)
	err = compare.CompareSets(dbIpSets, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		if len(removed[i].CloudregionId) > 0 {
			continue
		}
		err := removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		for i := 0; i < len(commondb); i++ {
			err := commondb[i].SyncWithCloudIpSet(ctx, userCred, commonext[i], provider, "")
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		err = provider.newFromCloudIpSet(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (provider *SCloudprovider) newFromCloudIpSet(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudIpSet) error {
	ret := &SIpSet{}
	ret.SetModelManager(IpSetManager, ret)
	ret.Name = ext.GetName()
	ret.CloudregionId = ""
	ret.ManagerId = provider.Id
	ret.ExternalId = ext.GetGlobalId()
	ret.Status = apis.STATUS_AVAILABLE
	ret.IpSetType = api.TIpSetType(ext.GetIpSetType())
	ret.Description = ext.GetDescription()
	ret.Data = strings.Join(ext.GetAddresses(), ",")
	err := IpSetManager.TableSpec().Insert(ctx, ret)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}
	syncVirtualResourceMetadata(ctx, userCred, ret, ext, false)
	SyncCloudProject(ctx, userCred, ret, provider.GetOwnerId(), ext, provider)
	return nil
}

func (ipset *SIpSet) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return ipset.RealDelete(ctx, userCred)
}

func (ipset *SIpSet) SyncWithCloudIpSet(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudIpSet, provider *SCloudprovider, cloudregionId string) error {
	_, err := db.Update(ipset, func() error {
		ipset.Status = apis.STATUS_AVAILABLE
		ipset.CloudregionId = cloudregionId
		if options.Options.EnableSyncName {
			ipset.Name = ext.GetName()
		}
		ipset.IpSetType = api.TIpSetType(ext.GetIpSetType())
		ipset.Data = strings.Join(ext.GetAddresses(), ",")
		if desc := ext.GetDescription(); len(desc) > 0 {
			ipset.Description = desc
		}
		return nil
	})
	if err != nil {
		return err
	}
	syncVirtualResourceMetadata(ctx, userCred, ipset, ext, false)
	SyncCloudProject(ctx, userCred, ipset, provider.GetOwnerId(), ext, provider)
	return nil
}

func (self *SCloudregion) newFromCloudIpSet(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudIpSet) error {
	ret := &SIpSet{}
	ret.SetModelManager(IpSetManager, ret)
	ret.Name = ext.GetName()
	ret.CloudregionId = self.Id
	ret.ManagerId = provider.Id
	ret.ExternalId = ext.GetGlobalId()
	ret.Status = apis.STATUS_AVAILABLE
	ret.IpSetType = api.TIpSetType(ext.GetIpSetType())
	ret.Description = ext.GetDescription()
	ret.Data = strings.Join(ext.GetAddresses(), ",")
	err := IpSetManager.TableSpec().Insert(ctx, ret)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}
	syncVirtualResourceMetadata(ctx, userCred, ret, ext, false)
	SyncCloudProject(ctx, userCred, ret, provider.GetOwnerId(), ext, provider)
	return nil
}
