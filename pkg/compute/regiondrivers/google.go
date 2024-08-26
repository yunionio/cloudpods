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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGoogleRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SGoogleRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SGoogleRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_GOOGLE
}

func (self *SGoogleRegionDriver) RequestCreateVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		provider := vpc.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to found vpc %s(%s) cloudprovider", vpc.Name, vpc.Id)
		}
		providerDriver, err := provider.GetProvider(ctx)
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
		iRds, err := instance.GetIDBInstance(ctx)
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

		result := models.DBInstanceBackupManager.SyncDBInstanceBackups(ctx, userCred, backup.GetCloudprovider(), instance, region, backups, false)
		log.Infof("SyncDBInstanceBackups for dbinstance %s(%s) result: %s", instance.Name, instance.Id, result.Result())
		instance.SetStatus(ctx, userCred, api.DBINSTANCE_RUNNING, "")
		return nil, nil
	})
	return nil
}

func (self *SGoogleRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	var cidrV = validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(ctx, jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
		return input, err
	}
	if cidrV.Value.MaskLen < 8 || cidrV.Value.MaskLen > 29 {
		return input, httperrors.NewInputParameterError("%s request the mask range should be between 8 and 29", self.GetProvider())
	}
	if len(input.GlobalvpcId) == 0 {
		_manager, err := validators.ValidateModel(ctx, userCred, models.CloudproviderManager, &input.CloudproviderId)
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
	_, err := validators.ValidateModel(ctx, userCred, models.GlobalVpcManager, &input.GlobalvpcId)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SGoogleRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		rule := input.Rules[i]
		if rule.Priority == nil {
			return nil, httperrors.NewMissingParameterError("priority")
		}
		if *rule.Priority < 0 || *rule.Priority > 65535 {
			return nil, httperrors.NewInputParameterError("invalid priority %d, range 0-65535", *rule.Priority)
		}

		if len(rule.Ports) > 0 && strings.Contains(input.Rules[i].Ports, ",") {
			return nil, httperrors.NewInputParameterError("invalid ports %s", input.Rules[i].Ports)
		}
	}
	return input, nil
}

func (self *SGoogleRegionDriver) RequestCreateSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	secgroup *models.SSecurityGroup,
	rules api.SSecgroupRuleResourceSet,
) error {
	vpc, err := secgroup.GetGlobalVpc()
	if err != nil {
		return errors.Wrapf(err, "GetVpc")
	}

	iVpc, err := vpc.GetICloudGlobalVpc(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetICloudGlobalVpc")
	}

	opts := &cloudprovider.SecurityGroupCreateInput{
		Name: secgroup.Name,
	}

	opts.Tags, _ = secgroup.GetAllUserMetadata()

	iGroup, err := iVpc.CreateISecurityGroup(opts)
	if err != nil {
		return errors.Wrapf(err, "CreateISecurityGroup")
	}

	_, err = db.Update(secgroup, func() error {
		secgroup.ExternalId = iGroup.GetGlobalId()
		secgroup.VpcId = ""
		secgroup.CloudregionId = "-"
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SetExternalId")
	}

	for i := range rules {
		opts := cloudprovider.SecurityGroupRuleCreateOptions{
			Desc:      rules[i].Description,
			Direction: secrules.TSecurityRuleDirection(rules[i].Direction),
			Action:    secrules.TSecurityRuleAction(rules[i].Action),
			Protocol:  rules[i].Protocol,
			CIDR:      rules[i].CIDR,
			Ports:     rules[i].Ports,
		}
		_, err := iGroup.CreateRule(&opts)
		if err != nil {
			return errors.Wrapf(err, "CreateRule")
		}
	}

	iRules, err := iGroup.GetRules()
	if err != nil {
		return errors.Wrapf(err, "GetRules")
	}

	result := secgroup.SyncRules(ctx, userCred, iRules)
	if result.IsError() {
		return result.AllError()
	}
	secgroup.SetStatus(ctx, userCred, api.SECGROUP_STATUS_READY, "")
	return nil
}

func (self *SGoogleRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Priority != nil && (*input.Priority < 0 || *input.Priority > 65535) {
		return nil, httperrors.NewInputParameterError("invalid priority %d, range 0-65535", *input.Priority)
	}

	if input.Ports != nil && strings.Contains(*input.Ports, ",") {
		return nil, httperrors.NewInputParameterError("invalid ports %s", *input.Ports)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateSecurityGroupRuleInput(ctx, userCred, input)
}

func (self *SGoogleRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("globalvpc_id", vpc.GlobalvpcId)
	}, nil
}

func (self *SGoogleRegionDriver) CreateDefaultSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	vpc *models.SVpc,
) (*models.SSecurityGroup, error) {
	newGroup := &models.SSecurityGroup{}
	newGroup.SetModelManager(models.SecurityGroupManager, newGroup)
	newGroup.Name = fmt.Sprintf("default-auto-%d", time.Now().Unix())
	newGroup.Description = "auto generage"
	newGroup.ManagerId = vpc.ManagerId
	newGroup.DomainId = ownerId.GetProjectDomainId()
	newGroup.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
	newGroup.GlobalvpcId = vpc.GlobalvpcId
	newGroup.ProjectId = ownerId.GetProjectId()
	err := models.SecurityGroupManager.TableSpec().Insert(ctx, newGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "insert")
	}

	region, err := vpc.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	driver := region.GetDriver()
	err = driver.RequestCreateSecurityGroup(ctx, userCred, newGroup, api.SSecgroupRuleResourceSet{})
	if err != nil {
		return nil, errors.Wrapf(err, "RequestCreateSecurityGroup")
	}
	return newGroup, nil
}
