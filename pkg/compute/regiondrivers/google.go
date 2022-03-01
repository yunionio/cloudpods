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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/pinyinutils"
)

type SGoogleRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SGoogleRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SGoogleRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return false
}

// 名称必须以小写字母开头，后面最多可跟 62 个小写字母、数字或连字符，但不能以连字符结尾
func (self *SGoogleRegionDriver) GenerateSecurityGroupName(name string) string {
	ret := ""
	for _, s := range strings.ToLower(pinyinutils.Text2Pinyin(name)) {
		if (s >= 'a' && s <= 'z') || (s >= '0' && s <= '9') || (s == '-') {
			ret = fmt.Sprintf("%s%s", ret, string(s))
		}
	}
	if len(ret) > 0 && (ret[0] < 'a' || ret[0] > 'z') {
		ret = fmt.Sprintf("sg-%s", ret)
	}
	return ret
}

func (self *SGoogleRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SGoogleRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:allow any")}
}

func (self *SGoogleRegionDriver) GetSecurityGroupRuleMaxPriority() int {
	return 0
}

func (self *SGoogleRegionDriver) GetSecurityGroupRuleMinPriority() int {
	return 65534
}

func (self *SGoogleRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_GOOGLE
}

func (self *SGoogleRegionDriver) IsSecurityGroupBelongGlobalVpc() bool {
	return true
}

func (self *SGoogleRegionDriver) IsVpcBelongGlobalVpc() bool {
	return true
}

func (self *SGoogleRegionDriver) RequestCreateVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		provider := vpc.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to found vpc %s(%s) cloudprovider", vpc.Name, vpc.Id)
		}
		providerDriver, err := provider.GetProvider()
		if err != nil {
			return nil, errors.Wrap(err, "provider.GetProvider")
		}
		iregion, err := providerDriver.GetIRegionById(region.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetIRegion")
		}
		gvpc, err := vpc.GetGlobalVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "GetGlobalVpc")
		}
		opts := &cloudprovider.VpcCreateOptions{
			NAME:                vpc.Name,
			CIDR:                vpc.CidrBlock,
			GlobalVpcExternalId: gvpc.ExternalId,
			Desc:                vpc.Description,
		}
		ivpc, err := iregion.CreateIVpc(opts)
		if err != nil {
			return nil, errors.Wrap(err, "iregion.CreateIVpc")
		}
		db.SetExternalId(vpc, userCred, ivpc.GetGlobalId())

		regions, err := provider.GetRegionByExternalIdPrefix(self.GetProvider())
		if err != nil {
			return nil, errors.Wrap(err, "GetRegionByExternalIdPrefix")
		}
		for _, region := range regions {
			iregion, err := providerDriver.GetIRegionById(region.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "providerDrivder.GetIRegionById")
			}
			region.SyncVpcs(ctx, userCred, iregion, provider)
		}

		err = vpc.SyncWithCloudVpc(ctx, userCred, ivpc, nil)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncWithCloudVpc")
		}

		err = vpc.SyncRemoteWires(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncRemoteWires")
		}
		return nil, nil
	})
	return nil
}

func (self *SGoogleRegionDriver) IsSupportedDBInstance() bool {
	return true
}

func (self *SGoogleRegionDriver) ValidateDBInstanceRecovery(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, input api.SDBInstanceRecoveryConfigInput) error {
	return nil
}

func (self *SGoogleRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (api.DBInstanceCreateInput, error) {
	if input.BillingType == billing_api.BILLING_TYPE_PREPAID {
		return input, httperrors.NewInputParameterError("Google dbinstance not support prepaid billing type")
	}

	if input.DiskSizeGB < 10 || input.DiskSizeGB > 30720 {
		return input, httperrors.NewInputParameterError("disk size gb must in range 10 ~ 30720 Gb")
	}

	if input.Engine != api.DBINSTANCE_TYPE_MYSQL && len(input.Password) == 0 {
		return input, httperrors.NewMissingParameterError("password")
	}

	return input, nil
}

func (self *SGoogleRegionDriver) InitDBInstanceUser(ctx context.Context, instance *models.SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error {
	user := "root"
	switch desc.Engine {
	case api.DBINSTANCE_TYPE_POSTGRESQL:
		user = "postgres"
	case api.DBINSTANCE_TYPE_SQLSERVER:
		user = "sqlserver"
	default:
		user = "root"
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

func (self *SGoogleRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceDatabaseCreateInput) (api.DBInstanceDatabaseCreateInput, error) {
	return input, nil
}

func (self *SGoogleRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceBackupCreateInput) (api.DBInstanceBackupCreateInput, error) {
	return input, nil
}

func (self *SGoogleRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceAccountCreateInput) (api.DBInstanceAccountCreateInput, error) {
	return input, nil
}

func (self *SGoogleRegionDriver) RequestCreateDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRds, err := instance.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrap(err, "instance.GetIDBInstance")
		}

		desc := &cloudprovider.SDBInstanceBackupCreateConfig{
			Name:        backup.Name,
			Description: backup.Description,
		}

		_, err = iRds.CreateIBackup(desc)
		if err != nil {
			return nil, errors.Wrap(err, "iRds.CreateBackup")
		}

		backups, err := iRds.GetIDBInstanceBackups()
		if err != nil {
			return nil, errors.Wrap(err, "iRds.GetIDBInstanceBackups")
		}

		region, _ := backup.GetRegion()

		result := models.DBInstanceBackupManager.SyncDBInstanceBackups(ctx, userCred, backup.GetCloudprovider(), instance, region, backups)
		log.Infof("SyncDBInstanceBackups for dbinstance %s(%s) result: %s", instance.Name, instance.Id, result.Result())
		instance.SetStatus(userCred, api.DBINSTANCE_RUNNING, "")
		return nil, nil
	})
	return nil
}

func (self *SGoogleRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	var cidrV = validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
		return input, err
	}
	if cidrV.Value.MaskLen < 8 || cidrV.Value.MaskLen > 29 {
		return input, httperrors.NewInputParameterError("%s request the mask range should be between 8 and 29", self.GetProvider())
	}
	if len(input.GlobalvpcId) == 0 {
		_manager, err := validators.ValidateModel(userCred, models.CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return input, err
		}
		manager := _manager.(*models.SCloudprovider)
		globalVpcs, err := manager.GetGlobalVpcs()
		if err != nil {
			return input, errors.Wrapf(err, "GetGlobalVpcs")
		}
		if len(globalVpcs) != 1 {
			return input, httperrors.NewMissingParameterError("globalvpc_id")
		}
		input.GlobalvpcId = globalVpcs[0].Id
	}
	_, err := validators.ValidateModel(userCred, models.GlobalVpcManager, &input.GlobalvpcId)
	if err != nil {
		return input, err
	}
	return input, nil
}
