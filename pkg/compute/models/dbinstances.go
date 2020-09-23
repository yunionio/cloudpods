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
	"net"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDBInstanceManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDeletePreventableResourceBaseManager

	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
	SVpcResourceBaseManager
}

var DBInstanceManager *SDBInstanceManager

func init() {
	DBInstanceManager = &SDBInstanceManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDBInstance{},
			"dbinstances_tbl",
			"dbinstance",
			"dbinstances",
		),
	}
	DBInstanceManager.SetVirtualObject(DBInstanceManager)
}

type SDBInstance struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SBillingResourceBase

	SCloudregionResourceBase
	SDeletePreventableResourceBase

	// 主实例Id
	MasterInstanceId string `width:"128" charset:"ascii" list:"user" create:"optional"`
	// CPU数量
	// example: 1
	VcpuCount int `nullable:"false" default:"1" list:"user" create:"optional"`
	// 内存大小
	// example: 1024
	VmemSizeMb int `nullable:"false" list:"user" create:"required"`
	// 存储类型
	// example: local_ssd
	StorageType string `nullable:"false" list:"user" create:"required"`
	// 存储大小
	// example: 10240
	DiskSizeGB int `nullable:"false" list:"user" create:"required"`
	// 端口
	// example: 3306
	Port int `nullable:"false" list:"user" create:"optional"`
	// 实例类型
	// example: ha
	Category string `nullable:"false" list:"user" create:"optional"`

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

	// 外部连接地址
	ConnectionStr string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	// 内部连接地址
	InternalConnectionStr string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"`

	// 可用区1
	Zone1 string `width:"36" charset:"ascii" nullable:"false" create:"optional" list:"user"`
	// 可用区2
	Zone2 string `width:"36" charset:"ascii" nullable:"false" create:"optional" list:"user"`
	// 可用区3
	Zone3 string `width:"36" charset:"ascii" nullable:"false" create:"optional" list:"user"`

	// 从备份创建新实例
	DBInstancebackupId string `width:"36" name:"dbinstancebackup_id" charset:"ascii" nullable:"false" create:"optional"`
}

func (manager *SDBInstanceManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

// RDS实例列表
func (man *SDBInstanceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceListInput,
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

	if len(query.ZoneId) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(userCred, query.ZoneId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ZoneManager.Keyword(), query.ZoneId)
			} else {
				return nil, errors.Wrap(err, "ZoneManager.FetchByIdOrName")
			}
		}
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("zone1"), zoneObj.GetId()),
			sqlchemy.Equals(q.Field("zone2"), zoneObj.GetId()),
			sqlchemy.Equals(q.Field("zone3"), zoneObj.GetId()),
		))
	}

	if len(query.MasterInstance) > 0 {
		instObj, err := DBInstanceManager.FetchByIdOrName(userCred, query.MasterInstance)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(DBInstanceManager.Keyword(), query.MasterInstance)
			} else {
				return nil, errors.Wrap(err, "DBInstanceManager.FetchByIdOrName")
			}
		}
		q = q.Equals("master_instance_id", instObj.GetId())
	}
	if query.VcpuCount > 0 {
		q = q.Equals("vcpu_count", query.VcpuCount)
	}
	if query.VmemSizeMb > 0 {
		q = q.Equals("vmem_size_mb", query.VmemSizeMb)
	}
	if len(query.StorageType) > 0 {
		q = q.Equals("storage_type", query.StorageType)
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

func (man *SDBInstanceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceListInput,
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

func (man *SDBInstanceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *SDBInstanceManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := api.DBInstanceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrapf(err, "data.Unmarshal")
	}
	input, err = manager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, errors.Wrapf(err, "ValidateCreateData")
	}
	return input.JSON(input), nil
}

func (man *SDBInstanceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DBInstanceCreateInput) (api.DBInstanceCreateInput, error) {
	if len(input.DBInstancebackupId) > 0 {
		_backup, err := validators.ValidateModel(userCred, DBInstanceBackupManager, &input.DBInstancebackupId)
		if err != nil {
			return input, err
		}
		backup := _backup.(*SDBInstanceBackup)
		err = backup.fillRdsConfig(&input)
		if err != nil {
			return input, err
		}
	}
	for _, v := range map[string]*string{"zone1": &input.Zone1, "zone2": &input.Zone2, "zone3": &input.Zone3} {
		if len(*v) > 0 {
			_, err := validators.ValidateModel(userCred, ZoneManager, v)
			if err != nil {
				return input, err
			}
		}
	}

	if len(input.Password) > 0 {
		err := seclib2.ValidatePassword(input.Password)
		if err != nil {
			return input, err
		}
	}
	var vpc *SVpc
	var network *SNetwork
	if len(input.NetworkId) > 0 {
		_network, err := validators.ValidateModel(userCred, NetworkManager, &input.NetworkId)
		if err != nil {
			return input, err
		}
		network = _network.(*SNetwork)
		if len(input.Address) > 0 {
			ip := net.ParseIP(input.Address).To4()
			if ip == nil {
				return input, httperrors.NewInputParameterError("invalid address: %s", input.Address)
			}
			addr, _ := netutils.NewIPV4Addr(input.Address)
			if !network.IsAddressInRange(addr) {
				return input, httperrors.NewInputParameterError("Ip %s not in network %s(%s) range", input.Address, network.Name, network.Id)
			}
		}
		vpc = network.GetVpc()
	} else if len(input.VpcId) > 0 {
		_vpc, err := validators.ValidateModel(userCred, VpcManager, &input.VpcId)
		if err != nil {
			return input, err
		}
		vpc = _vpc.(*SVpc)
	} else {
		return input, httperrors.NewMissingParameterError("vpc_id")
	}

	input.VpcId = vpc.Id
	input.ManagerId = vpc.ManagerId
	cloudprovider := vpc.GetCloudprovider()
	if cloudprovider == nil {
		return input, httperrors.NewGeneralError(fmt.Errorf("failed to get vpc %s(%s) cloudprovider", vpc.Name, vpc.Id))
	}
	if !cloudprovider.IsAvailable() {
		return input, httperrors.NewInputParameterError("cloudprovider %s(%s) is not available", cloudprovider.Name, cloudprovider.Id)
	}
	region, err := vpc.GetRegion()
	if err != nil {
		return input, err
	}
	input.CloudregionId = region.Id

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

	for k, v := range map[string]string{
		"engine":         input.Engine,
		"engine_version": input.EngineVersion,
		"category":       input.Category,
		"storage_type":   input.StorageType,
	} {
		if len(v) == 0 {
			return input, httperrors.NewMissingParameterError(k)
		}
	}

	info := getDBInstanceInfo(region, nil)
	if info == nil {
		return input, httperrors.NewNotSupportedError("cloudregion %s not support create rds", region.Name)
	}

	versionsInfo, ok := info[input.Engine]
	if !ok {
		return input, httperrors.NewNotSupportedError("cloudregion %s not support create %s rds", region.Name, input.Engine)
	}

	categoryInfo, ok := versionsInfo[input.EngineVersion]
	if !ok {
		return input, httperrors.NewNotSupportedError("cloudregion %s not support create %s rds", region.Name, input.EngineVersion)
	}

	storageInfo, ok := categoryInfo[input.Category]
	if !ok {
		return input, httperrors.NewNotSupportedError("cloudregion %s not support create %s rds", region.Name, input.Category)
	}

	if !utils.IsInStringArray(input.StorageType, storageInfo) {
		return input, httperrors.NewNotSupportedError("cloudregion %s not support create %s rds", region.Name, input.StorageType)
	}

	if len(input.InstanceType) == 0 && (input.VcpuCount == 0 || input.VmemSizeMb == 0) {
		return input, httperrors.NewMissingParameterError("Missing instance_type or vcpu_count, vmem_size_mb parameters")
	}

	instance := SDBInstance{}
	jsonutils.Update(&instance, input)
	skus, err := instance.GetAvailableDBInstanceSkus(false)
	if err != nil {
		return input, httperrors.NewGeneralError(err)
	}

	if len(skus) == 0 {
		return input, httperrors.NewInputParameterError("not match any dbinstance sku")
	}

	if len(input.InstanceType) > 0 { //设置下cpu和内存的大小
		input.VcpuCount = skus[0].VcpuCount
		input.VmemSizeMb = skus[0].VmemSizeMb
	}

	input.VirtualResourceCreateInput, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	driver := region.GetDriver()
	secCount := driver.GetRdsSupportSecgroupCount()
	if secCount == 0 && len(input.SecgroupIds) > 0 {
		return input, httperrors.NewNotSupportedError("%s rds not support secgroup", driver.GetProvider())
	}
	if len(input.SecgroupIds) > secCount {
		return input, httperrors.NewNotSupportedError("%s rds Support up to %d security groups", driver.GetProvider(), secCount)
	}
	for i := range input.SecgroupIds {
		_, err := validators.ValidateModel(userCred, SecurityGroupManager, &input.SecgroupIds[i])
		if err != nil {
			return input, err
		}
	}

	input, err = driver.ValidateCreateDBInstanceData(ctx, userCred, ownerId, input, skus, network)
	if err != nil {
		return input, err
	}

	quotaKeys := fetchRegionalQuotaKeys(rbacutils.ScopeProject, ownerId, region, cloudprovider)
	pendingUsage := SRegionQuota{Rds: 1}
	pendingUsage.SetKeys(quotaKeys)
	if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
		return input, httperrors.NewOutOfQuotaError("%s", err)
	}

	return input, nil
}

func (self *SDBInstance) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	pendingUsage := SRegionQuota{Rds: 1}
	pendingUsage.SetKeys(self.GetQuotaKeys())
	err := quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, true)
	if err != nil {
		log.Errorf("CancelPendingUsage error %s", err)
	}

	input := api.DBInstanceCreateInput{}
	data.Unmarshal(&input)
	if len(input.NetworkId) > 0 {
		err := DBInstanceNetworkManager.newNetwork(ctx, userCred, self.Id, input.NetworkId, input.Address)
		if err != nil {
			log.Errorf("DBInstanceNetworkManager.Insert")
		}
	}
	ids := []string{}
	for _, secgroupId := range input.SecgroupIds {
		if !utils.IsInStringArray(secgroupId, ids) {
			err := self.assignSecgroup(ctx, userCred, secgroupId)
			if err != nil {
				log.Errorf("assignSecgroup")
			}
			ids = append(ids, secgroupId)
		}
	}
	resetPassword := true
	if input.ResetPassword != nil && !*input.ResetPassword {
		resetPassword = false
	}
	self.StartDBInstanceCreateTask(ctx, userCred, resetPassword, input.Password, "")
}

func (self *SDBInstance) StartDBInstanceCreateTask(ctx context.Context, userCred mcclient.TokenCredential, resetPassword bool, password, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(password), "password")
	params.Add(jsonutils.NewBool(resetPassword), "reset_password")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(userCred, api.DBINSTANCE_DEPLOYING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.DBInstanceDetails, error) {
	return api.DBInstanceDetails{}, nil
}

func (manager *SDBInstanceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DBInstanceDetails {
	rows := make([]api.DBInstanceDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	rdsIds := make([]string, len(rows))
	vpcIds := make([]string, len(rows))
	zone1Ids := make([]string, len(rows))
	zone2Ids := make([]string, len(rows))
	zone3Ids := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.DBInstanceDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regRows[i],
		}
		instance := objs[i].(*SDBInstance)
		rdsIds[i] = instance.Id
		vpcIds[i] = instance.VpcId
		zone1Ids[i] = instance.Zone1
		zone2Ids[i] = instance.Zone2
		zone3Ids[i] = instance.Zone3
	}

	vpcs := make(map[string]SVpc)

	err := db.FetchStandaloneObjectsByIds(VpcManager, vpcIds, &vpcs)
	if err != nil {
		log.Errorf("db.FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if vpc, ok := vpcs[vpcIds[i]]; ok {
			rows[i].Vpc = vpc.Name
			rows[i].VpcExtId = vpc.ExternalId
		}
	}

	q := SecurityGroupManager.Query()
	ownerId, queryScope, err := db.FetchCheckQueryOwnerScope(ctx, userCred, query, SecurityGroupManager, policy.PolicyActionList, true)
	if err != nil {
		log.Errorf("FetchCheckQueryOwnerScope error: %v", err)
		return rows
	}
	secgroups := SecurityGroupManager.FilterByOwner(q, ownerId, queryScope).SubQuery()
	rdssecgroups := DBInstanceSecgroupManager.Query().SubQuery()

	secQ := rdssecgroups.Query(rdssecgroups.Field("dbinstance_id"), rdssecgroups.Field("secgroup_id"), secgroups.Field("name").Label("secgroup_name")).Join(secgroups, sqlchemy.Equals(rdssecgroups.Field("secgroup_id"), secgroups.Field("id"))).Filter(sqlchemy.In(rdssecgroups.Field("dbinstance_id"), rdsIds))

	type sRdsSecgroupInfo struct {
		DBInstanceId string `json:"dbinstance_id"`
		SecgroupName string
		SecgroupId   string
	}
	rsgs := []sRdsSecgroupInfo{}
	err = secQ.All(&rsgs)
	if err != nil {
		log.Errorf("secQ.All error: %v", err)
		return rows
	}

	ret := make(map[string][]apis.StandaloneShortDesc)
	for i := range rsgs {
		rsg, ok := ret[rsgs[i].DBInstanceId]
		if !ok {
			rsg = make([]apis.StandaloneShortDesc, 0)
		}
		rsg = append(rsg, apis.StandaloneShortDesc{
			Id:   rsgs[i].SecgroupId,
			Name: rsgs[i].SecgroupName,
		})
		ret[rsgs[i].DBInstanceId] = rsg
	}

	zone1, err := db.FetchIdNameMap2(ZoneManager, zone1Ids)
	if err != nil {
		return rows
	}

	zone2, err := db.FetchIdNameMap2(ZoneManager, zone2Ids)
	if err != nil {
		return rows
	}

	zone3, err := db.FetchIdNameMap2(ZoneManager, zone3Ids)
	if err != nil {
		return rows
	}

	for i := range rows {
		rows[i].Zone1Name = zone1[zone1Ids[i]]
		rows[i].Zone2Name = zone2[zone2Ids[i]]
		rows[i].Zone3Name = zone3[zone3Ids[i]]
		rows[i].Secgroups, _ = ret[rdsIds[i]]
	}

	return rows
}

func (self *SDBInstance) GetVpc() (*SVpc, error) {
	vpc, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, err
	}
	return vpc.(*SVpc), nil
}

type sDBInstanceZone struct {
	Id   string
	Name string
}

func fetchDBInstanceZones(rdsIds []string) map[string][]sDBInstanceZone {
	instances := DBInstanceManager.Query().SubQuery()
	zones := ZoneManager.Query().SubQuery()

	result := map[string][]sDBInstanceZone{}

	for _, zone := range []string{"zone1", "zone2", "zone3"} {
		zoneInfo := []struct {
			InstanceId string
			Id         string
			Name       string
		}{}

		q := zones.Query(instances.Field("id").Label("instance_id"), zones.Field("id"), zones.Field("name"))
		q = q.Join(instances, sqlchemy.Equals(zones.Field("id"), instances.Field(zone)))
		q = q.Filter(sqlchemy.In(instances.Field("id"), rdsIds))
		q = q.Filter(sqlchemy.NOT(sqlchemy.IsNullOrEmpty(instances.Field(zone))))

		err := q.All(&zoneInfo)
		if err != nil {
			return nil
		}

		for _, _zone := range zoneInfo {
			if _, ok := result[_zone.InstanceId]; !ok {
				result[_zone.InstanceId] = []sDBInstanceZone{}
			}
			result[_zone.InstanceId] = append(result[_zone.InstanceId], sDBInstanceZone{
				Id:   fmt.Sprintf("%s_name", zone),
				Name: _zone.Name,
			})
		}
	}

	return result
}

func (self *SDBInstance) getSecgroupsByExternalIds(externalIds []string) ([]SSecurityGroup, error) {
	sq := SecurityGroupCacheManager.Query("secgroup_id").In("external_id", externalIds).Equals("manager_id", self.ManagerId)
	q := SecurityGroupManager.Query().In("id", sq.SubQuery())
	secgroups := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, q, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return secgroups, nil
}

func (self *SDBInstance) GetSecgroups() ([]SSecurityGroup, error) {
	sq := DBInstanceSecgroupManager.Query("secgroup_id").Equals("dbinstance_id", self.Id).SubQuery()
	q := SecurityGroupManager.Query().In("id", sq)
	secgroups := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, q, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return secgroups, nil
}

func (self *SDBInstance) GetDBInstanceSecgroups() ([]SDBInstanceSecgroup, error) {
	q := DBInstanceSecgroupManager.Query().Equals("dbinstance_id", self.Id)
	secgroups := []SDBInstanceSecgroup{}
	err := db.FetchModelObjects(DBInstanceSecgroupManager, q, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return secgroups, nil
}

func (self *SDBInstance) RevokeSecgroup(ctx context.Context, userCred mcclient.TokenCredential, id string) error {
	secgroups, err := self.GetDBInstanceSecgroups()
	if err != nil {
		return errors.Wrapf(err, "GetDBInstanceSecgroups")
	}
	for i := range secgroups {
		if secgroups[i].SecgroupId == id {
			err = secgroups[i].Detach(ctx, userCred)
			if err != nil {
				return errors.Wrapf(err, "secgroups.Detach %d", secgroups[i].RowId)
			}
		}
	}
	return nil
}

func (self *SDBInstance) AssignSecgroup(ctx context.Context, userCred mcclient.TokenCredential, id string) error {
	secgroups, err := self.GetDBInstanceSecgroups()
	if err != nil {
		return errors.Wrapf(err, "GetDBInstanceSecgroups")
	}
	for i := range secgroups {
		if secgroups[i].SecgroupId == id {
			return fmt.Errorf("secgroup %s already assign rds %s(%s)", id, self.Name, self.Id)
		}
	}
	return self.assignSecgroup(ctx, userCred, id)
}

func (self *SDBInstance) assignSecgroup(ctx context.Context, userCred mcclient.TokenCredential, id string) error {
	ds := &SDBInstanceSecgroup{}
	ds.DBInstanceId = self.Id
	ds.SecgroupId = id
	ds.SetModelManager(DBInstanceSecgroupManager, ds)
	return DBInstanceSecgroupManager.TableSpec().Insert(ctx, ds)
}

func (self *SDBInstance) GetMasterInstance() (*SDBInstance, error) {
	instance, err := DBInstanceManager.FetchById(self.MasterInstanceId)
	if err != nil {
		return nil, err
	}
	return instance.(*SDBInstance), nil
}

func (self *SDBInstance) GetIDBInstance() (cloudprovider.ICloudDBInstance, error) {
	iregion, err := self.GetIRegion()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIRegion")
	}
	iRds, err := iregion.GetIDBInstanceById(self.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIDBInstanceById(%s)", self.ExternalId)
	}
	return iRds, nil
}

func (self *SDBInstance) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	backups, err := self.GetDBInstanceBackups()
	if err != nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("failed get backups: %v", err))
	}
	for i := range backups {
		_, err := backups[i].PerformChangeOwner(ctx, userCred, query, input)
		if err != nil {
			return nil, err
		}
	}
	return self.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, input)
}

func (self *SDBInstance) AllowPerformRecovery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "recovery")
}

func (self *SDBInstance) PerformRecovery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SDBInstanceRecoveryConfigInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do recovery dbinstance in status %s required status %s", self.Status, api.DBINSTANCE_RUNNING)
	}

	_backup, err := DBInstanceBackupManager.FetchByIdOrName(userCred, input.DBInstancebackupId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("dbinstancebackup", input.DBInstancebackupId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	input.DBInstancebackupId = _backup.GetId()

	databases, err := self.GetDBInstanceDatabases()
	if err != nil {
		return nil, err
	}

	dbDatabases := []string{}
	for _, database := range databases {
		dbDatabases = append(dbDatabases, database.Name)
	}

	backup := _backup.(*SDBInstanceBackup)
	for src, dest := range input.Databases {
		if len(dest) == 0 {
			dest = src
		}
		if strings.Index(backup.DBNames, src) < 0 {
			return nil, httperrors.NewInputParameterError("backup %s(%s) not contain database %s", backup.Name, backup.Id, src)
		}

		if utils.IsInStringArray(dest, dbDatabases) {
			return nil, httperrors.NewConflictError("conflict database %s for instance %s(%s)", dest, self.Name, self.Id)
		}
		input.Databases[src] = dest
	}

	if backup.ManagerId != self.ManagerId {
		return nil, httperrors.NewInputParameterError("back and instance not in same cloudaccount")
	}

	if backup.CloudregionId != self.CloudregionId {
		return nil, httperrors.NewInputParameterError("backup and instance not in same cloudregion")
	}

	if len(backup.Engine) > 0 && backup.Engine != self.Engine {
		return nil, httperrors.NewInputParameterError("can not recover data from diff rds engine")
	}

	driver, err := self.GetRegionDriver()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	err = driver.ValidateDBInstanceRecovery(ctx, userCred, self, backup, input)
	if err != nil {
		return nil, err
	}

	return nil, self.StartDBInstanceRecoveryTask(ctx, userCred, input.JSON(input), "")
}

func (self *SDBInstance) StartDBInstanceRecoveryTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_RESTORING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceRecoveryTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SDBInstance) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("purge", jsonutils.JSONTrue)
	return nil, self.StartDBInstanceDeleteTask(ctx, userCred, params, "")
}

func (self *SDBInstance) AllowPerformReboot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "reboot")
}

func (self *SDBInstance) PerformReboot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING, api.DBINSTANCE_REBOOT_FAILED}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do reboot dbinstance in status %s", self.Status)
	}
	return nil, self.StartDBInstanceRebootTask(ctx, userCred, jsonutils.NewDict(), "")
}

//同步RDS实例状态
func (self *SDBInstance) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

func (self *SDBInstance) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("DBInstance has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "DBInstanceSyncStatusTask", "")
}

func (self *SDBInstance) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "sync")
}

func (self *SDBInstance) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("DBInstance has %d task active, can't sync status", count)
	}

	return nil, self.StartDBInstanceSyncTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SDBInstance) AllowPerformSyncStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "sync-status")
}

func (self *SDBInstance) PerformSyncStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("DBInstance has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "DBInstanceSyncStatusTask", "")
}

func (self *SDBInstance) AllowPerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "renew")
}

func (self *SDBInstance) PerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do renew dbinstance in status %s required status %s", self.Status, api.DBINSTANCE_RUNNING)
	}

	durationStr := jsonutils.GetAnyString(data, []string{"duration"})
	if len(durationStr) == 0 {
		return nil, httperrors.NewInputParameterError("missong duration")
	}

	bc, err := billing.ParseBillingCycle(durationStr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid duration %s: %s", durationStr, err)
	}

	if !self.GetRegion().GetDriver().IsSupportedBillingCycle(bc, DBInstanceManager.KeywordPlural()) {
		return nil, httperrors.NewInputParameterError("unsupported duration %s", durationStr)
	}

	return nil, self.StartDBInstanceRenewTask(ctx, userCred, durationStr, "")
}

func (self *SDBInstance) AllowPerformSetAutoRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "set-auto-renew")
}

func (self *SDBInstance) SetAutoRenew(autoRenew bool) error {
	_, err := db.Update(self, func() error {
		self.AutoRenew = autoRenew
		return nil
	})
	return err
}

// 设置自动续费
// 要求RDS状态为running
// 要求RDS计费类型为包年包月(预付费)
func (self *SDBInstance) PerformSetAutoRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DBInstanceAutoRenewInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING}) {
		return nil, httperrors.NewUnsupportOperationError("The dbinstance status need be %s, current is %s", api.DBINSTANCE_RUNNING, self.Status)
	}

	if self.BillingType != billing_api.BILLING_TYPE_PREPAID {
		return nil, httperrors.NewUnsupportOperationError("Only %s dbinstance support this operation", billing_api.BILLING_TYPE_PREPAID)
	}

	if self.AutoRenew == input.AutoRenew {
		return nil, nil
	}

	driver, err := self.GetRegionDriver()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegionDriver")
	}

	if !driver.IsSupportedDBInstanceAutoRenew() {
		err := self.SetAutoRenew(input.AutoRenew)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}

		logclient.AddSimpleActionLog(self, logclient.ACT_SET_AUTO_RENEW, jsonutils.Marshal(input), userCred, true)
		return nil, nil
	}

	return nil, self.StartSetAutoRenewTask(ctx, userCred, input.AutoRenew, "")
}

func (self *SDBInstance) StartSetAutoRenewTask(ctx context.Context, userCred mcclient.TokenCredential, autoRenew bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("auto_renew", jsonutils.NewBool(autoRenew))
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceSetAutoRenewTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DBINSTANCE_SET_AUTO_RENEW, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) AllowPerformPublicConnection(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "public-connection")
}

func (self *SDBInstance) PerformPublicConnection(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	open := jsonutils.QueryBoolean(data, "open", true)
	if open && len(self.ConnectionStr) > 0 {
		return nil, httperrors.NewInputParameterError("DBInstance has opened the outer network connection")
	}
	if !open && len(self.ConnectionStr) == 0 {
		return nil, httperrors.NewInputParameterError("The extranet connection is not open")
	}

	region := self.GetRegion()
	if region == nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("failed to found region for dbinstance %s(%s)", self.Name, self.Id))
	}

	if !region.GetDriver().IsSupportDBInstancePublicConnection() {
		return nil, httperrors.NewInputParameterError("%s not support this operation", region.Provider)
	}

	return nil, self.StartDBInstancePublicConnectionTask(ctx, userCred, "", open)
}

func (self *SDBInstance) StartDBInstancePublicConnectionTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, open bool) error {
	self.SetStatus(userCred, api.DBINSTANCE_DEPLOYING, "")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(open), "open")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstancePublicConnectionTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) AllowPerformChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "change-config")
}

func (self *SDBInstance) PerformChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SDBInstanceChangeConfigInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING}) {
		return nil, httperrors.NewInputParameterError("Cannot change config in status %s", self.Status)
	}

	if input.DiskSizeGB != 0 && input.DiskSizeGB < self.DiskSizeGB {
		return nil, httperrors.NewUnsupportOperationError("DBInstance Disk cannot be thrink")
	}

	if input.DiskSizeGB == self.DiskSizeGB && input.InstanceType == self.InstanceType {
		return nil, nil
	}

	return nil, self.StartDBInstanceChangeConfig(ctx, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), "")
}

func (self *SDBInstance) StartDBInstanceChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_CHANGE_CONFIG, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceChangeConfigTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) StartDBInstanceRenewTask(ctx context.Context, userCred mcclient.TokenCredential, duration string, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_RENEWING, "")
	params := jsonutils.NewDict()
	params.Set("duration", jsonutils.NewString(duration))
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceRenewTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) SaveRenewInfo(
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
		log.Errorf("Update error %s", err)
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, self.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SDBInstance) StartDBInstanceDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceDeleteTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) StartDBInstanceRebootTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_REBOOTING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceRebootTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) StartDBInstanceSyncTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_SYNC_CONFIG, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceSyncTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (manager *SDBInstanceManager) getDBInstancesByProviderId(providerId string) ([]SDBInstance, error) {
	instances := []SDBInstance{}
	err := fetchByManagerId(manager, providerId, &instances)
	if err != nil {
		return nil, errors.Wrapf(err, "getDBInstancesByProviderId.fetchByManagerId")
	}
	return instances, nil
}

func (self *SDBInstance) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	params := jsonutils.NewDict()
	params.Set("keep_backup", jsonutils.NewBool(jsonutils.QueryBoolean(data, "keep_backup", false)))
	return self.StartDBInstanceDeleteTask(ctx, userCred, params, "")
}

func (self *SDBInstance) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("dbinstance delete do nothing")
	return nil
}

func (self *SDBInstance) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SDBInstance) GetDBInstanceParameters() ([]SDBInstanceParameter, error) {
	params := []SDBInstanceParameter{}
	q := DBInstanceParameterManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceParameterManager, q, &params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceParameters.FetchModelObjects for instance %s", self.Id)
	}
	return params, nil
}

func (self *SDBInstance) GetDBInstanceBackup(name string) (*SDBInstanceBackup, error) {
	q := DBInstanceBackupManager.Query().Equals("dbinstance_id", self.Id)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("name"), name),
			sqlchemy.Equals(q.Field("id"), name),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, fmt.Errorf("Duplicate %d backup %s for dbinstance %s(%s)", count, name, self.Name, self.Id)
	}
	if count == 0 {
		return nil, fmt.Errorf("Failed to found backup %s for dbinstance %s(%s)", name, self.Name, self.Id)
	}
	backup := &SDBInstanceBackup{}
	backup.SetModelManager(DBInstanceBackupManager, backup)
	return backup, q.First(backup)
}

func (self *SDBInstance) GetDBInstanceDatabase(name string) (*SDBInstanceDatabase, error) {
	q := DBInstanceDatabaseManager.Query().Equals("dbinstance_id", self.Id)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("name"), name),
			sqlchemy.Equals(q.Field("id"), name),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, fmt.Errorf("Duplicate %d database %s for dbinstance %s(%s)", count, name, self.Name, self.Id)
	}
	if count == 0 {
		return nil, fmt.Errorf("Failed to found database %s for dbinstance %s(%s)", name, self.Name, self.Id)
	}
	database := &SDBInstanceDatabase{}
	database.SetModelManager(DBInstanceDatabaseManager, database)
	return database, q.First(database)
}

func (self *SDBInstance) GetDBInstanceDatabases() ([]SDBInstanceDatabase, error) {
	databases := []SDBInstanceDatabase{}
	q := DBInstanceDatabaseManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceDatabaseManager, q, &databases)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceDatabases.FetchModelObjects for instance %s", self.Id)
	}
	return databases, nil
}

func (self *SDBInstance) GetDBInstanceAccount(name string) (*SDBInstanceAccount, error) {
	q := DBInstanceAccountManager.Query().Equals("dbinstance_id", self.Id)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("name"), name),
			sqlchemy.Equals(q.Field("id"), name),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, fmt.Errorf("Duplicate %d account %s for dbinstance %s(%s)", count, name, self.Name, self.Id)
	}
	if count == 0 {
		return nil, fmt.Errorf("Failed to found account %s for dbinstance %s(%s)", name, self.Name, self.Id)
	}
	account := &SDBInstanceAccount{}
	account.SetModelManager(DBInstanceAccountManager, account)
	return account, q.First(account)
}

func (self *SDBInstance) GetDBInstanceAccounts() ([]SDBInstanceAccount, error) {
	accounts := []SDBInstanceAccount{}
	q := DBInstanceAccountManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceAccountManager, q, &accounts)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceAccounts.FetchModelObjects for instance %s", self.Id)
	}
	return accounts, nil
}

func (self *SDBInstance) GetDBInstancePrivilege(account, database string) (*SDBInstancePrivilege, error) {
	instances := DBInstanceManager.Query().SubQuery()
	accounts := DBInstanceAccountManager.Query().SubQuery()
	databases := DBInstanceDatabaseManager.Query().SubQuery()
	q := DBInstancePrivilegeManager.Query()
	q = q.Join(accounts, sqlchemy.Equals(accounts.Field("id"), q.Field("dbinstanceaccount_id"))).
		Join(databases, sqlchemy.Equals(databases.Field("id"), q.Field("dbinstancedatabase_id"))).
		Join(instances, sqlchemy.AND(sqlchemy.Equals(instances.Field("id"), accounts.Field("dbinstance_id")), sqlchemy.Equals(instances.Field("id"), databases.Field("dbinstance_id"))))
	q = q.Filter(
		sqlchemy.AND(
			sqlchemy.OR(
				sqlchemy.Equals(accounts.Field("id"), account),
				sqlchemy.Equals(accounts.Field("name"), account),
			),
			sqlchemy.OR(
				sqlchemy.Equals(databases.Field("id"), database),
				sqlchemy.Equals(databases.Field("name"), database),
			),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	if count == 0 {
		return nil, sql.ErrNoRows
	}
	privilege := &SDBInstancePrivilege{}
	privilege.SetModelManager(DBInstancePrivilegeManager, privilege)
	err = q.First(privilege)
	if err != nil {
		return nil, errors.Wrap(err, "q.First()")
	}
	return privilege, nil
}

func (self *SDBInstance) GetDBInstanceBackupByMode(mode string) ([]SDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	q := DBInstanceBackupManager.Query().Equals("dbinstance_id", self.Id)
	switch mode {
	case api.BACKUP_MODE_MANUAL, api.BACKUP_MODE_AUTOMATED:
		q = q.Equals("backup_mode", mode)
	}
	err := db.FetchModelObjects(DBInstanceBackupManager, q, &backups)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstanceBackups.FetchModelObjects")
	}
	return backups, nil

}

func (self *SDBInstance) GetDBInstanceBackups() ([]SDBInstanceBackup, error) {
	return self.GetDBInstanceBackupByMode("")
}

func (self *SDBInstance) GetDBDatabases() ([]SDBInstanceDatabase, error) {
	databases := []SDBInstanceDatabase{}
	q := DBInstanceDatabaseManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceDatabaseManager, q, &databases)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBDatabases.FetchModelObjects")
	}
	return databases, nil
}

func (self *SDBInstance) GetDBParameters() ([]SDBInstanceParameter, error) {
	parameters := []SDBInstanceParameter{}
	q := DBInstanceParameterManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceParameterManager, q, &parameters)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBParameters.FetchModelObjects")
	}
	return parameters, nil
}

func (self *SDBInstance) GetDBNetworks() ([]SDBInstanceNetwork, error) {
	q := DBInstanceNetworkManager.Query().Equals("dbinstance_id", self.Id)
	networks := []SDBInstanceNetwork{}
	err := db.FetchModelObjects(DBInstanceNetworkManager, q, &networks)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return networks, nil
}

func (manager *SDBInstanceManager) SyncDBInstanceMasterId(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, cloudDBInstances []cloudprovider.ICloudDBInstance) {
	for _, instance := range cloudDBInstances {
		masterId := instance.GetMasterInstanceId()
		if len(masterId) > 0 {
			master, err := db.FetchByExternalIdAndManagerId(manager, masterId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", provider.Id)
			})
			if err != nil {
				log.Errorf("failed to found master dbinstance by externalId: %s error: %v", masterId, err)
				continue
			}
			slave, err := db.FetchByExternalIdAndManagerId(manager, instance.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", provider.Id)
			})
			if err != nil {
				log.Errorf("failed to found local dbinstance by externalId %s error: %v", instance.GetGlobalId(), err)
				continue
			}
			localInstance := slave.(*SDBInstance)
			_, err = db.Update(localInstance, func() error {
				localInstance.MasterInstanceId = master.GetId()
				return nil
			})
			if err != nil {
				log.Errorf("failed to update dbinstance %s(%s) master instanceId error: %v", localInstance.Name, localInstance.Id, err)
			}
		}
	}
}

func (manager *SDBInstanceManager) SyncDBInstances(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, region *SCloudregion, cloudDBInstances []cloudprovider.ICloudDBInstance) ([]SDBInstance, []cloudprovider.ICloudDBInstance, compare.SyncResult) {
	lockman.LockRawObject(ctx, "dbinstances", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "dbinstances", fmt.Sprintf("%s-%s", provider.Id, region.Id))

	localDBInstances := []SDBInstance{}
	remoteDBInstances := []cloudprovider.ICloudDBInstance{}
	syncResult := compare.SyncResult{}

	dbInstances, err := region.GetDBInstances(provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := range dbInstances {
		if taskman.TaskManager.IsInTask(&dbInstances[i]) {
			syncResult.Error(fmt.Errorf("dbInstance %s(%s)in task", dbInstances[i].Name, dbInstances[i].Id))
			return nil, nil, syncResult
		}
	}

	removed := make([]SDBInstance, 0)
	commondb := make([]SDBInstance, 0)
	commonext := make([]cloudprovider.ICloudDBInstance, 0)
	added := make([]cloudprovider.ICloudDBInstance, 0)
	if err := compare.CompareSets(dbInstances, cloudDBInstances, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudDBInstance(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudDBInstance(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncVirtualResourceMetadata(ctx, userCred, &commondb[i], commonext[i])
		localDBInstances = append(localDBInstances, commondb[i])
		remoteDBInstances = append(remoteDBInstances, commonext[i])
		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		instance, err := manager.newFromCloudDBInstance(ctx, userCred, syncOwnerId, provider, region, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncVirtualResourceMetadata(ctx, userCred, instance, added[i])
		localDBInstances = append(localDBInstances, *instance)
		remoteDBInstances = append(remoteDBInstances, added[i])
		syncResult.Add()
	}
	return localDBInstances, remoteDBInstances, syncResult
}

func (self *SDBInstance) syncRemoveCloudDBInstance(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.Purge(ctx, userCred)
}

func (self *SDBInstance) ValidateDeleteCondition(ctx context.Context) error {
	if self.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("DBInstance is locked, cannot delete")
	}
	return self.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SDBInstance) GetDBInstanceSkuQuery(skipZoneCheck bool) *sqlchemy.SQuery {
	q := DBInstanceSkuManager.Query().Equals("storage_type", self.StorageType).Equals("category", self.Category).
		Equals("cloudregion_id", self.CloudregionId).Equals("engine", self.Engine).Equals("engine_version", self.EngineVersion)

	if !skipZoneCheck {
		for k, v := range map[string]string{"zone1": self.Zone1, "zone2": self.Zone2, "zone3": self.Zone3} {
			if len(v) > 0 {
				q = q.Equals(k, v)
			}
		}
	}

	if len(self.InstanceType) > 0 {
		q = q.Equals("name", self.InstanceType)
	} else {
		q = q.Equals("vcpu_count", self.VcpuCount).Equals("vmem_size_mb", self.VmemSizeMb)
	}
	return q
}

func (self *SDBInstance) GetAvailableDBInstanceSkus(skipZoneCheck bool) ([]SDBInstanceSku, error) {
	skus := []SDBInstanceSku{}
	q := self.GetDBInstanceSkuQuery(skipZoneCheck).Equals("status", api.DBINSTANCE_SKU_AVAILABLE)
	err := db.FetchModelObjects(DBInstanceSkuManager, q, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil

}

func (self *SDBInstance) GetDBInstanceSkus(skipZoneCheck bool) ([]SDBInstanceSku, error) {
	skus := []SDBInstanceSku{}
	q := self.GetDBInstanceSkuQuery(skipZoneCheck)
	err := db.FetchModelObjects(DBInstanceSkuManager, q, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil
}

func (self *SDBInstance) GetAvailableInstanceTypes() ([]cloudprovider.SInstanceType, error) {
	instanceTypes := []cloudprovider.SInstanceType{}
	skus, err := self.GetAvailableDBInstanceSkus(false)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetAvailableDBInstanceSkus")
	}

	for _, sku := range skus {
		instanceType := cloudprovider.SInstanceType{}
		instanceType.InstanceType = sku.Name
		instanceType.SZoneInfo, _ = sku.GetZoneInfo()
		instanceTypes = append(instanceTypes, instanceType)
	}
	return instanceTypes, nil
}

func (self *SDBInstance) setZoneInfo() error {
	sku := SDBInstanceSku{}
	q := self.GetDBInstanceSkuQuery(false)
	count, err := q.CountWithError()
	if err != nil {
		return errors.Wrapf(err, "q.CountWithError")
	}
	if count == 0 {
		q.DebugQuery()
		return fmt.Errorf("failed to fetch any sku for dbinstance %s(%s)", self.Name, self.Id)
	}
	if count > 1 {
		q.DebugQuery()
		return fmt.Errorf("fetch %d skus for dbinstance %s(%s)", count, self.Name, self.Id)
	}
	err = q.First(&sku)
	if err != nil {
		return errors.Wrap(err, "q.First()")
	}
	self.Zone1 = sku.Zone1
	self.Zone2 = sku.Zone2
	self.Zone3 = sku.Zone3
	return nil
}

func (self *SDBInstance) SetZoneInfo(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		return self.setZoneInfo()
	})
	return err
}

func (self *SDBInstance) SetZoneIds(extInstance cloudprovider.ICloudDBInstance) error {
	region := self.GetRegion()
	if region == nil {
		return fmt.Errorf("failed found region for dbinstance %s", self.Name)
	}
	zones, err := region.GetZones()
	if err != nil {
		return errors.Wrapf(err, "GetZones")
	}
	var setZoneId = func(input string, output *string) {
		for _, zone := range zones {
			if strings.HasSuffix(zone.ExternalId, input) {
				*output = zone.Id
				break
			}
		}
		return
	}
	zone1 := extInstance.GetZone1Id()
	if len(zone1) > 0 {
		setZoneId(zone1, &self.Zone1)
	}
	zone2 := extInstance.GetZone2Id()
	if len(zone2) > 0 {
		setZoneId(zone2, &self.Zone2)
	}
	zone3 := extInstance.GetZone3Id()
	if len(zone3) > 0 {
		setZoneId(zone3, &self.Zone3)
	}
	return nil
}

func (self *SDBInstance) SyncAllWithCloudDBInstance(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extInstance cloudprovider.ICloudDBInstance) error {
	err := self.SyncWithCloudDBInstance(ctx, userCred, provider, extInstance)
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudDBInstance")
	}
	syncDBInstanceResource(ctx, userCred, SSyncResultSet{}, self, extInstance)
	return nil
}

func (self *SDBInstance) SyncWithCloudDBInstance(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extInstance cloudprovider.ICloudDBInstance) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.ExternalId = extInstance.GetGlobalId()
		self.Engine = extInstance.GetEngine()
		self.EngineVersion = extInstance.GetEngineVersion()
		self.InstanceType = extInstance.GetInstanceType()
		self.VcpuCount = extInstance.GetVcpuCount()
		self.VmemSizeMb = extInstance.GetVmemSizeMB()
		self.DiskSizeGB = extInstance.GetDiskSizeGB()
		self.StorageType = extInstance.GetStorageType()
		self.Status = extInstance.GetStatus()
		self.Port = extInstance.GetPort()

		self.ConnectionStr = extInstance.GetConnectionStr()
		self.InternalConnectionStr = extInstance.GetInternalConnectionStr()

		self.MaintainTime = extInstance.GetMaintainTime()
		self.SetZoneIds(extInstance)

		if createdAt := extInstance.GetCreatedAt(); !createdAt.IsZero() {
			self.CreatedAt = createdAt
		}

		if expiredAt := extInstance.GetExpiredAt(); !expiredAt.IsZero() {
			self.ExpiredAt = expiredAt
		}

		if len(self.VpcId) == 0 {
			if vpcId := extInstance.GetIVpcId(); len(vpcId) > 0 {
				vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
					return q.Equals("manager_id", provider.Id)
				})
				if err != nil {
					log.Errorf("FetchVpcId(%s) error: %v", vpcId, err)
				} else {
					self.VpcId = vpc.GetId()
				}
			}
		}
		if len(self.VpcId) == 0 {
			region := self.GetRegion()
			vpc, err := VpcManager.GetOrCreateVpcForClassicNetwork(ctx, provider, region)
			if err != nil {
				log.Errorf("failed to create classic vpc for region %s error: %v", region.Name, err)
			} else {
				self.VpcId = vpc.GetId()
			}
		}

		factory, err := provider.GetProviderFactory()
		if err != nil {
			return errors.Wrap(err, "SyncWithCloudDBInstance.GetProviderFactory")
		}

		if factory.IsSupportPrepaidResources() && !extInstance.GetExpiredAt().IsZero() {
			self.BillingType = extInstance.GetBillingType()
			if expired := extInstance.GetExpiredAt(); !expired.IsZero() {
				self.ExpiredAt = expired
			}
			self.AutoRenew = extInstance.IsAutoRenew()
		}

		return nil
	})
	if err != nil {
		return err
	}
	syncVirtualResourceMetadata(ctx, userCred, self, extInstance)
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SDBInstance) GetSlaveDBInstances() ([]SDBInstance, error) {
	dbinstances := []SDBInstance{}
	q := DBInstanceManager.Query().Equals("master_instance_id", self.Id)
	return dbinstances, db.FetchModelObjects(DBInstanceManager, q, &dbinstances)
}

func (manager *SDBInstanceManager) newFromCloudDBInstance(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, provider *SCloudprovider, region *SCloudregion, extInstance cloudprovider.ICloudDBInstance) (*SDBInstance, error) {

	instance := SDBInstance{}
	instance.SetModelManager(manager, &instance)

	instance.ExternalId = extInstance.GetGlobalId()
	instance.CloudregionId = region.Id
	instance.ManagerId = provider.Id
	instance.IsEmulated = extInstance.IsEmulated()
	instance.Status = extInstance.GetStatus()
	instance.Port = extInstance.GetPort()

	instance.Engine = extInstance.GetEngine()
	instance.EngineVersion = extInstance.GetEngineVersion()
	instance.InstanceType = extInstance.GetInstanceType()
	instance.Category = extInstance.GetCategory()
	instance.VcpuCount = extInstance.GetVcpuCount()
	instance.VmemSizeMb = extInstance.GetVmemSizeMB()
	instance.DiskSizeGB = extInstance.GetDiskSizeGB()
	instance.ConnectionStr = extInstance.GetConnectionStr()
	instance.StorageType = extInstance.GetStorageType()
	instance.InternalConnectionStr = extInstance.GetInternalConnectionStr()

	instance.MaintainTime = extInstance.GetMaintainTime()
	instance.SetZoneIds(extInstance)

	if vpcId := extInstance.GetIVpcId(); len(vpcId) > 0 {
		vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", provider.Id)
		})
		if err != nil {
			log.Errorf("FetchVpcId(%s) error: %v", vpcId, err)
		} else {
			instance.VpcId = vpc.GetId()
		}
	}
	if len(instance.VpcId) == 0 {
		vpc, err := VpcManager.GetOrCreateVpcForClassicNetwork(ctx, provider, region)
		if err != nil {
			log.Errorf("failed to create classic vpc for region %s error: %v", region.Name, err)
		} else {
			instance.VpcId = vpc.GetId()
		}
	}

	if createdAt := extInstance.GetCreatedAt(); !createdAt.IsZero() {
		instance.CreatedAt = createdAt
	}

	factory, err := provider.GetProviderFactory()
	if err != nil {
		return nil, errors.Wrap(err, "newFromCloudDBInstance.GetProviderFactory")
	}

	if factory.IsSupportPrepaidResources() {
		instance.BillingType = extInstance.GetBillingType()
		if expired := extInstance.GetExpiredAt(); !expired.IsZero() {
			instance.ExpiredAt = expired
		}
		instance.AutoRenew = extInstance.IsAutoRenew()
	}

	err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		instance.Name, err = db.GenerateName(ctx, manager, ownerId, extInstance.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}
		return manager.TableSpec().Insert(ctx, &instance)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudDBInstance.Insert")
	}

	SyncCloudProject(userCred, &instance, ownerId, extInstance, provider.Id)

	db.OpsLog.LogEvent(&instance, db.ACT_CREATE, instance.GetShortDesc(ctx), userCred)

	return &instance, nil
}

type SRdsCountStat struct {
	TotalRdsCount  int
	TotalCpuCount  int
	TotalMemSizeMb int
}

func (man *SDBInstanceManager) TotalCount(
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	providers []string, brands []string, cloudEnv string,
) (SRdsCountStat, error) {
	sq := man.Query().SubQuery()
	q := sq.Query(sqlchemy.COUNT("total_rds_count"),
		sqlchemy.SUM("total_cpu_count", sq.Field("vcpu_count")),
		sqlchemy.SUM("total_mem_size_mb", sq.Field("vmem_size_mb")))

	q = scopeOwnerIdFilter(q, scope, ownerId)
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	q = RangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"), nil, nil)

	stat := SRdsCountStat{}
	row := q.Row()
	err := q.Row2Struct(row, &stat)
	return stat, err
}

func (dbinstance *SDBInstance) GetQuotaKeys() quotas.IQuotaKeys {
	return fetchRegionalQuotaKeys(
		rbacutils.ScopeProject,
		dbinstance.GetOwnerId(),
		dbinstance.GetRegion(),
		dbinstance.GetCloudprovider(),
	)
}

func (dbinstance *SDBInstance) GetUsages() []db.IUsage {
	if dbinstance.PendingDeleted || dbinstance.Deleted {
		return nil
	}
	usage := SRegionQuota{Rds: 1}
	keys := dbinstance.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (dbinstance *SDBInstance) GetIRegion() (cloudprovider.ICloudRegion, error) {
	region := dbinstance.GetRegion()
	if region == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid cloudregion")
	}
	provider, err := dbinstance.GetDriver()
	if err != nil {
		return nil, errors.Wrap(err, "dbinstance.GetDriver")
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (manager *SDBInstanceManager) ListItemExportKeys(ctx context.Context,
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

func (manager *SDBInstanceManager) getExpiredPostpaids() []SDBInstance {
	q := ListExpiredPostpaidResources(manager.Query(), options.Options.ExpiredPrepaidMaxCleanBatchSize)
	q = q.IsFalse("pending_deleted")

	dbs := make([]SDBInstance, 0)
	err := db.FetchModelObjects(DBInstanceManager, q, &dbs)
	if err != nil {
		log.Errorf("fetch dbinstances error %s", err)
		return nil
	}

	return dbs
}

func (cache *SDBInstance) SetDisableDelete(userCred mcclient.TokenCredential, val bool) error {
	diff, err := db.Update(cache, func() error {
		if val {
			cache.DisableDelete = tristate.True
		} else {
			cache.DisableDelete = tristate.False
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(cache, db.ACT_UPDATE, diff, userCred)
	logclient.AddSimpleActionLog(cache, logclient.ACT_UPDATE, diff, userCred, true)
	return err
}

func (self *SDBInstance) doExternalSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	provider := self.GetCloudprovider()
	if provider != nil {
		return fmt.Errorf("no cloud provider???")
	}

	iregion, err := self.GetIRegion()
	if err != nil || iregion == nil {
		return fmt.Errorf("no cloud region??? %s", err)
	}

	idbs, err := iregion.GetIDBInstanceById(self.ExternalId)
	if err != nil {
		return err
	}
	return self.SyncWithCloudDBInstance(ctx, userCred, provider, idbs)
}

func (manager *SDBInstanceManager) DeleteExpiredPostpaids(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	dbs := manager.getExpiredPostpaids()
	if dbs == nil {
		return
	}
	for i := 0; i < len(dbs); i += 1 {
		if len(dbs[i].ExternalId) > 0 {
			err := dbs[i].doExternalSync(ctx, userCred)
			if err == nil && dbs[i].IsValidPostPaid() {
				continue
			}
		}
		dbs[i].SetDisableDelete(userCred, false)
		dbs[i].StartDBInstanceDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
	}
}

func (self *SDBInstance) AllowPerformPostpaidExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "postpaid-expire")
}

func (self *SDBInstance) PerformPostpaidExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PostpaidExpireInput) (jsonutils.JSONObject, error) {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return nil, httperrors.NewBadRequestError("dbinstance billing type is %s", self.BillingType)
	}

	bc, err := ParseBillingCycleInput(&self.SBillingResourceBase, input)
	if err != nil {
		return nil, err
	}

	err = self.SaveRenewInfo(ctx, userCred, bc, nil, billing_api.BILLING_TYPE_POSTPAID)
	return nil, err
}

func (self *SDBInstance) AllowPerformCancelExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "cancel-expire")
}

func (self *SDBInstance) PerformCancelExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := self.CancelExpireTime(ctx, userCred); err != nil {
		return nil, err
	}

	return nil, nil
}

func (self *SDBInstance) CancelExpireTime(ctx context.Context, userCred mcclient.TokenCredential) error {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return httperrors.NewBadRequestError("dbinstance billing type %s not support cancel expire", self.BillingType)
	}

	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set expired_at = NULL and billing_cycle = NULL where id = ?",
			DBInstanceManager.TableSpec().Name(),
		), self.Id,
	)
	if err != nil {
		return errors.Wrap(err, "dbinstance cancel expire time")
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, "dbinstance cancel expire time", userCred)
	return nil
}

func (self *SDBInstance) AllowPerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "remote-update")
}

func (self *SDBInstance) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DBInstanceRemoteUpdateInput) (jsonutils.JSONObject, error) {
	err := self.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRemoteUpdateTask")
	}
	return nil, nil
}

func (self *SDBInstance) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	if replaceTags {
		data.Add(jsonutils.JSONTrue, "replace_tags")
	}
	if task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return errors.Wrap(err, "Start ElasticcacheRemoteUpdateTask")
	} else {
		self.SetStatus(userCred, api.DBINSTANCE_UPDATE_TAGS, "StartRemoteUpdateTask")
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDBInstance) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(self.ExternalId) == 0 {
		return
	}
	err := self.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}

func (self *SDBInstance) AllowPerformSetSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "set-secgroup")
}

func (self *SDBInstance) PerformSetSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DBInstanceSetSecgroupInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DBINSTANCE_RUNNING {
		return nil, httperrors.NewInvalidStatusError("this operation requires rds state to be %s", api.DBINSTANCE_RUNNING)
	}
	if len(input.SecgroupIds) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroup_ids")
	}
	for i := range input.SecgroupIds {
		_, err := validators.ValidateModel(userCred, SecurityGroupManager, &input.SecgroupIds[i])
		if err != nil {
			return nil, err
		}
	}
	driver, err := self.GetRegionDriver()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetRegionDriver"))
	}
	max := driver.GetRdsSupportSecgroupCount()
	if len(input.SecgroupIds) > max {
		return nil, httperrors.NewUnsupportOperationError("%s supported secgroup count is %d", driver.GetProvider(), max)
	}

	secgroups, err := self.GetSecgroups()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetSecgroups"))
	}
	secMaps := map[string]bool{}
	for i := range secgroups {
		if !utils.IsInStringArray(secgroups[i].Id, input.SecgroupIds) {
			err := self.RevokeSecgroup(ctx, userCred, secgroups[i].Id)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrapf(err, "RevokeSecgroup(%s)", secgroups[i].Id))
			}
		}
		secMaps[secgroups[i].Id] = true
	}
	for _, id := range input.SecgroupIds {
		if _, ok := secMaps[id]; !ok {
			err = self.AssignSecgroup(ctx, userCred, id)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrapf(err, "AssignSecgroup(%s)", id))
			}
		}
	}

	return nil, self.StartSyncSecgroupsTask(ctx, userCred, "")
}

func (self *SDBInstance) StartSyncSecgroupsTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceSyncSecgroupsTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DBINSTANCE_DEPLOYING, "sync secgroups")
	task.ScheduleRun(nil)
	return nil
}
