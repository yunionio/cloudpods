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

package regiondrivers

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/choices"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SAliyunRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SAliyunRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SAliyunRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ALIYUN
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error) {
	if len(input.LoadbalancerSpec) == 0 {
		input.LoadbalancerSpec = api.LB_ALIYUN_SPEC_SHAREABLE
	}
	if !utils.IsInStringArray(input.LoadbalancerSpec, api.LB_ALIYUN_SPECS) {
		return nil, httperrors.NewInputParameterError("invalid loadbalancer_spec %s", input.LoadbalancerSpec)
	}

	if input.ChargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		if input.EgressMbps < 1 || input.EgressMbps > 5000 {
			return nil, httperrors.NewInputParameterError("egress_mbps shoud be 1-5000 mbps")
		}
	}

	if input.AddressType == api.LB_ADDR_TYPE_INTRANET && input.ChargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		return nil, httperrors.NewUnsupportOperationError("intranet loadbalancer not support bandwidth charge type")
	}

	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, ownerId, input)
}

func (self *SAliyunRegionDriver) GetBackendStatusForAdd() []string {
	return []string{api.VM_RUNNING}
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, input *api.LoadbalancerBackendGroupCreateInput) (*api.LoadbalancerBackendGroupCreateInput, error) {
	switch input.Type {
	case "", api.LB_BACKENDGROUP_TYPE_NORMAL:
		break
	case api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
		if len(input.Backends) != 2 {
			return nil, httperrors.NewInputParameterError("master slave backendgorup must contain two backend")
		}
	default:
		return nil, httperrors.NewInputParameterError("Unsupport backendgorup type %s", input.Type)
	}
	for _, backend := range input.Backends {
		if len(backend.ExternalId) == 0 {
			return nil, httperrors.NewInputParameterError("invalid guest %s", backend.Name)
		}
		if backend.Weight < 0 || backend.Weight > 100 {
			return nil, httperrors.NewInputParameterError("Aliyun instance weight must be in the range of 0 ~ 100")
		}
	}
	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup,
	input *api.LoadbalancerBackendCreateInput) (*api.LoadbalancerBackendCreateInput, error) {
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerBackendData(ctx, userCred, lb, lbbg, input)
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, input *api.LoadbalancerBackendUpdateInput) (*api.LoadbalancerBackendUpdateInput, error) {
	switch lbbg.Type {
	case api.LB_BACKENDGROUP_TYPE_DEFAULT:
		if input.Port != nil {
			return nil, httperrors.NewInputParameterError("%s backend group not support change port", lbbg.Type)
		}
	case api.LB_BACKENDGROUP_TYPE_NORMAL:
		return input, nil
	case api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
		if input.Port != nil || input.Weight != nil {
			return input, httperrors.NewInputParameterError("%s backend group not support change port or weight", lbbg.Type)
		}
	default:
		return nil, httperrors.NewInputParameterError("Unknown backend group type %s", lbbg.Type)
	}
	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerRuleCreateInput) (*api.LoadbalancerListenerRuleCreateInput, error) {
	return input, nil
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LoadbalancerListenerRuleUpdateInput) (*api.LoadbalancerListenerRuleUpdateInput, error) {
	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerCreateInput,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup) (*api.LoadbalancerListenerCreateInput, error) {
	if input.ClientReqeustTimeout < 1 || input.ClientReqeustTimeout > 600 {
		input.ClientReqeustTimeout = 10
	}
	if input.ClientIdleTimeout < 1 || input.ClientIdleTimeout > 600 {
		input.ClientIdleTimeout = 90
	}
	if input.BackendConnectTimeout < 1 || input.BackendConnectTimeout > 180 {
		input.BackendConnectTimeout = 5
	}
	if input.BackendIdleTimeout < 1 || input.BackendIdleTimeout > 600 {
		input.BackendIdleTimeout = 90
	}

	if len(input.HealthCheckDomain) > 80 {
		return nil, httperrors.NewInputParameterError("health_check_domain must be in the range of 1 ~ 80")
	}
	return input, nil
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput) (*api.LoadbalancerListenerUpdateInput, error) {
	lb, err := lblis.GetLoadbalancer()
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancer")
	}
	if input.Scheduler != nil && utils.IsInStringArray(*input.Scheduler, []string{api.LB_SCHEDULER_SCH, api.LB_SCHEDULER_TCH, api.LB_SCHEDULER_QCH}) {
		if len(lb.LoadbalancerSpec) == 0 {
			return nil, httperrors.NewInputParameterError("The specified Scheduler %v is invalid for performance sharing loadbalancer", input.Scheduler)
		}
		region, err := lb.GetRegion()
		if err != nil {
			return nil, errors.Wrapf(err, "GetRegion")
		}
		supportRegions := []string{}
		for region := range map[string]string{
			"ap-northeast-1":   "东京",
			"ap-southeast-2":   "悉尼",
			"ap-southeast-3":   "吉隆坡",
			"ap-southeast-5":   "雅加达",
			"eu-frankfurt":     "法兰克福",
			"na-siliconvalley": "硅谷",
			"us-east-1":        "弗吉利亚",
			"me-east-1":        "迪拜",
			"cn-huhehaote":     "呼和浩特",
		} {
			supportRegions = append(supportRegions, "Aliyun/"+region)
		}
		if !utils.IsInStringArray(region.ExternalId, supportRegions) {
			return nil, httperrors.NewUnsupportOperationError("cloudregion %s(%s) not support %v scheduler", region.Name, region.Id, input.Scheduler)
		}
	}

	if input.HealthCheckDomain != nil && len(*input.HealthCheckDomain) > 80 {
		return nil, httperrors.NewInputParameterError("health_check_domain must be in the range of 1 ~ 80")
	}

	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	if strings.HasPrefix(input.Name, "auto") || strings.HasPrefix(input.Name, "http://") || strings.HasPrefix(input.Name, "https://") {
		return httperrors.NewBadRequestError(
			"Snapshot for %s name can't start with auto, http:// or https://", self.GetProvider())
	}
	return nil
}

func (self *SAliyunRegionDriver) IsSecurityGroupBelongVpc() bool {
	return true
}

func (self *SAliyunRegionDriver) ValidateDBInstanceRecovery(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, input api.SDBInstanceRecoveryConfigInput) error {
	if !utils.IsInStringArray(instance.Engine, []string{api.DBINSTANCE_TYPE_MYSQL, api.DBINSTANCE_TYPE_SQLSERVER}) {
		return httperrors.NewNotSupportedError("Aliyun %s not support recovery", instance.Engine)
	}
	if instance.Engine == api.DBINSTANCE_TYPE_MYSQL {
		if backup.DBInstanceId != instance.Id {
			return httperrors.NewUnsupportOperationError("Aliyun %s only support recover from it self backups", instance.Engine)
		}
		if !((utils.IsInStringArray(instance.EngineVersion, []string{"8.0", "5.7"}) &&
			instance.StorageType == api.ALIYUN_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD &&
			instance.Category == api.ALIYUN_DBINSTANCE_CATEGORY_HA) || (instance.EngineVersion == "5.6" && instance.Category == api.ALIYUN_DBINSTANCE_CATEGORY_HA)) {
			return httperrors.NewUnsupportOperationError("Aliyun %s only 8.0 and 5.7 high_availability local_ssd or 5.6 high_availability support recovery from it self backups", instance.Engine)
		}
	}
	if len(input.Databases) == 0 {
		return httperrors.NewMissingParameterError("databases")
	}
	return nil
}

func (self *SAliyunRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (api.DBInstanceCreateInput, error) {
	if input.BillingType == billing_api.BILLING_TYPE_PREPAID && len(input.MasterInstanceId) > 0 {
		return input, httperrors.NewInputParameterError("slave dbinstance not support prepaid billing type")
	}

	if network != nil {
		wire, _ := network.GetWire()
		if wire == nil {
			return input, httperrors.NewGeneralError(fmt.Errorf("failed to found wire for network %s(%s)", network.Name, network.Id))
		}
		zone, _ := wire.GetZone()
		if zone == nil {
			return input, httperrors.NewGeneralError(fmt.Errorf("failed to found zone for wire %s(%s)", wire.Name, wire.Id))
		}

		match := false
		for _, sku := range skus {
			if utils.IsInStringArray(zone.Id, []string{sku.Zone1, sku.Zone2, sku.Zone3}) {
				match = true
				break
			}
		}
		if !match {
			return input, httperrors.NewInputParameterError("failed to match any skus in the network %s(%s) zone %s(%s)", network.Name, network.Id, zone.Name, zone.Id)
		}
	}

	var master *models.SDBInstance
	var slaves []models.SDBInstance
	var err error
	if len(input.MasterInstanceId) > 0 {
		_master, _ := models.DBInstanceManager.FetchById(input.MasterInstanceId)
		master = _master.(*models.SDBInstance)
		slaves, err = master.GetSlaveDBInstances()
		if err != nil {
			return input, httperrors.NewGeneralError(err)
		}

		switch master.Engine {
		case api.DBINSTANCE_TYPE_MYSQL:
			switch master.EngineVersion {
			case "5.6":
				break
			case "5.7", "8.0":
				if master.Category != api.ALIYUN_DBINSTANCE_CATEGORY_HA {
					return input, httperrors.NewInputParameterError("Not support create readonly dbinstance for MySQL %s %s", master.EngineVersion, master.Category)
				}
				if master.StorageType != api.ALIYUN_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD {
					return input, httperrors.NewInputParameterError("Not support create readonly dbinstance for MySQL %s %s with storage type %s, only support %s", master.EngineVersion, master.Category, master.StorageType, api.ALIYUN_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD)
				}
			default:
				return input, httperrors.NewInputParameterError("Not support create readonly dbinstance for MySQL %s", master.EngineVersion)
			}
		case api.DBINSTANCE_TYPE_SQLSERVER:
			if master.Category != api.ALIYUN_DBINSTANCE_CATEGORY_ALWAYSON || master.EngineVersion != "2017_ent" {
				return input, httperrors.NewInputParameterError("SQL Server only support create readonly dbinstance for 2017_ent")
			}
			if len(slaves) >= 7 {
				return input, httperrors.NewInputParameterError("SQL Server cannot have more than seven read-only dbinstances")
			}
		default:
			return input, httperrors.NewInputParameterError("Not support create readonly dbinstance with master dbinstance engine %s", master.Engine)
		}
	}

	switch input.Engine {
	case api.DBINSTANCE_TYPE_MYSQL:
		if input.VmemSizeMb/1024 >= 64 && len(slaves) >= 10 {
			return input, httperrors.NewInputParameterError("Master dbinstance memory ≥64GB, up to 10 read-only instances are allowed to be created")
		} else if input.VmemSizeMb/1024 < 64 && len(slaves) >= 5 {
			return input, httperrors.NewInputParameterError("Master dbinstance memory <64GB, up to 5 read-only instances are allowed to be created")
		}
	case api.DBINSTANCE_TYPE_SQLSERVER:
		if input.Category == api.ALIYUN_DBINSTANCE_CATEGORY_ALWAYSON {
			vpc, _ := network.GetVpc()
			count, err := vpc.GetNetworkCount()
			if err != nil {
				return input, httperrors.NewGeneralError(err)
			}
			if count < 2 {
				return input, httperrors.NewInputParameterError("At least two networks are required under vpc %s(%s) with aliyun %s(%s)", vpc.Name, vpc.Id, input.Engine, input.Category)
			}
		}
	}

	if len(input.Name) > 0 {
		if strings.HasPrefix(input.Description, "http://") || strings.HasPrefix(input.Description, "https://") {
			return input, httperrors.NewInputParameterError("Description can not start with http:// or https://")
		}
	}

	return input, nil
}

func (self *SAliyunRegionDriver) IsSupportedBillingCycle(bc billing.SBillingCycle, resource string) bool {
	switch resource {
	case models.DBInstanceManager.KeywordPlural(),
		models.ElasticcacheManager.KeywordPlural(),
		models.NatGatewayManager.KeywordPlural(),
		models.FileSystemManager.KeywordPlural():
		years := bc.GetYears()
		months := bc.GetMonths()
		if (years >= 1 && years <= 3) || (months >= 1 && months <= 9) {
			return true
		}
	}
	return false
}

func (self *SAliyunRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceAccountCreateInput) (api.DBInstanceAccountCreateInput, error) {
	if len(input.Name) < 2 || len(input.Name) > 16 {
		return input, httperrors.NewInputParameterError("Aliyun DBInstance account name length shoud be 2~16 characters")
	}

	DENY_KEY := map[string][]string{
		api.DBINSTANCE_TYPE_MYSQL:     api.ALIYUN_MYSQL_DENY_KEYWORKD,
		api.DBINSTANCE_TYPE_SQLSERVER: api.ALIYUN_SQL_SERVER_DENY_KEYWORD,
	}

	if keys, ok := DENY_KEY[instance.Engine]; ok && utils.IsInStringArray(input.Name, keys) {
		return input, httperrors.NewInputParameterError("%s is reserved for aliyun %s, please use another", input.Name, instance.Engine)
	}

	for i, s := range input.Name {
		if !unicode.IsLetter(s) && !unicode.IsDigit(s) && s != '_' {
			return input, httperrors.NewInputParameterError("invalid character %s for account name", string(s))
		}
		if s == '_' && (i == 0 || i == len(input.Name)) {
			return input, httperrors.NewInputParameterError("account name can not start or end with _")
		}
	}

	for _, privilege := range input.Privileges {
		err := self.ValidateDBInstanceAccountPrivilege(ctx, userCred, instance, input.Name, privilege.Privilege)
		if err != nil {
			return input, err
		}
	}

	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceDatabaseCreateInput) (api.DBInstanceDatabaseCreateInput, error) {
	if len(input.CharacterSet) == 0 {
		return input, httperrors.NewMissingParameterError("character_set")
	}

	for _, account := range input.Accounts {
		err := self.ValidateDBInstanceAccountPrivilege(ctx, userCred, instance, account.Account, account.Privilege)
		if err != nil {
			return input, err
		}
	}

	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceBackupCreateInput) (api.DBInstanceBackupCreateInput, error) {
	return input, nil
}

func (self *SAliyunRegionDriver) ValidateDBInstanceAccountPrivilege(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string, privilege string) error {
	switch privilege {
	case api.DATABASE_PRIVILEGE_RW:
	case api.DATABASE_PRIVILEGE_R:
	case api.DATABASE_PRIVILEGE_DDL, api.DATABASE_PRIVILEGE_DML:
		if instance.Engine != api.DBINSTANCE_TYPE_MYSQL && instance.Engine != api.DBINSTANCE_TYPE_MARIADB {
			return httperrors.NewInputParameterError("%s only support aliyun %s or %s", privilege, api.DBINSTANCE_TYPE_MARIADB, api.DBINSTANCE_TYPE_MYSQL)
		}
	case api.DATABASE_PRIVILEGE_OWNER:
		if instance.Engine != api.DBINSTANCE_TYPE_SQLSERVER {
			return httperrors.NewInputParameterError("%s only support aliyun %s", privilege, api.DBINSTANCE_TYPE_SQLSERVER)
		}
	default:
		return httperrors.NewInputParameterError("Unknown privilege %s", privilege)
	}
	return nil
}

func (self *SAliyunRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ElasticcacheCreateInput) (*api.ElasticcacheCreateInput, error) {
	if !utils.IsInStringArray(input.Engine, []string{"redis", "memcache"}) {
		return nil, httperrors.NewInputParameterError("invalid engine %s", input.Engine)
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateElasticcacheData(ctx, userCred, ownerId, input)
}

func (self *SAliyunRegionDriver) ValidateCreateElasticcacheAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	elasticCacheV := validators.NewModelIdOrNameValidator("elasticcache", "elasticcache", ownerId)
	accountTypeV := validators.NewStringChoicesValidator("account_type", choices.NewChoices("normal", "admin")).Default("normal")
	accountPrivilegeV := validators.NewStringChoicesValidator("account_privilege", choices.NewChoices("read", "write", "repl"))

	keyV := map[string]validators.IValidator{
		"elasticcache":      elasticCacheV,
		"account_type":      accountTypeV,
		"account_privilege": accountPrivilegeV.Default("read"),
	}

	for _, v := range keyV {
		if err := v.Validate(ctx, data); err != nil {
			return nil, err
		}
	}

	passwd, _ := data.GetString("password")
	err := seclib2.ValidatePassword(passwd)
	if err != nil {
		return nil, httperrors.NewWeakPasswordError()
	}

	if accountPrivilegeV.Value == "repl" && elasticCacheV.Model.(*models.SElasticcache).EngineVersion != "4.0" {
		return nil, httperrors.NewInputParameterError("account_privilege %s only support redis version 4.0",
			accountPrivilegeV.Value)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateCreateElasticcacheAccountData(ctx, userCred, ownerId, data)
}

func (self *SAliyunRegionDriver) RequestCreateElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
		if err != nil {
			return nil, errors.Wrap(nil, "aliyunRegionDriver.CreateElasticcacheAccount.GetElasticcache")
		}

		ec := _ec.(*models.SElasticcache)
		iregion, err := ec.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(nil, "aliyunRegionDriver.CreateElasticcacheAccount.GetIRegion")
		}

		params, err := ea.GetCreateAliyunElasticcacheAccountParams()
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcacheAccount.GetCreateAliyunElasticcacheAccountParams")
		}

		iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcacheAccount.GetIElasticcacheById")
		}

		iea, err := iec.CreateAccount(params)
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcacheAccount.CreateAccount")
		}

		ea.SetModelManager(models.ElasticcacheAccountManager, ea)
		if err := db.SetExternalId(ea, userCred, iea.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcacheAccount.SetExternalId")
		}

		err = cloudprovider.WaitStatusWithDelay(iea, api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE, 3*time.Second, 3*time.Second, 180*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcacheAccount.WaitStatusWithDelay")
		}

		if err = ea.SyncWithCloudElasticcacheAccount(ctx, userCred, iea); err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcacheAccount.SyncWithCloudElasticcache")
		}

		return nil, nil
	})

	return nil
}

func (self *SAliyunRegionDriver) RequestCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, eb *models.SElasticcacheBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_ec, err := db.FetchById(models.ElasticcacheManager, eb.ElasticcacheId)
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.GetElasticcache")
		}

		ec := _ec.(*models.SElasticcache)
		iregion, err := ec.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.GetIRegion")
		}

		iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.GetIElasticcacheById")
		}

		oBackups, err := iec.GetICloudElasticcacheBackups()
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.GetICloudElasticcacheBackups")
		}

		backupIds := []string{}
		for i := range oBackups {
			backupIds = append(backupIds, oBackups[i].GetGlobalId())
		}

		_, err = iec.CreateBackup("")
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.CreateBackup")
		}

		var ieb cloudprovider.ICloudElasticcacheBackup
		cloudprovider.Wait(30*time.Second, 1800*time.Second, func() (b bool, e error) {
			backups, err := iec.GetICloudElasticcacheBackups()
			if err != nil {
				return false, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.WaitCreated")
			}

			for i := range backups {
				if !utils.IsInStringArray(backups[i].GetGlobalId(), backupIds) && backups[i].GetStatus() == api.ELASTIC_CACHE_BACKUP_STATUS_SUCCESS {
					ieb = backups[i]
					return true, nil
				}
			}

			return false, nil
		})

		eb.SetModelManager(models.ElasticcacheBackupManager, eb)
		if err := db.SetExternalId(eb, userCred, ieb.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.SetExternalId")
		}

		if err := eb.SyncWithCloudElasticcacheBackup(ctx, userCred, ieb); err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.SyncWithCloudElasticcacheBackup")
		}

		return nil, nil
	})
	return nil
}

func (self *SAliyunRegionDriver) RequestElasticcacheAccountResetPassword(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	iregion, err := ea.GetIRegion(ctx)
	if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.FetchById")
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.GetIElasticcacheById")
	}

	iea, err := iec.GetICloudElasticcacheAccount(ea.GetExternalId())
	if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.GetICloudElasticcacheBackup")
	}

	data := task.GetParams()
	if data == nil {
		return errors.Wrap(fmt.Errorf("data is nil"), "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.GetParams")
	}

	input, err := ea.GetUpdateAliyunElasticcacheAccountParams(*data)
	if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.GetUpdateAliyunElasticcacheAccountParams")
	}

	err = iea.UpdateAccount(input)
	if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.UpdateAccount")
	}

	if input.Password != nil {
		err = ea.SavePassword(*input.Password)
		if err != nil {
			return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.SavePassword")
		}
	}

	err = cloudprovider.WaitStatusWithDelay(iea, api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE, 10*time.Second, 5*time.Second, 60*time.Second)
	if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.WaitStatusWithDelay")
	}

	return ea.SyncWithCloudElasticcacheAccount(ctx, userCred, iea)
}

func (self *SAliyunRegionDriver) IsSupportedDBInstance() bool {
	return true
}

func (self *SAliyunRegionDriver) GetRdsSupportSecgroupCount() int {
	return 3
}

func (self *SAliyunRegionDriver) IsSupportedDBInstanceAutoRenew() bool {
	return true
}

func (self *SAliyunRegionDriver) IsSupportedElasticcache() bool {
	return true
}

func (self *SAliyunRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return false
}

func (self *SAliyunRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 0
}

func (self *SAliyunRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	cidrV := validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(ctx, jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
		return input, err
	}

	err := IsInPrivateIpRange(cidrV.Value.ToIPRange())
	if err != nil {
		return input, err
	}

	if cidrV.Value.MaskLen > 24 {
		return input, httperrors.NewInputParameterError("invalid cidr range %s, mask length should less than or equal to 24", cidrV.Value.String())
	}

	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateNatGateway(ctx context.Context, userCred mcclient.TokenCredential, input api.NatgatewayCreateInput) (api.NatgatewayCreateInput, error) {
	return input, nil
}

func (self *SAliyunRegionDriver) IsSupportedNatGateway() bool {
	return true
}

func (self *SAliyunRegionDriver) IsSupportedNas() bool {
	return true
}

func (self *SAliyunRegionDriver) ValidateCreateWafInstanceData(ctx context.Context, userCred mcclient.TokenCredential, input api.WafInstanceCreateInput) (api.WafInstanceCreateInput, error) {
	if !regutils.DOMAINNAME_REG.MatchString(input.Name) {
		return input, httperrors.NewInputParameterError("invalid domain name %s", input.Name)
	}
	input.Type = cloudprovider.WafTypeDefault
	if len(input.SourceIps) == 0 && len(input.CloudResources) == 0 {
		return input, httperrors.NewMissingParameterError("source_ips")
	}
	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateWafRuleData(ctx context.Context, userCred mcclient.TokenCredential, waf *models.SWafInstance, input api.WafRuleCreateInput) (api.WafRuleCreateInput, error) {
	return input, httperrors.NewUnsupportOperationError("not supported create rule")
}

func (self *SAliyunRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		rule := input.Rules[i]
		if rule.Priority == nil {
			return nil, httperrors.NewMissingParameterError("priority")
		}

		if *rule.Priority < 1 || *rule.Priority > 100 {
			return nil, httperrors.NewInputParameterError("invalid priority %d, range 1-100", *rule.Priority)
		}

		if len(rule.Ports) > 0 && strings.Contains(input.Rules[i].Ports, ",") {
			return nil, httperrors.NewInputParameterError("invalid ports %s", input.Rules[i].Ports)
		}
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateSecurityGroupInput(ctx, userCred, input)
}

func (self *SAliyunRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Priority != nil && (*input.Priority < 1 || *input.Priority > 100) {
		return nil, httperrors.NewInputParameterError("invalid priority %d, range 1-100", *input.Priority)
	}

	if input.Ports != nil && strings.Contains(*input.Ports, ",") {
		return nil, httperrors.NewInputParameterError("invalid ports %s", *input.Ports)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateSecurityGroupRuleInput(ctx, userCred, input)
}
