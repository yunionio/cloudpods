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
	"time"

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
	"yunion.io/x/onecloud/pkg/util/choices"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SQcloudRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SQcloudRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SQcloudRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error) {
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, ownerId, input)
}

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerCreateInput,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup) (*api.LoadbalancerListenerCreateInput, error) {
	if len(lbbg.ExternalId) > 0 {
		return input, httperrors.NewResourceBusyError("loadbalancer backend group %s has aleady used by other listener", lbbg.Name)
	}
	return input, nil
}

func (self *SQcloudRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SQcloudRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		provider := lblis.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
		}

		opts, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrapf(err, "lblis.GetLoadbalancerListenerParams")
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

func (self *SQcloudRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		err := func() error {
			if len(lblis.ExternalId) == 0 {
				return nil
			}
			lb, err := lblis.GetLoadbalancer()
			if err != nil {
				return errors.Wrapf(err, "GetLoadbalancer")
			}
			iLb, err := lb.GetILoadbalancer(ctx)
			if err != nil {
				return errors.Wrapf(err, "GetILoadbalancer")
			}

			iListener, err := iLb.GetILoadBalancerListenerById(lblis.ExternalId)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					return nil
				}
				return errors.Wrapf(err, "GetILoadBalancerListenerById(%s)", lblis.ExternalId)
			}
			return iListener.Delete(ctx)
		}()
		if err != nil {
			return nil, err
		}
		lbbg, err := lblis.GetLoadbalancerBackendGroup()
		if err != nil {
			return nil, errors.Wrapf(err, "GetLoadbalancerBackendGroup")
		}
		return nil, db.SetExternalId(lbbg, userCred, "")
	})
	return nil
}

func (self *SQcloudRegionDriver) GetLoadbalancerListenerRuleInputParams(lblis *models.SLoadbalancerListener, lbr *models.SLoadbalancerListenerRule) *cloudprovider.SLoadbalancerListenerRule {
	scheduler := ""
	switch lblis.Scheduler {
	case api.LB_SCHEDULER_WRR:
		scheduler = "WRR"
	case api.LB_SCHEDULER_WLC:
		scheduler = "LEAST_CONN"
	case api.LB_SCHEDULER_SCH:
		scheduler = "IP_HASH"
	default:
		scheduler = "WRR"
	}

	sessionTimeout := 0
	if lblis.StickySession == api.LB_BOOL_ON {
		sessionTimeout = lblis.StickySessionCookieTimeout
	}

	rule := &cloudprovider.SLoadbalancerListenerRule{
		Name:   lbr.Name,
		Domain: lbr.Domain,
		Path:   lbr.Path,

		Scheduler: scheduler,

		HealthCheck:         lblis.HealthCheck,
		HealthCheckType:     lblis.HealthCheckType,
		HealthCheckTimeout:  lblis.HealthCheckTimeout,
		HealthCheckDomain:   lblis.HealthCheckDomain,
		HealthCheckHttpCode: lblis.HealthCheckHttpCode,
		HealthCheckURI:      lblis.HealthCheckURI,
		HealthCheckInterval: lblis.HealthCheckInterval,

		HealthCheckRise: lblis.HealthCheckRise,
		HealthCheckFail: lblis.HealthCheckFall,

		StickySessionCookieTimeout: sessionTimeout,
	}

	return rule
}

func (self *SQcloudRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return nil, cloudprovider.ErrNotImplemented
	})
	return nil
}

func (self *SQcloudRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	cidrV := validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(ctx, jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
		return input, err
	}

	err := IsInPrivateIpRange(cidrV.Value.ToIPRange())
	if err != nil {
		return input, err
	}

	if cidrV.Value.MaskLen < 16 || cidrV.Value.MaskLen > 28 {
		return input, httperrors.NewInputParameterError("%s request the mask range should be between 16 and 28", self.GetProvider())
	}
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput) (*api.LoadbalancerListenerUpdateInput, error) {
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, input *api.LoadbalancerBackendUpdateInput) (*api.LoadbalancerBackendUpdateInput, error) {
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerRuleCreateInput) (*api.LoadbalancerListenerRuleCreateInput, error) {
	if len(input.Path) == 0 {
		return nil, httperrors.NewInputParameterError("path can not be emtpy")
	}
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LoadbalancerListenerRuleUpdateInput) (*api.LoadbalancerListenerRuleUpdateInput, error) {
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup, input *api.LoadbalancerBackendCreateInput) (*api.LoadbalancerBackendCreateInput, error) {
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerBackendData(ctx, userCred, lb, lbbg, input)
}

func (self *SQcloudRegionDriver) RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return nil, cloudprovider.ErrNotImplemented
	})
	return nil
}

func (self *SQcloudRegionDriver) InitDBInstanceUser(ctx context.Context, instance *models.SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error {
	user := "root"
	account := models.SDBInstanceAccount{}
	account.DBInstanceId = instance.Id
	account.Name = user
	account.Host = "%"
	if instance.Engine == api.DBINSTANCE_TYPE_MYSQL && instance.Category == api.QCLOUD_DBINSTANCE_CATEGORY_BASIC {
		account.Host = "localhost"
	}
	account.Status = api.DBINSTANCE_USER_AVAILABLE
	account.SetModelManager(models.DBInstanceAccountManager, &account)
	err := models.DBInstanceAccountManager.TableSpec().Insert(ctx, &account)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}
	return account.SetPassword(desc.Password)
}

func (self *SQcloudRegionDriver) IsSupportedDBInstance() bool {
	return true
}

func (self *SQcloudRegionDriver) IsSupportedDBInstanceAutoRenew() bool {
	return true
}

func (self *SQcloudRegionDriver) GetRdsSupportSecgroupCount() int {
	return 5
}

func (self *SQcloudRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (api.DBInstanceCreateInput, error) {
	if input.Engine == api.DBINSTANCE_TYPE_MYSQL && input.Category != api.QCLOUD_DBINSTANCE_CATEGORY_BASIC && len(input.SecgroupIds) == 0 {
		input.SecgroupIds = []string{api.SECGROUP_DEFAULT_ID}
	}
	return input, nil
}

func (self *SQcloudRegionDriver) IsSupportedBillingCycle(bc billing.SBillingCycle, resource string) bool {
	switch resource {
	case models.DBInstanceManager.KeywordPlural(), models.ElasticcacheManager.KeywordPlural():
		years := bc.GetYears()
		months := bc.GetMonths()
		if (years >= 1 && years <= 3) || (months >= 1 && months <= 12) {
			return true
		}
	}
	return false
}

func (self *SQcloudRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceBackupCreateInput) (api.DBInstanceBackupCreateInput, error) {
	switch instance.Engine {
	case api.DBINSTANCE_TYPE_MYSQL:
		if instance.Category == api.QCLOUD_DBINSTANCE_CATEGORY_BASIC {
			return input, httperrors.NewNotSupportedError("Qcloud Basic MySQL instance not support create backup")
		}
	}
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceAccountCreateInput) (api.DBInstanceAccountCreateInput, error) {
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceDatabaseCreateInput) (api.DBInstanceDatabaseCreateInput, error) {
	return input, httperrors.NewNotSupportedError("Not support create Qcloud databases")
}

func (self *SQcloudRegionDriver) ValidateDBInstanceAccountPrivilege(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string, privilege string) error {
	switch privilege {
	case api.DATABASE_PRIVILEGE_RW:
	case api.DATABASE_PRIVILEGE_R:
	default:
		return httperrors.NewInputParameterError("Unknown privilege %s", privilege)
	}
	return nil
}

func (self *SQcloudRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return true
}

func (self *SQcloudRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 10
}

func (self *SQcloudRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ElasticcacheCreateInput) (*api.ElasticcacheCreateInput, error) {
	if len(input.NetworkType) == 0 {
		input.NetworkType = api.LB_NETWORK_TYPE_VPC
	}
	input.Engine = "redis"
	return self.SManagedVirtualizationRegionDriver.ValidateCreateElasticcacheData(ctx, userCred, ownerId, input)
}

func (self *SQcloudRegionDriver) IsSupportedElasticcache() bool {
	return true
}

func (self *SQcloudRegionDriver) ValidateCreateElasticcacheAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	elasticCacheV := validators.NewModelIdOrNameValidator("elasticcache", "elasticcache", ownerId)
	accountTypeV := validators.NewStringChoicesValidator("account_type", choices.NewChoices("normal")).Default("normal")
	accountPrivilegeV := validators.NewStringChoicesValidator("account_privilege", choices.NewChoices("read", "write"))

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

	ec := elasticCacheV.Model.(*models.SElasticcache)
	if ec.Engine == "redis" && ec.EngineVersion == "2.8" {
		return nil, httperrors.NewNotSupportedError("redis version 2.8 not support create account")
	}

	passwd, _ := data.GetString("password")
	err := seclib2.ValidatePassword(passwd)
	if err != nil {
		return nil, err
	}

	return self.SManagedVirtualizationRegionDriver.ValidateCreateElasticcacheAccountData(ctx, userCred, ownerId, data)
}

func (self *SQcloudRegionDriver) RequestCreateElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
		if err != nil {
			return nil, errors.Wrap(nil, "qcloudRegionDriver.CreateElasticcacheAccount.GetElasticcache")
		}

		ec := _ec.(*models.SElasticcache)
		iregion, err := ec.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(nil, "qcloudRegionDriver.CreateElasticcacheAccount.GetIRegion")
		}

		params, err := ea.GetCreateQcloudElasticcacheAccountParams()
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.CreateElasticcacheAccount.GetCreateQcloudElasticcacheAccountParams")
		}

		iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.CreateElasticcacheAccount.GetIElasticcacheById")
		}

		iea, err := iec.CreateAccount(params)
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.CreateElasticcacheAccount.CreateAccount")
		}

		ea.SetModelManager(models.ElasticcacheAccountManager, ea)
		if err := db.SetExternalId(ea, userCred, iea.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.CreateElasticcacheAccount.SetExternalId")
		}

		err = cloudprovider.WaitStatusWithDelay(iea, api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE, 3*time.Second, 3*time.Second, 180*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.CreateElasticcacheAccount.WaitStatusWithDelay")
		}

		if err = ea.SyncWithCloudElasticcacheAccount(ctx, userCred, iea); err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.CreateElasticcacheAccount.SyncWithCloudElasticcache")
		}

		return nil, nil
	})

	return nil
}

func (self *SQcloudRegionDriver) RequestElasticcacheAccountResetPassword(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	iregion, err := ea.GetIRegion(ctx)
	if err != nil {
		return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.FetchById")
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.GetIElasticcacheById")
	}

	iea, err := iec.GetICloudElasticcacheAccount(ea.GetExternalId())
	if err != nil {
		return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.GetICloudElasticcacheAccount")
	}

	data := task.GetParams()
	if data == nil {
		return errors.Wrap(fmt.Errorf("data is nil"), "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.GetParams")
	}

	input, err := ea.GetUpdateQcloudElasticcacheAccountParams(*data)
	if err != nil {
		return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.GetUpdateQcloudElasticcacheAccountParams")
	}

	if iec.GetEngine() == "redis" && iec.GetEngineVersion() == "2.8" {
		pwd := ""
		if input.Password != nil {
			pwd = *input.Password
		}

		noAuth := false
		if len(pwd) > 0 {
			noAuth = false
		} else if input.NoPasswordAccess != nil {
			noAuth = *input.NoPasswordAccess
		} else if ec.AuthMode == "off" {
			noAuth = true
		}

		err = iec.UpdateAuthMode(noAuth, pwd)
	} else {
		err = iea.UpdateAccount(input)
		if err != nil {
			return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.UpdateAccount")
		}
	}

	if input.Password != nil {
		err = ea.SavePassword(*input.Password)
		if err != nil {
			return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.SavePassword")
		}

		if iea.GetName() == "root" {
			_, err := db.UpdateWithLock(ctx, ec, func() error {
				ec.AuthMode = api.LB_BOOL_ON
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.UpdateAuthMode")
			}
		}
	}

	err = cloudprovider.WaitStatusWithDelay(iea, api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE, 10*time.Second, 5*time.Second, 60*time.Second)
	if err != nil {
		return errors.Wrap(err, "qcloudRegionDriver.RequestElasticcacheAccountResetPassword.WaitStatusWithDelay")
	}

	return ea.SyncWithCloudElasticcacheAccount(ctx, userCred, iea)
}

func (self *SQcloudRegionDriver) IsCertificateBelongToRegion() bool {
	return false
}

func (self *SQcloudRegionDriver) ValidateCreateCdnData(ctx context.Context, userCred mcclient.TokenCredential, input api.CDNDomainCreateInput) (api.CDNDomainCreateInput, error) {
	if !utils.IsInStringArray(input.ServiceType, []string{
		api.CDN_SERVICE_TYPE_WEB,
		api.CND_SERVICE_TYPE_DOWNLOAD,
		api.CND_SERVICE_TYPE_MEDIA,
	}) {
		return input, httperrors.NewNotSupportedError("service_type %s", input.ServiceType)
	}
	if !utils.IsInStringArray(input.Area, []string{
		api.CDN_DOMAIN_AREA_MAINLAND,
		api.CDN_DOMAIN_AREA_OVERSEAS,
		api.CDN_DOMAIN_AREA_GLOBAL,
	}) {
		return input, httperrors.NewNotSupportedError("area %s", input.Area)
	}
	if input.Origins == nil {
		return input, httperrors.NewMissingParameterError("origins")
	}
	for _, origin := range *input.Origins {
		if len(origin.Origin) == 0 {
			return input, httperrors.NewMissingParameterError("origins.origin")
		}
		if !utils.IsInStringArray(origin.Type, []string{
			api.CDN_DOMAIN_ORIGIN_TYPE_DOMAIN,
			api.CDN_DOMAIN_ORIGIN_TYPE_IP,
			api.CDN_DOMAIN_ORIGIN_TYPE_BUCKET,
			api.CDN_DOMAIN_ORIGIN_THIRED_PARTY,
		}) {
			return input, httperrors.NewInputParameterError("invalid origin type %s", origin.Type)
		}
	}
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		rule := input.Rules[i]
		if rule.Priority == nil {
			return nil, httperrors.NewMissingParameterError("priority")
		}

		if *rule.Priority < 0 || *rule.Priority > 99 {
			return nil, httperrors.NewInputParameterError("invalid priority %d, range 0-99", *rule.Priority)
		}

	}
	return input, nil
}

func (self *SQcloudRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Priority != nil && (*input.Priority < 0 || *input.Priority > 99) {
		return nil, httperrors.NewInputParameterError("invalid priority %d, range 0-99", *input.Priority)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateSecurityGroupRuleInput(ctx, userCred, input)
}

func (self *SQcloudRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("cloudregion_id", vpc.CloudregionId).Equals("manager_id", vpc.ManagerId)
	}, nil
}
