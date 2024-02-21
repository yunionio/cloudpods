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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHuaWeiRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SHuaWeiRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SHuaWeiRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_HUAWEI
}

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error) {
	// 公网ELB需要指定EIP
	if input.AddressType == api.LB_ADDR_TYPE_INTERNET && len(input.EipId) == 0 {
		return nil, httperrors.NewMissingParameterError("eip_id")
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, ownerId, input)
}

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, input *api.LoadbalancerBackendGroupCreateInput) (*api.LoadbalancerBackendGroupCreateInput, error) {
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerBackendGroupData(ctx, userCred, lb, input)
}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup,
	input *api.LoadbalancerBackendCreateInput) (*api.LoadbalancerBackendCreateInput, error) {
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerBackendData(ctx, userCred, lb, lbbg, input)
}

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerCreateInput,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup) (*api.LoadbalancerListenerCreateInput, error) {
	if len(lbbg.ExternalId) > 0 {
		return input, httperrors.NewResourceBusyError("loadbalancer backend group %s has aleady used by other listener", lbbg.Name)
	}
	return input, nil
}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		provider := lblis.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
		}

		lbbg, err := lblis.GetLoadbalancerBackendGroup()
		if err != nil {
			return nil, errors.Wrapf(err, "GetLoadbalancerBackendGroup")
		}
		lb, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, errors.Wrapf(err, "GetLoadbalancer")
		}
		iLb, err := lb.GetILoadbalancer(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadbalancer")
		}

		opts, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrapf(err, "GetLoadbalancerListenerParams")
		}

		{
			if lblis.ListenerType == api.LB_LISTENER_TYPE_HTTPS && len(lblis.CertificateId) > 0 {
				cert, err := lblis.GetCertificate()
				if err != nil {
					return nil, errors.Wrapf(err, "GetCertificate")
				}
				opts.CertificateId = cert.ExternalId
			}
		}

		if len(lbbg.ExternalId) == 0 {
			lbbgOpts := &cloudprovider.SLoadbalancerBackendGroup{
				Name:      lbbg.Name,
				Scheduler: lblis.Scheduler,
				Protocol:  lblis.ListenerType,
			}

			iLbbg, err := iLb.CreateILoadBalancerBackendGroup(lbbgOpts)
			if err != nil {
				return nil, errors.Wrapf(err, "CreateILoadBalancerBackendGroup")
			}
			err = db.SetExternalId(lbbg, userCred, iLbbg.GetGlobalId())
			if err != nil {
				return nil, errors.Wrapf(err, "db.SetExternalId")
			}
			opts.BackendGroupId = iLbbg.GetGlobalId()
		}

		iLis, err := iLb.CreateILoadBalancerListener(ctx, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateILoadBalancerListener")
		}
		err = db.SetExternalId(lbbg, userCred, iLis.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "lbbg.SetExternalId")
		}
		err = db.SetExternalId(lblis, userCred, iLis.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "lblis.SetExternalId")
		}
		if lblis.HealthCheck == api.LB_BOOL_ON {
			err := iLis.SetHealthCheck(ctx, &opts.ListenerHealthCheckOptions)
			if err != nil {
				return nil, err
			}
		}
		backends, err := lbbg.GetBackends()
		if err != nil {
			return nil, errors.Wrapf(err, "GetBackends")
		}
		if len(backends) == 0 {
			return nil, nil
		}
		iLbbg, err := lbbg.GetICloudLoadbalancerBackendGroup(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetICloudLoadbalancerBackendGroup")
		}
		for i := range backends {
			_, err := iLbbg.AddBackendServer(backends[i].ExternalId, backends[i].Port, backends[i].Weight)
			if err != nil {
				return nil, errors.Wrapf(err, "AddBackendServer")
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg, err := lblis.GetLoadbalancerBackendGroup()
		if err != nil {
			return nil, errors.Wrapf(err, "GetLoadbalancerBackendGroup")
		}
		if len(lbbg.ExternalId) > 0 {
			iLbbg, err := lbbg.GetICloudLoadbalancerBackendGroup(ctx)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					return nil, db.SetExternalId(lbbg, userCred, "")
				}
				return nil, errors.Wrapf(err, "GetICloudLoadbalancerBackendGroup")
			}
			err = iLbbg.Delete(ctx)
			if err != nil {
				return nil, errors.Wrapf(err, "iLbbg.Delete")
			}
			return nil, db.SetExternalId(lbbg, userCred, "")
		}

		if len(lblis.ExternalId) == 0 {
			return nil, nil
		}
		lb, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iLb, err := lb.GetILoadbalancer(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadbalancer")
		}

		iListener, err := iLb.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "GetILoadBalancerListenerById(%s)", lblis.ExternalId)
		}
		return nil, iListener.Delete(ctx)
	})
	return nil
}

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerRuleCreateInput) (*api.LoadbalancerListenerRuleCreateInput, error) {
	if input.Domain == "" && input.Path == "" {
		return input, fmt.Errorf("'domain' or 'path' should not be empty.")
	}
	return input, nil
}

func (self *SHuaWeiRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LoadbalancerListenerRuleUpdateInput) (*api.LoadbalancerListenerRuleUpdateInput, error) {
	return input, nil
}

func (self *SHuaWeiRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, input *api.LoadbalancerBackendUpdateInput) (*api.LoadbalancerBackendUpdateInput, error) {
	if input.Port != nil {
		return input, fmt.Errorf("can not update backend port.")
	}
	return input, nil
}

func (self *SHuaWeiRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput) (*api.LoadbalancerListenerUpdateInput, error) {
	return input, nil
}

func (self *SHuaWeiRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (api.DBInstanceCreateInput, error) {
	if len(input.MasterInstanceId) > 0 && input.Engine == api.DBINSTANCE_TYPE_SQLSERVER {
		return input, httperrors.NewInputParameterError("Not support create read-only dbinstance for %s", input.Engine)
	}

	if len(input.Name) < 4 || len(input.Name) > 64 {
		return input, httperrors.NewInputParameterError("Huawei dbinstance name length shoud be 4~64 characters")
	}

	if input.DiskSizeGB < 40 || input.DiskSizeGB > 4000 {
		return input, httperrors.NewInputParameterError("%s require disk size must in 40 ~ 4000 GB", self.GetProvider())
	}

	if input.DiskSizeGB%10 > 0 {
		return input, httperrors.NewInputParameterError("The disk_size_gb must be an integer multiple of 10")
	}

	if len(input.Password) == 0 { // 华为云RDS必须要有密码
		resetPassword := true
		input.ResetPassword = &resetPassword
	}

	if len(input.SecgroupIds) == 0 {
		input.SecgroupIds = []string{api.SECGROUP_DEFAULT_ID}
	}

	return input, nil
}

func (self *SHuaWeiRegionDriver) InitDBInstanceUser(ctx context.Context, instance *models.SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error {
	user := "root"
	if desc.Engine == api.DBINSTANCE_TYPE_SQLSERVER {
		user = "rdsuser"
	}

	account := models.SDBInstanceAccount{}
	account.DBInstanceId = instance.Id
	account.Name = user
	account.Status = api.DBINSTANCE_USER_AVAILABLE
	account.SetModelManager(models.DBInstanceAccountManager, &account)
	err := models.DBInstanceAccountManager.TableSpec().Insert(ctx, &account)
	if err != nil {
		return err
	}

	return account.SetPassword(desc.Password)
}

func (self *SHuaWeiRegionDriver) IsSupportedBillingCycle(bc billing.SBillingCycle, resource string) bool {
	switch resource {
	case models.DBInstanceManager.KeywordPlural():
		years := bc.GetYears()
		months := bc.GetMonths()
		if (years >= 1 && years <= 3) || (months >= 1 && months <= 9) {
			return true
		}
	}
	return false
}

func (self *SHuaWeiRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceAccountCreateInput) (api.DBInstanceAccountCreateInput, error) {
	if utils.IsInStringArray(instance.Engine, []string{api.DBINSTANCE_TYPE_POSTGRESQL, api.DBINSTANCE_TYPE_SQLSERVER}) {
		return input, httperrors.NewInputParameterError("Not support create account for huawei cloud %s instance", instance.Engine)
	}
	if len(input.Name) == len(input.Password) {
		for i := range input.Name {
			if input.Name[i] != input.Password[len(input.Password)-i-1] {
				return input, nil
			}
		}
		return input, httperrors.NewInputParameterError("Huawei rds password cannot be in the same reverse order as the account")
	}
	return input, nil
}

func (self *SHuaWeiRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceDatabaseCreateInput) (api.DBInstanceDatabaseCreateInput, error) {
	if utils.IsInStringArray(instance.Engine, []string{api.DBINSTANCE_TYPE_POSTGRESQL, api.DBINSTANCE_TYPE_SQLSERVER}) {
		return input, httperrors.NewInputParameterError("Not support create database for huawei cloud %s instance", instance.Engine)
	}
	return input, nil
}

func (self *SHuaWeiRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceBackupCreateInput) (api.DBInstanceBackupCreateInput, error) {
	if len(input.Name) < 4 || len(input.Name) > 64 {
		return input, httperrors.NewInputParameterError("Huawei DBInstance backup name length shoud be 4~64 characters")
	}

	if len(input.Databases) > 0 && instance.Engine != api.DBINSTANCE_TYPE_SQLSERVER {
		return input, httperrors.NewInputParameterError("Huawei only supports specified databases with %s", api.DBINSTANCE_TYPE_SQLSERVER)
	}

	return input, nil
}

func (self *SHuaWeiRegionDriver) ValidateChangeDBInstanceConfigData(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, input *api.SDBInstanceChangeConfigInput) error {
	if input.DiskSizeGB != 0 && input.DiskSizeGB < instance.DiskSizeGB {
		return httperrors.NewUnsupportOperationError("Huawei DBInstance Disk cannot be thrink")
	}
	return nil
}

func (self *SHuaWeiRegionDriver) IsSupportDBInstancePublicConnection() bool {
	//目前华为云未对外开放打开远程连接的API接口
	return false
}

func (self *SHuaWeiRegionDriver) ValidateResetDBInstancePassword(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string) error {
	return httperrors.NewUnsupportOperationError("Huawei current not support reset dbinstance account password")
}

func (self *SHuaWeiRegionDriver) IsSupportKeepDBInstanceManualBackup() bool {
	return true
}

func (self *SHuaWeiRegionDriver) ValidateDBInstanceAccountPrivilege(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string, privilege string) error {
	if account == "root" {
		return httperrors.NewInputParameterError("No need to grant or revoke privilege for admin account")
	}
	if !utils.IsInStringArray(privilege, []string{api.DATABASE_PRIVILEGE_RW, api.DATABASE_PRIVILEGE_R}) {
		return httperrors.NewInputParameterError("Unknown privilege %s", privilege)
	}
	return nil
}

// https://support.huaweicloud.com/api-rds/rds_09_0009.html
func (self *SHuaWeiRegionDriver) ValidateDBInstanceRecovery(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, input api.SDBInstanceRecoveryConfigInput) error {
	if backup.Engine == api.DBINSTANCE_TYPE_POSTGRESQL {
		return httperrors.NewNotSupportedError("%s not support recovery", backup.Engine)
	}
	if backup.DBInstanceId == instance.Id && instance.Engine != api.DBINSTANCE_TYPE_SQLSERVER {
		return httperrors.NewNotSupportedError("Huawei %s rds not support recovery from it self rds backup", instance.Engine)
	}
	if len(input.Databases) > 0 {
		if instance.Engine != api.DBINSTANCE_TYPE_SQLSERVER {
			return httperrors.NewInputParameterError("Huawei only %s engine support databases recovery", instance.Engine)
		}
		invalidDbs := []string{"rdsadmin", "master", "msdb", "tempdb", "model"}
		for _, db := range input.Databases {
			if utils.IsInStringArray(strings.ToLower(db), invalidDbs) {
				return httperrors.NewInputParameterError("New databases name can not be one of %s", invalidDbs)
			}
		}
	}
	return nil
}

func validatorSlaveZones(ctx context.Context, ownerId mcclient.IIdentityProvider, regionId string, data *jsonutils.JSONDict, optional bool) error {
	s, err := data.GetString("slave_zones")
	if err != nil {
		if optional {
			return nil
		}

		return fmt.Errorf("missing parameter slave_zones")
	}

	zones := strings.Split(s, ",")
	ret := []string{}
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	for i := range zones {
		_data := jsonutils.NewDict()
		_data.Set("zone", jsonutils.NewString(zones[i]))
		if err := zoneV.Validate(ctx, _data); err != nil {
			return errors.Wrap(err, "validatorSlaveZones")
		} else {
			if zoneV.Model.(*models.SZone).GetCloudRegionId() != regionId {
				return errors.Wrap(fmt.Errorf("zone %s is not in region %s", zoneV.Model.GetName(), regionId), "GetCloudRegionId")
			}
			ret = append(ret, zoneV.Model.GetId())
		}
	}

	//if sku, err := data.GetString("sku"); err != nil || len(sku) == 0 {
	//	return httperrors.NewMissingParameterError("sku")
	//} else {
	//	chargeType, _ := data.GetString("charge_type")
	//
	//	_skuModel, err := db.FetchByIdOrName(models.ElasticcacheSkuManager, ownerId, sku)
	//	if err != nil {
	//		return err
	//	}
	//
	//	skuModel := _skuModel.(*models.SElasticcacheSku)
	//	for _, zoneId := range zones {
	//		if err := ValidateElasticcacheSku(zoneId, chargeType, skuModel, nil); err != nil {
	//			return err
	//		}
	//	}
	//}

	data.Set("slave_zones", jsonutils.NewString(strings.Join(ret, ",")))
	return nil
}

func (self *SHuaWeiRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ElasticcacheCreateInput) (*api.ElasticcacheCreateInput, error) {
	if !utils.IsInStringArray(input.Engine, []string{"redis", "memcache"}) {
		return nil, httperrors.NewInputParameterError("invalid engine %s", input.Engine)
	}
	if len(input.MaintainStartTime) > 0 && !utils.IsInStringArray(input.MaintainStartTime, []string{"22:00:00", "02:00:00", "06:00:00", "10:00:00", "14:00:00", "18:00:00"}) {
		return nil, httperrors.NewInputParameterError("invalid maintain_start_time %s", input.MaintainStartTime)
	}

	if input.CapacityMb == 0 {
		return nil, httperrors.NewMissingParameterError("capacity_mb")
	}

	return self.SManagedVirtualizationRegionDriver.ValidateCreateElasticcacheData(ctx, userCred, ownerId, input)
}

func (self *SHuaWeiRegionDriver) ValidateCreateElasticcacheAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewUnsupportOperationError("%s not support create account", self.GetProvider())
}

func (self *SHuaWeiRegionDriver) RequestElasticcacheAccountResetPassword(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	iregion, err := ea.GetIRegion(ctx)
	if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.FetchById")
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.GetIElasticcacheById")
	}

	iea, err := iec.GetICloudElasticcacheAccount(ea.GetExternalId())
	if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.GetICloudElasticcacheBackup")
	}

	data := task.GetParams()
	if data == nil {
		return errors.Wrap(fmt.Errorf("data is nil"), "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.GetParams")
	}

	input, err := ea.GetUpdateHuaweiElasticcacheAccountParams(*data)
	if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.GetUpdateHuaweiElasticcacheAccountParams")
	}

	err = iea.UpdateAccount(input)
	if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.UpdateAccount")
	}

	if input.Password != nil {
		err = ea.SavePassword(*input.Password)
		if err != nil {
			return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.SavePassword")
		}
	}

	return ea.SetStatus(ctx, userCred, api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE, "")
}

func (self *SHuaWeiRegionDriver) RequestUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return errors.Wrap(fmt.Errorf("not support update huawei elastic cache auth_mode"), "HuaWeiRegionDriver.RequestUpdateElasticcacheAuthMode")
}

func (self *SHuaWeiRegionDriver) AllowCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *models.SElasticcache) error {
	if elasticcache.LocalCategory == api.ELASTIC_CACHE_ARCH_TYPE_SINGLE {
		return httperrors.NewBadRequestError("huawei %s mode elastic not support create backup", elasticcache.LocalCategory)
	}

	return nil
}

func (self *SHuaWeiRegionDriver) AllowUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *models.SElasticcache) error {
	return fmt.Errorf("not support update huawei elastic cache auth_mode")
}

func (self *SHuaWeiRegionDriver) IsSupportedDBInstance() bool {
	return true
}

func (self *SHuaWeiRegionDriver) IsSupportedElasticcache() bool {
	return true
}

func (self *SHuaWeiRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return false
}

func (self *SHuaWeiRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 0
}

func (self *SHuaWeiRegionDriver) GetBackendStatusForAdd() []string {
	return []string{api.VM_RUNNING, api.VM_READY}
}

func (self *SHuaWeiRegionDriver) GetRdsSupportSecgroupCount() int {
	return 1
}

func (self *SHuaWeiRegionDriver) IsSupportedElasticcacheAutoRenew() bool {
	return false
}

func (self *SHuaWeiRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	var cidrV = validators.NewIPv4PrefixValidator("cidr_block")
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

func (self *SHuaWeiRegionDriver) OnNatEntryDeleteComplete(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip) error {
	return models.StartResourceSyncStatusTask(ctx, userCred, eip, "EipSyncstatusTask", "")
}

func (self *SHuaWeiRegionDriver) RequestAssociateEipForNAT(ctx context.Context, userCred mcclient.TokenCredential, nat *models.SNatGateway, eip *models.SElasticip, task taskman.ITask) error {
	_, err := db.Update(eip, func() error {
		eip.AssociateType = api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
		eip.AssociateId = nat.Id
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return task.ScheduleRun(nil)
}

func (self *SHuaWeiRegionDriver) ValidateCreateNatGateway(ctx context.Context, userCred mcclient.TokenCredential, input api.NatgatewayCreateInput) (api.NatgatewayCreateInput, error) {
	if len(input.Eip) > 0 || input.EipBw > 0 {
		return input, httperrors.NewInputParameterError("Huawei nat not support associate eip")
	}
	return input, nil
}

func (self *SHuaWeiRegionDriver) IsSupportedNatGateway() bool {
	return true
}

func (self *SHuaWeiRegionDriver) IsSupportedNas() bool {
	return true
}

func (self *SHuaWeiRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		rule := input.Rules[i]
		if input.Rules[i].Priority == nil {
			return nil, httperrors.NewMissingParameterError("priority")
		}
		if *input.Rules[i].Priority < 1 || *input.Rules[i].Priority > 100 {
			return nil, httperrors.NewInputParameterError("invalid priority %d, range 1-100", *input.Rules[i].Priority)
		}

		if len(rule.Ports) > 0 && strings.Contains(input.Rules[i].Ports, ",") {
			return nil, httperrors.NewInputParameterError("invalid ports %s", input.Rules[i].Ports)
		}

	}
	return input, nil
}

func (self *SHuaWeiRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	return nil, httperrors.NewNotSupportedError("not support update security group rule")
}

func (self *SHuaWeiRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("cloudregion_id", vpc.CloudregionId).Equals("manager_id", vpc.ManagerId)
	}, nil
}
