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
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SAwsRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SAwsRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SAwsRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_AWS
}

func (self *SAwsRegionDriver) IsSupportedDBInstance() bool {
	return true
}

func (self *SAwsRegionDriver) GetRdsSupportSecgroupCount() int {
	return 1
}

func (self *SAwsRegionDriver) IsSupportedElasticcache() bool {
	return true
}

func (self *SAwsRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return false
}

func (self *SAwsRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 0
}

func (self *SAwsRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ElasticcacheCreateInput) (*api.ElasticcacheCreateInput, error) {
	if len(input.Password) > 0 && len(input.Password) < 16 {
		passwd := seclib2.RandomPassword2(30)
		input.Password = ""
		for _, s := range passwd {
			if ok, _ := utils.InArray(s, []rune{'!', '&', '#', '$', '^', '<', '>', '-', '.'}); ok || unicode.IsDigit(s) || unicode.IsLetter(s) {
				input.Password += string(s)
			}
		}
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateElasticcacheData(ctx, userCred, ownerId, input)
}

func (self *SAwsRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (api.DBInstanceCreateInput, error) {
	if len(input.Password) > 0 {
		for _, s := range input.Password {
			if s == '/' || s == '"' || s == '@' || s == '\'' {
				return input, httperrors.NewInputParameterError("aws rds not support password character %s", string(s))
			}
		}
	}
	if len(input.Password) == 0 {
		for _, s := range seclib2.RandomPassword2(100) {
			if s == '/' || s == '"' || s == '@' || s == '\'' {
				continue
			}
			input.Password += string(s)
			if len(input.Password) >= 20 {
				break
			}
		}
	}
	return input, nil
}

func (self *SAwsRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceBackupCreateInput) (api.DBInstanceBackupCreateInput, error) {
	return input, nil
}

func (self *SAwsRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceDatabaseCreateInput) (api.DBInstanceDatabaseCreateInput, error) {
	return input, httperrors.NewNotSupportedError("aws not support create rds database")
}

func (self *SAwsRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceAccountCreateInput) (api.DBInstanceAccountCreateInput, error) {
	return input, httperrors.NewNotSupportedError("aws not support create rds account")
}

func (self *SAwsRegionDriver) InitDBInstanceUser(ctx context.Context, instance *models.SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error {
	user := "admin"
	if desc.Engine == api.DBINSTANCE_TYPE_POSTGRESQL || desc.Category == api.DBINSTANCE_TYPE_POSTGRESQL {
		user = "postgres"
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

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error) {
	if len(input.LoadbalancerSpec) == 0 {
		input.LoadbalancerSpec = api.LB_AWS_SPEC_APPLICATION
	}
	if !utils.IsInStringArray(input.LoadbalancerSpec, api.LB_AWS_SPECS) {
		return nil, httperrors.NewInputParameterError("invalid loadbalancer_spec %s", input.LoadbalancerSpec)
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, ownerId, input)
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerCreateInput,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup) (*api.LoadbalancerListenerCreateInput, error) {
	input.Scheduler = api.LB_SCHEDULER_RR
	input.AclStatus = api.LB_BOOL_OFF
	input.StickySession = api.LB_BOOL_OFF
	return input, nil
}

func (self *SAwsRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput) (*api.LoadbalancerListenerUpdateInput, error) {
	return input, nil
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerRuleCreateInput) (*api.LoadbalancerListenerRuleCreateInput, error) {
	segs := []string{}
	if len(input.Path) > 0 {
		segs = append(segs, fmt.Sprintf(`{"field":"path-pattern","pathPatternConfig":{"values":["%s"]},"values":["%s"]}`, input.Path, input.Path))
	}

	if len(input.Domain) > 0 {
		segs = append(segs, fmt.Sprintf(`{"field":"host-header","hostHeaderConfig":{"values":["%s"]},"values":["%s"]}`, input.Domain, input.Domain))
	}
	input.Condition = fmt.Sprintf(`[%s]`, strings.Join(segs, ","))
	err := models.ValidateListenerRuleConditions(input.Condition)
	if err != nil {
		return nil, httperrors.NewInputParameterError("%s", err)
	}
	return input, nil
}

func (self *SAwsRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LoadbalancerListenerRuleUpdateInput) (*api.LoadbalancerListenerRuleUpdateInput, error) {
	return input, nil
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup,
	input *api.LoadbalancerBackendCreateInput) (*api.LoadbalancerBackendCreateInput, error) {
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerBackendData(ctx, userCred, lb, lbbg, input)
}

func (self *SAwsRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, input *api.LoadbalancerBackendUpdateInput) (*api.LoadbalancerBackendUpdateInput, error) {
	// 不能更新端口和权重
	if input.Port != nil {
		return input, fmt.Errorf("can not update backend port.")
	}

	if input.Weight != nil {
		return input, fmt.Errorf("can not update backend weight.")
	}
	return input, nil
}

func (self *SAwsRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
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
			vpc, err := lb.GetVpc()
			if err != nil {
				return nil, errors.Wrapf(err, "GetVpc")
			}
			lbbgOpts := &cloudprovider.SLoadbalancerBackendGroup{
				Name:       lbbg.Name,
				Scheduler:  lblis.Scheduler,
				Protocol:   lblis.ListenerType,
				ListenPort: lblis.ListenerPort,
				VpcId:      vpc.ExternalId,
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

func (self *SAwsRegionDriver) RequestStartLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SAwsRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SAwsRegionDriver) IsCertificateBelongToRegion() bool {
	return false
}

func (self *SAwsRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	cidrV := validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(ctx, jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
		return input, err
	}
	if cidrV.Value.MaskLen < 16 || cidrV.Value.MaskLen > 28 {
		return input, httperrors.NewInputParameterError("%s request the mask range should be between 16 and 28", self.GetProvider())
	}
	return input, nil
}

func (self *SAwsRegionDriver) RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip, input api.ElasticipAssociateInput, obj db.IStatusStandaloneModel, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iEip, err := eip.GetIEip(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "eip.GetIEip")
		}

		conf := &cloudprovider.AssociateConfig{
			InstanceId:    input.InstanceExternalId,
			Bandwidth:     eip.Bandwidth,
			AssociateType: api.EIP_ASSOCIATE_TYPE_SERVER,
		}

		err = iEip.Associate(conf)
		if err != nil {
			return nil, errors.Wrapf(err, "iEip.Associate")
		}

		err = cloudprovider.WaitStatus(iEip, api.EIP_STATUS_READY, 3*time.Second, 60*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.WaitStatus")
		}

		if obj.GetStatus() != api.INSTANCE_ASSOCIATE_EIP {
			db.StatusBaseSetStatus(ctx, obj, userCred, api.INSTANCE_ASSOCIATE_EIP, "associate eip")
		}

		err = eip.AssociateInstance(ctx, userCred, input.InstanceType, obj)
		if err != nil {
			return nil, errors.Wrapf(err, "eip.AssociateVM")
		}

		if input.InstanceType == api.EIP_ASSOCIATE_TYPE_SERVER {
			// 如果aws已经绑定了EIP，则要把多余的公有IP删除
			if iEip.GetMode() == api.EIP_MODE_STANDALONE_EIP {
				server := obj.(*models.SGuest)
				publicIP, err := server.GetPublicIp()
				if err != nil {
					return nil, errors.Wrap(err, "AwsGuestDriver.GetPublicIp")
				}

				if publicIP != nil {
					err = db.DeleteModel(ctx, userCred, publicIP)
					if err != nil {
						return nil, errors.Wrap(err, "AwsGuestDriver.DeletePublicIp")
					}
				}
			}
		}

		eip.SetStatus(ctx, userCred, api.EIP_STATUS_READY, "associate")
		return nil, nil
	})
	return nil
}

func (self *SAwsRegionDriver) ValidateCreateWafInstanceData(ctx context.Context, userCred mcclient.TokenCredential, input api.WafInstanceCreateInput) (api.WafInstanceCreateInput, error) {
	if len(input.Type) == 0 {
		input.Type = cloudprovider.WafTypeRegional
	}
	switch input.Type {
	case cloudprovider.WafTypeRegional:
	case cloudprovider.WafTypeCloudFront:
		_region, err := models.CloudregionManager.FetchById(input.CloudregionId)
		if err != nil {
			return input, err
		}
		region := _region.(*models.SCloudregion)
		if !strings.HasSuffix(region.ExternalId, "us-east-1") {
			return input, httperrors.NewUnsupportOperationError("only us-east-1 support %s", input.Type)
		}
	default:
		return input, httperrors.NewInputParameterError("Invalid aws waf type %s", input.Type)
	}
	if input.DefaultAction == nil {
		input.DefaultAction = &cloudprovider.DefaultAction{
			Action: cloudprovider.WafActionAllow,
		}
	}
	return input, nil
}

func (self *SAwsRegionDriver) ValidateCreateWafRuleData(ctx context.Context, userCred mcclient.TokenCredential, waf *models.SWafInstance, input api.WafRuleCreateInput) (api.WafRuleCreateInput, error) {
	return input, nil
}

func (self *SAwsRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		if input.Rules[i].Action != string(secrules.SecurityRuleAllow) {
			return nil, httperrors.NewInputParameterError("invalid action %s, only support allow", input.Rules[i].Action)
		}

		if len(input.Rules[i].Ports) > 0 && strings.Contains(input.Rules[i].Ports, ",") {
			return nil, httperrors.NewInputParameterError("invalid ports %s", input.Rules[i].Ports)
		}

	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateSecurityGroupInput(ctx, userCred, input)
}

func (self *SAwsRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Action != nil && *input.Action != string(secrules.SecurityRuleAllow) {
		return nil, httperrors.NewInputParameterError("invalid action %s", *input.Action)
	}

	if input.Ports != nil && strings.Contains(*input.Ports, ",") {
		return nil, httperrors.NewInputParameterError("invalid ports %s", *input.Ports)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateSecurityGroupRuleInput(ctx, userCred, input)
}

func (self *SAwsRegionDriver) RequestRemoteUpdateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := elasticcache.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "elasticcache.GetIRegion")
		}

		iElasticcache, err := iRegion.GetIElasticcacheById(elasticcache.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIElasticcacheById(%s)", elasticcache.ExternalId)
		}

		oldTags, err := iElasticcache.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "iElasticcache.GetTags()")
		}
		tags, err := elasticcache.GetAllUserMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
		mangerId := ""
		if vpc, _ := elasticcache.GetVpc(); vpc != nil {
			mangerId = vpc.ManagerId
		}
		err = cloudprovider.SetTags(ctx, iElasticcache, mangerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}

			logclient.AddActionLogWithStartable(task, elasticcache, logclient.ACT_UPDATE_TAGS, err, userCred, false)
			return nil, errors.Wrap(err, "iElasticcache.SetTags")
		}
		cloudprovider.WaitMultiStatus(iElasticcache, []string{api.ELASTIC_CACHE_STATUS_RUNNING}, 15*time.Second, 2*time.Minute)
		logclient.AddActionLogWithStartable(task, elasticcache, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, userCred, true)
		return nil, nil
	})
	return nil
}
