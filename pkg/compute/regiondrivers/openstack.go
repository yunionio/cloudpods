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
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SOpenStackRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SOpenStackRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SOpenStackRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return true
}

func (self *SOpenStackRegionDriver) GenerateSecurityGroupName(name string) string {
	if strings.ToLower(name) == "default" {
		return "DefaultGroup"
	}
	return name
}

func (self *SOpenStackRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SOpenStackRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:deny any")}
}

func (self *SOpenStackRegionDriver) GetSecurityGroupRuleMaxPriority() int {
	return 0
}

func (self *SOpenStackRegionDriver) GetSecurityGroupRuleMinPriority() int {
	return 0
}

func (self *SOpenStackRegionDriver) IsOnlySupportAllowRules() bool {
	return true
}

func (self *SOpenStackRegionDriver) GetSecurityGroupPublicScope(service string) rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (self *SOpenStackRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackRegionDriver) IsVpcCreateNeedInputCidr() bool {
	return false
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	//zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", ownerId)
	addressTypeV := validators.NewStringChoicesValidator("address_type", api.LB_ADDR_TYPES)
	networkV := validators.NewModelIdOrNameValidator("network", "network", ownerId)

	keyV := map[string]validators.IValidator{
		"status":       validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),
		"address_type": addressTypeV.Default(api.LB_ADDR_TYPE_INTRANET),
		"network":      networkV,
		//"zone":         zoneV,
		"manager": managerIdV,
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	// 检查网络可用
	network := networkV.Model.(*models.SNetwork)
	_, _, vpc, _, err := network.ValidateElbNetwork(nil)
	if err != nil {
		return nil, err
	}

	if managerIdV.Model.GetId() != vpc.ManagerId {
		return nil, httperrors.NewInputParameterError("Loadbalancer's manager (%s(%s)) does not match vpc's(%s(%s)) (%s)", managerIdV.Model.GetName(), managerIdV.Model.GetId(), vpc.GetName(), vpc.GetId(), vpc.ManagerId)
	}

	// 公网ELB需要指定EIP
	if addressTypeV.Value == api.LB_ADDR_TYPE_INTERNET {
		eipV := validators.NewModelIdOrNameValidator("eip", "eip", nil)
		if err := eipV.Validate(data); err != nil {
			return nil, err
		}

		eip := eipV.Model.(*models.SElasticip)
		if eip.Status != api.EIP_STATUS_READY {
			return nil, fmt.Errorf("eip status not ready")
		}

		if len(eip.ExternalId) == 0 {
			return nil, fmt.Errorf("eip external id is empty")
		}

		data.Set("eip_id", jsonutils.NewString(eip.ExternalId))
	}

	// region := zoneV.Model.(*models.SZone).GetRegion()
	region, _ := networkV.Model.(*models.SNetwork).GetRegion()
	if region == nil {
		return nil, fmt.Errorf("getting region failed")
	}

	// data.Set("network_type", jsonutils.NewString(api.LB_NETWORK_TYPE_VPC))
	data.Set("cloudregion_id", jsonutils.NewString(region.GetId()))
	data.Set("vpc_id", jsonutils.NewString(vpc.GetId()))
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, ownerId, data)
}

func (self *SOpenStackRegionDriver) RequestCreateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion(ctx)
		if err != nil {
			return nil, err
		}

		params, err := lb.GetCreateLoadbalancerParams(ctx, iRegion)
		if err != nil {
			return nil, err
		}

		Scloudprovider := lb.GetCloudprovider()
		params.ProjectId, err = Scloudprovider.SyncProject(ctx, userCred, lb.ProjectId)
		if err != nil {
			log.Errorf("failed to sync project %s for create %s lb %s error: %v", lb.ProjectId, Scloudprovider.Provider, lb.Name, err)
		}

		iLoadbalancer, err := iRegion.CreateILoadBalancer(params)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lb, userCred, iLoadbalancer.GetGlobalId()); err != nil {
			return nil, err
		}
		//wait async create result
		err = cloudprovider.WaitMultiStatus(iLoadbalancer, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_UNKNOWN}, 10*time.Second, 8*time.Minute)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.WaitMultiStatus")
		}
		if iLoadbalancer.GetStatus() == api.LB_STATUS_UNKNOWN {
			return nil, errors.Wrap(fmt.Errorf("status error"), "check status")
		}

		region, _ := lb.GetRegion()
		if err := lb.SyncWithCloudLoadbalancer(ctx, userCred, iLoadbalancer, nil, lb.GetCloudprovider(), region); err != nil {
			return nil, err
		}

		// 公网lb,需要同步public ip
		if lb.AddressType == api.LB_ADDR_TYPE_INTERNET {
			publicIp, err := iLoadbalancer.GetIEIP()
			if err != nil {
				return nil, errors.Wrap(err, "iLoadbalancer.GetIEIP()")
			}
			lb.SyncLoadbalancerEip(ctx, userCred, lb.GetCloudprovider(), publicIp)
		}

		lbbgs, err := iLoadbalancer.GetILoadBalancerBackendGroups()
		if err != nil {
			return nil, err
		}
		if len(lbbgs) > 0 {
			provider := lb.GetCloudprovider()
			if provider == nil {
				return nil, fmt.Errorf("failed to find cloudprovider for lb %s", lb.Name)
			}
			models.LoadbalancerBackendGroupManager.SyncLoadbalancerBackendgroups(ctx, userCred, provider, lb, lbbgs, &models.SSyncRange{})
		}
		return nil, nil
	})
	return nil
}

func (self *SOpenStackRegionDriver) ValidateDeleteLoadbalancerCondition(ctx context.Context, lb *models.SLoadbalancer) error {
	listeners, err := lb.GetLoadbalancerListeners()
	if err != nil {
		return err
	}

	if len(listeners) > 0 {
		return httperrors.NewConflictError("loadbalancer is using by %d listener.", len(listeners))
	}

	lbbgs, err := lb.GetLoadbalancerBackendgroups()
	if err != nil {
		return err
	}

	if len(lbbgs) > 0 {
		return httperrors.NewConflictError("loadbalancer is using by %d backendgroup.", len(lbbgs))
	}
	return nil
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// 访问控制： 与listener是1v1的
	// 关系，创建时即需要与具体的listener绑定，不能再变更listner。
	// required: listener_id, acl_type: "white", acl_status: "on", manager,cloudregion,acl_entries
	data, err := self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerAclData(ctx, userCred, data)
	if err != nil {
		return data, err
	}

	listenerV := validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", nil)
	err = listenerV.Validate(data)
	if err != nil {
		return data, err
	}

	return data, nil
}

func (self *SOpenStackRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.createLoadbalancerAcl(ctx, userCred, lbacl)
	})
	return nil
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer certificate", self.GetProvider())
}

func (self *SOpenStackRegionDriver) ValidateCreateEipData(ctx context.Context, userCred mcclient.TokenCredential, input *api.SElasticipCreateInput) error {
	if len(input.NetworkId) == 0 {
		return httperrors.NewMissingParameterError("network_id")
	}
	_network, err := models.NetworkManager.FetchByIdOrName(userCred, input.NetworkId)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperrors.NewResourceNotFoundError2("network", input.NetworkId)
		}
		return httperrors.NewGeneralError(err)
	}
	network := _network.(*models.SNetwork)
	input.NetworkId = network.Id

	vpc, _ := network.GetVpc()
	if vpc == nil {
		return httperrors.NewInputParameterError("failed to found vpc for network %s(%s)", network.Name, network.Id)
	}
	input.ManagerId = vpc.ManagerId
	region, err := vpc.GetRegion()
	if err != nil {
		return err
	}
	if region.Id != input.CloudregionId {
		return httperrors.NewUnsupportOperationError("network %s(%s) does not belong to %s", network.Name, network.Id, self.GetProvider())
	}
	return nil
}

func (self *SOpenStackRegionDriver) RequestPullLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localLoadbalancer *models.SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *models.SSyncRange) error {
	models.SyncOpenstackLoadbalancerBackendgroups(ctx, userCred, syncResults, provider, localLoadbalancer, remoteLoadbalancer, syncRange)
	return nil
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	listenerTypeV := validators.NewStringChoicesValidator("listener_type", api.LB_LISTENER_TYPES)
	listenerPortV := validators.NewPortValidator("listener_port")
	aclStatusV := validators.NewStringChoicesValidator("acl_status", api.LB_BOOL_VALUES)
	aclTypeV := validators.NewStringChoicesValidator("acl_type", api.LB_ACL_TYPES)
	aclV := validators.NewModelIdOrNameValidator("acl", "loadbalanceracl", ownerId)
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),

		"listener_type": listenerTypeV,
		"listener_port": listenerPortV,

		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Default(api.LB_SENDPROXY_OFF),
		"acl_status": aclStatusV.Default(api.LB_BOOL_OFF),
		"acl_type":   aclTypeV.Optional(true),
		"acl":        aclV.Optional(true),
		"scheduler":  validators.NewStringChoicesValidator("scheduler", api.LB_SCHEDULER_TYPES),

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES).Default(api.LB_BOOL_OFF),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", api.LB_STICKY_SESSION_TYPES).Default(api.LB_STICKY_SESSION_TYPE_INSERT),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)).Optional(true),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout").Optional(true),

		"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for").Default(true),
		// "gzip":            validators.NewBoolValidator("gzip").Default(false),
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	// listener uniqueness
	listenerType := listenerTypeV.Value
	err := models.LoadbalancerListenerManager.CheckListenerUniqueness(ctx, lb, listenerType, listenerPortV.Value)
	if err != nil {
		return nil, err
	}

	// backendgroup check
	if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != lb.Id {
		return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
			lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, lb.Id)
	}

	// https additional certificate check
	if listenerType == api.LB_LISTENER_TYPE_TERMINATED_HTTPS {
		certV := validators.NewModelIdOrNameValidator("certificate", "loadbalancercertificate", ownerId)
		tlsCipherPolicyV := validators.NewStringChoicesValidator("tls_cipher_policy", api.LB_TLS_CIPHER_POLICIES).Default(api.LB_TLS_CIPHER_POLICY_1_2)
		httpsV := map[string]validators.IValidator{
			"certificate":       certV,
			"tls_cipher_policy": tlsCipherPolicyV,
			//"enable_http2":      validators.NewBoolValidator("enable_http2").Default(true),
		}

		if err := RunValidators(httpsV, data, false); err != nil {
			return nil, err
		}
	}

	// health check default depends on input parameters
	checkTypeV := models.LoadbalancerListenerManager.CheckTypeV(listenerType)
	keyVHealth := map[string]validators.IValidator{
		"health_check":      validators.NewStringChoicesValidator("health_check", api.LB_BOOL_VALUES).Default(api.LB_BOOL_ON),
		"health_check_type": checkTypeV,

		"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true).Default(""),
		"health_check_path":      validators.NewURLPathValidator("health_check_path").Default(""),
		"health_check_uri":       validators.NewURLPathValidator("health_check_uri").Default(""),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 10).Default(3),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 50).Default(5),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Default(10),
	}

	if err := RunValidators(keyVHealth, data, false); err != nil {
		return nil, err
	}

	if t, _ := data.Int("health_check_rise"); t > 0 {
		data.Set("health_check_fall", jsonutils.NewInt(t))
	}

	interval, _ := data.Int("health_check_interval")
	timeout, _ := data.Int("health_check_timeout")
	if timeout >= interval {
		data.Set("health_check_interval", jsonutils.NewInt(interval+timeout))
	}

	// acl check
	if err := models.LoadbalancerListenerManager.ValidateAcl(aclStatusV, aclTypeV, aclV, data, api.CLOUD_PROVIDER_OPENSTACK); err != nil {
		return nil, err
	}

	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerListenerData(ctx, userCred, ownerId, data, lb, backendGroup)
}

func (self *SOpenStackRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		{
			certId, _ := task.GetParams().GetString("certificate_id")
			if len(certId) > 0 {
				provider := lblis.GetCloudprovider()
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				cert, err := models.LoadbalancerCertificateManager.FetchById(certId)
				if err != nil {
					return nil, errors.Wrapf(err, "LoadbalancerCertificateManager.FetchById(%s)", certId)
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, errors.Wrap(err, "CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate")
				}

				if len(lbcert.ExternalId) == 0 {
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, errors.Wrap(err, "createLoadbalancerCertificate")
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedCertificateId = lbcert.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestCreateLoadbalancerListener.UpdateCachedCertificateId")
				}
			}
		}

		params, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrapf(err, "lblis.GetLoadbalancerListenerParams")
		}
		loadbalancer, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "loadbalancer.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "iRegion.GetILoadBalancerById(%s)", loadbalancer.ExternalId)
		}
		iListener, err := iLoadbalancer.CreateILoadBalancerListener(ctx, params)
		if err != nil {
			return nil, errors.Wrap(err, "iLoadbalancer.CreateILoadBalancerListener")
		}
		if err := db.SetExternalId(lblis, userCred, iListener.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "db.SetExternalId")
		}
		// wait async result
		err = cloudprovider.WaitMultiStatus(iListener, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_UNKNOWN}, 10*time.Second, 8*time.Minute)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.WaitMultiStatus")
		}
		if iListener.GetStatus() == api.LB_STATUS_UNKNOWN {
			return nil, errors.Wrap(fmt.Errorf("status error"), "check status")
		}

		{
			aclId, _ := task.GetParams().GetString("acl_id")
			if len(aclId) > 0 {
				provider := lblis.GetCloudprovider()
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				acl, err := models.LoadbalancerAclManager.FetchById(aclId)
				if err != nil {
					return nil, errors.Wrap(err, "LoadbalancerAclManager.FetchById")
				}

				lbacl, err := models.CachedLoadbalancerAclManager.GetOrCreateCachedAcl(ctx, userCred, provider, lblis, acl.(*models.SLoadbalancerAcl))
				if err != nil {
					return nil, errors.Wrap(err, "CachedLoadbalancerAclManager.GetOrCreateCachedAcl")
				}

				if len(lbacl.ExternalId) == 0 {
					_, err = self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, errors.Wrap(err, "createLoadbalancerAcl")
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedAclId = lbacl.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestCreateLoadbalancerListener.UpdateCachedAclId")
				}
			}
		}

		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, loadbalancer.GetOwnerId(), lblis.GetCloudprovider())
	})
	return nil
}

func (self *SOpenStackRegionDriver) createLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl) (jsonutils.JSONObject, error) {
	iRegion, err := lbacl.GetIRegion(ctx)
	if err != nil {
		return nil, err
	}

	listener, err := lbacl.GetListener()
	if err != nil {
		return nil, err
	}

	acl := &cloudprovider.SLoadbalancerAccessControlList{
		ListenerId:          listener.GetExternalId(),
		Name:                lbacl.Name,
		Entrys:              []cloudprovider.SLoadbalancerAccessControlListEntry{},
		AccessControlEnable: listener.AclStatus == api.LB_BOOL_ON,
	}

	_originAcl, err := db.FetchById(models.LoadbalancerAclManager, lbacl.AclId)
	if err != nil {
		return nil, errors.Wrap(err, "regionDriver.FetchAcl")
	}

	originAcl := _originAcl.(*models.SLoadbalancerAcl)
	if originAcl.AclEntries != nil {
		for _, entry := range *originAcl.AclEntries {
			acl.Entrys = append(acl.Entrys, cloudprovider.SLoadbalancerAccessControlListEntry{CIDR: entry.Cidr, Comment: entry.Comment})
		}
	}
	iLoadbalancerAcl, err := iRegion.CreateILoadBalancerAcl(acl)
	if err != nil {
		return nil, err
	}

	lbacl.SetModelManager(models.CachedLoadbalancerAclManager, lbacl)
	if err := db.SetExternalId(lbacl, userCred, iLoadbalancerAcl.GetGlobalId()); err != nil {
		return nil, err
	}
	return nil, lbacl.SyncWithCloudLoadbalancerAcl(ctx, userCred, iLoadbalancerAcl, lbacl.GetOwnerId())
}

func (self *SOpenStackRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lblis *models.SLoadbalancerListener, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	if lblis.Status == api.LB_SYNC_CONF {
		return nil, httperrors.NewResourceNotFoundError("loadbalancer listener %s is already updating", lblis.Name)
	}
	ownerId := lblis.GetOwnerId()
	// todo: fix me here
	aclStatusV := validators.NewStringChoicesValidator("acl_status", api.LB_BOOL_VALUES)
	aclStatusV.Default(lblis.AclStatus)
	aclTypeV := validators.NewStringChoicesValidator("acl_type", api.LB_ACL_TYPES)
	if api.LB_ACL_TYPES.Has(lblis.AclType) {
		aclTypeV.Default(lblis.AclType)
	}
	aclV := validators.NewModelIdOrNameValidator("acl", "loadbalanceracl", ownerId)
	if len(lblis.AclId) > 0 {
		aclV.Default(lblis.AclId)
	}

	certV := validators.NewModelIdOrNameValidator("certificate", "loadbalancercertificate", ownerId)
	tlsCipherPolicyV := validators.NewStringChoicesValidator("tls_cipher_policy", api.LB_TLS_CIPHER_POLICIES).Default(api.LB_TLS_CIPHER_POLICY_1_2)
	keyV := map[string]validators.IValidator{
		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES),
		"scheduler":  validators.NewStringChoicesValidator("scheduler", api.LB_SCHEDULER_TYPES),

		"acl_status": aclStatusV,
		"acl_type":   aclTypeV,
		"acl":        aclV,

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", api.LB_STICKY_SESSION_TYPES),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout"),

		"health_check":      validators.NewStringChoicesValidator("health_check", api.LB_BOOL_VALUES),
		"health_check_type": models.LoadbalancerListenerManager.CheckTypeV(lblis.ListenerType),

		"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true).Default(""),
		"health_check_path":      validators.NewURLPathValidator("health_check_path").Default(""),
		"health_check_uri":       validators.NewURLPathValidator("health_check_uri").Default(""),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 10).Default(3),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 50).Default(10),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Default(5),

		"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for"),

		"certificate":       certV,
		"tls_cipher_policy": tlsCipherPolicyV,
	}

	if err := RunValidators(keyV, data, true); err != nil {
		return nil, err
	}

	if t, _ := data.Int("health_check_rise"); t > 0 {
		data.Set("health_check_fall", jsonutils.NewInt(t))
	}

	{
		if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != lblis.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, lblis.LoadbalancerId)
		}
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateLoadbalancerListenerData(ctx, userCred, data, lblis, backendGroup)
}

func (self *SOpenStackRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		{
			certId, _ := task.GetParams().GetString("certificate_id")
			if len(certId) > 0 {
				provider := lblis.GetCloudprovider()
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				cert, err := models.LoadbalancerCertificateManager.FetchById(certId)
				if err != nil {
					return nil, err
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, err
				}

				if len(lbcert.ExternalId) == 0 {
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, err
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedCertificateId = lbcert.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerListener.UpdateCachedCertificateId")
				}
			}
		}

		params, err := lblis.GetOpenstackLoadbalancerListenerParams()
		if err != nil {
			return nil, err
		}
		loadbalancer, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, err
		}
		if err := iListener.Sync(ctx, params); err != nil {
			return nil, err
		}

		{
			aclId, _ := task.GetParams().GetString("acl_id")
			if len(aclId) > 0 {
				provider := lblis.GetCloudprovider()
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				var lbacl *models.SCachedLoadbalancerAcl
				// 先读取缓存，缓存不存在的情况下，从ACL表中取数据创建缓存
				if _lbacl, err := models.CachedLoadbalancerAclManager.FetchById(aclId); err == nil && _lbacl != nil {
					lbacl = _lbacl.(*models.SCachedLoadbalancerAcl)
				} else {
					acl, err := models.LoadbalancerAclManager.FetchById(aclId)
					if err != nil {
						return nil, err
					}

					lbacl, err = models.CachedLoadbalancerAclManager.GetOrCreateCachedAcl(ctx, userCred, provider, lblis, acl.(*models.SLoadbalancerAcl))
					if err != nil {
						return nil, err
					}
				}

				if len(lbacl.ExternalId) == 0 {
					_, err = self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, err
					}
				} else {
					_, err = self.syncLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, err
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedAclId = lbacl.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerListener.UpdateCachedAclId")
				}
			}
		}

		if err := iListener.Refresh(); err != nil {
			return nil, err
		}
		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, lblis.GetOwnerId(), lblis.GetCloudprovider())
	})
	return nil
}

func (self *SOpenStackRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		loadbalancer, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
		if err != nil {
			return nil, err
		}

		if len(loadbalancer.ExternalId) == 0 {
			return nil, nil
		}

		if len(lblis.ExternalId) == 0 {
			return nil, nil
		}

		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}

			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}

		// 取消服务器组关联
		backendgroupId := iListener.GetBackendGroupId()
		if len(backendgroupId) > 0 {
			// 删除后端服务器组
			ilbbg, err := iLoadbalancer.GetILoadBalancerBackendGroupById(backendgroupId)
			if err != nil {
				return nil, errors.Wrap(err, "OpenStackRegionDriver.RequestDeleteLoadbalancerListener.GetILoadBalancerBackendGroup")
			}

			err = deleteOpenstackLoadbalancerBackendGroup(ctx, userCred, iLoadbalancer, ilbbg)
			if err != nil {
				return nil, errors.Wrap(err, "OpenStackRegionDriver.RequestDeleteLoadbalancerListener.DeleteBackendGroup")
			}
		}

		err = deleteOpenstackCachedLbbg(ctx, userCred, lblis.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "OpenstackRegionDriver.RequestDeleteLoadbalancerListener.deleteOpenstackCachedLbbg")
		}

		// 删除访问控制
		aclId := iListener.GetAclId()
		if len(aclId) > 0 {
			iAcl, err := iRegion.GetILoadBalancerAclById(aclId)
			if err != nil {
				return nil, err
			}

			err = iAcl.Delete()
			if err != nil {
				return nil, err
			}

			acl := lblis.GetCachedLoadbalancerAcl()
			if acl != nil {
				err := db.DeleteModel(ctx, userCred, acl)
				if err != nil {
					return nil, err
				}
			}
		}

		// remove rules
		irules, err := iListener.GetILoadbalancerListenerRules()
		if err != nil {
			return nil, errors.Wrap(err, "OpenStackRegionDriver.GetILoadbalancerListenerRules")
		}

		for i := range irules {
			irule := irules[i]
			err = deleteOpenstackLoadbalancerListenerRule(ctx, userCred, iLoadbalancer, irule)
			if err != nil {
				return nil, errors.Wrap(err, "OpenStackRegionDriver.deleteOpenStackLoadbalancerListenerRule")
			}
		}

		rules, err := lblis.GetLoadbalancerListenerRules()
		if err != nil && err != sql.ErrNoRows {
			return nil, errors.Wrap(err, "OpenStackRegionDriver.GetLoadbalancerListenerRules")
		}

		for i := range rules {
			rule := rules[i]
			if err := deleteOpenstackLblisRule(ctx, userCred, rule.GetId()); err != nil {
				return nil, errors.Wrap(err, "OpenStackRegionDriver.Rule.deleteOpenstackCachedLbbg")
			}
		}

		return nil, iListener.Delete(ctx)
	})
	return nil
}

func (self *SOpenStackRegionDriver) syncLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl) (jsonutils.JSONObject, error) {
	iRegion, err := lbacl.GetIRegion(ctx)
	if err != nil {
		return nil, err
	}

	acl := &cloudprovider.SLoadbalancerAccessControlList{
		Name:   lbacl.Name,
		Entrys: []cloudprovider.SLoadbalancerAccessControlListEntry{},
	}

	lblis, err := lbacl.GetListener()
	if err == nil {
		if api.LB_BOOL_ON == lblis.AclStatus {
			acl.AccessControlEnable = true
		}
	} else {
		return nil, fmt.Errorf("SOpenStackRegionDriver.syncLoadbalancerAcl %s", err)
	}

	_localAcl, err := db.FetchById(models.LoadbalancerAclManager, lbacl.AclId)
	if err != nil {
		return nil, errors.Wrap(err, "regionDriver.FetchById.LoaclAcl")
	}

	localAcl := _localAcl.(*models.SLoadbalancerAcl)
	if localAcl.AclEntries != nil {
		for _, entry := range *localAcl.AclEntries {
			acl.Entrys = append(acl.Entrys, cloudprovider.SLoadbalancerAccessControlListEntry{CIDR: entry.Cidr, Comment: entry.Comment})
		}
	}

	lockman.LockRawObject(ctx, "acl", lbacl.Id)
	defer lockman.ReleaseRawObject(ctx, "acl", lbacl.Id)

	iLoadbalancerAcl, err := iRegion.GetILoadBalancerAclById(lbacl.ExternalId)
	if err != nil {
		return nil, err
	}
	return nil, iLoadbalancerAcl.Sync(acl)
}

func (self *SOpenStackRegionDriver) IsSupportLoadbalancerListenerRuleRedirect() bool {
	return true
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	var (
		listenerV = validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", ownerId)
		domainV   = validators.NewHostPortValidator("domain").OptionalPort(true)
		pathV     = validators.NewURLPathValidator("path")

		redirectV       = validators.NewStringChoicesValidator("redirect", api.LB_REDIRECT_TYPES)
		redirectCodeV   = validators.NewIntChoicesValidator("redirect_code", api.LB_REDIRECT_CODES)
		redirectSchemeV = validators.NewStringChoicesValidator("redirect_scheme", api.LB_REDIRECT_SCHEMES)
		redirectHostV   = validators.NewHostPortValidator("redirect_host").OptionalPort(true)
		redirectPathV   = validators.NewURLPathValidator("redirect_path")
	)
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),

		"listener": listenerV,
		"domain":   domainV.AllowEmpty(true).Default(""),
		"path":     pathV.Default(""),

		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate").Default(0),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src").Default(0),

		"redirect":        redirectV.Default(api.LB_REDIRECT_OFF),
		"redirect_code":   redirectCodeV.Default(api.LB_REDIRECT_CODE_302),
		"redirect_scheme": redirectSchemeV.Optional(true),
		"redirect_host":   redirectHostV.AllowEmpty(true).Optional(true),
		"redirect_path":   redirectPathV.AllowEmpty(true).Optional(true),
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	listener := listenerV.Model.(*models.SLoadbalancerListener)
	listenerType := listener.ListenerType
	if listenerType != api.LB_LISTENER_TYPE_HTTP && listenerType != api.LB_LISTENER_TYPE_HTTPS {
		return nil, httperrors.NewInputParameterError("listener type must be http/https, got %s", listenerType)
	}

	redirectType := redirectV.Value
	if redirectType != api.LB_REDIRECT_OFF {
		if redirectType == api.LB_REDIRECT_RAW {
			scheme, host, path := redirectSchemeV.Value, redirectHostV.Value, redirectPathV.Value
			if (scheme == "" || scheme == listenerType) && host == "" && path == "" {
				return nil, httperrors.NewInputParameterError("redirect must have at least one of scheme, host, path changed")
			}
			if scheme == "" {
				data.Set("redirect_scheme", jsonutils.NewString(listenerType))
			}
			if host == "" {
				data.Set("redirect_host", jsonutils.NewString(domainV.Value))
			}
		}
	}

	{
		if redirectV.Value == api.LB_REDIRECT_OFF {
			if backendGroup == nil {
				return nil, httperrors.NewInputParameterError("backend_group argument is missing")
			}
		}
		if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != listener.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, listener.LoadbalancerId)
		}
	}

	err := models.LoadbalancerListenerRuleCheckUniqueness(ctx, listener, domainV.Value, pathV.Value)
	if err != nil {
		return nil, err
	}

	data.Set("cloudregion_id", jsonutils.NewString(listener.GetRegionId()))
	data.Set("manager_id", jsonutils.NewString(listener.GetCloudproviderId()))
	return data, nil
}

func (self *SOpenStackRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	var (
		lbr     = ctx.Value("lbr").(*models.SLoadbalancerListenerRule)
		domainV = validators.NewHostPortValidator("domain").OptionalPort(true)
		pathV   = validators.NewURLPathValidator("path")

		redirectV       = validators.NewStringChoicesValidator("redirect", api.LB_REDIRECT_TYPES)
		redirectCodeV   = validators.NewIntChoicesValidator("redirect_code", api.LB_REDIRECT_CODES)
		redirectSchemeV = validators.NewStringChoicesValidator("redirect_scheme", api.LB_REDIRECT_SCHEMES)
		redirectHostV   = validators.NewHostPortValidator("redirect_host").OptionalPort(true)
		redirectPathV   = validators.NewURLPathValidator("redirect_path")
	)
	if lbr.Redirect != "" {
		redirectV.Default(lbr.Redirect)
	}
	if lbr.RedirectCode > 0 {
		redirectCodeV.Default(int64(lbr.RedirectCode))
	} else {
		redirectCodeV.Default(api.LB_REDIRECT_CODE_302)
	}
	if lbr.RedirectScheme != "" {
		redirectSchemeV.Default(lbr.RedirectScheme)
	}
	if lbr.RedirectHost != "" {
		redirectHostV.Default(lbr.RedirectHost)
	}
	if lbr.RedirectPath != "" {
		redirectPathV.Default(lbr.RedirectPath)
	}
	keyV := map[string]validators.IValidator{
		"domain": domainV.AllowEmpty(true).Default(lbr.Domain),
		"path":   pathV.Default(lbr.Path),

		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate"),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src"),

		"redirect":        redirectV,
		"redirect_code":   redirectCodeV,
		"redirect_scheme": redirectSchemeV,
		"redirect_host":   redirectHostV.AllowEmpty(true),
		"redirect_path":   redirectPathV.AllowEmpty(true),
	}
	for _, v := range keyV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}

	var (
		redirectType = redirectV.Value
	)
	if redirectType != api.LB_REDIRECT_OFF {
		if redirectType == api.LB_REDIRECT_RAW {
			var (
				lblis, _     = lbr.GetLoadbalancerListener()
				listenerType = lblis.ListenerType
			)
			scheme, host, path := redirectSchemeV.Value, redirectHostV.Value, redirectPathV.Value
			if (scheme == "" || scheme == listenerType) && host == "" && path == "" {
				return nil, httperrors.NewInputParameterError("redirect must have at least one of scheme, host, path changed")
			}
		}
	}

	if redirectType == api.LB_REDIRECT_OFF && backendGroup == nil {
		return nil, httperrors.NewInputParameterError("non redirect lblistener rule must have backend_group set")
	}
	if backendGroup, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && backendGroup.Id != lbr.BackendGroupId {
		listenerM, err := models.LoadbalancerListenerManager.FetchById(lbr.ListenerId)
		if err != nil {
			return nil, httperrors.NewInputParameterError("loadbalancerlistenerrule %s(%s): fetching listener %s failed",
				lbr.Name, lbr.Id, lbr.ListenerId)
		}
		listener := listenerM.(*models.SLoadbalancerListener)
		if backendGroup.LoadbalancerId != listener.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, listener.LoadbalancerId)
		}
	}
	return data, nil
}

func (self *SOpenStackRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		listener, err := lbr.GetLoadbalancerListener()
		if err != nil {
			return nil, err
		}
		loadbalancer, err := listener.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(listener.ExternalId)
		if err != nil {
			return nil, err
		}
		rule := &cloudprovider.SLoadbalancerListenerRule{
			Name:   lbr.Name,
			Domain: lbr.Domain,
			Path:   lbr.Path,

			Redirect:       lbr.Redirect,
			RedirectCode:   lbr.RedirectCode,
			RedirectScheme: lbr.RedirectScheme,
			RedirectHost:   lbr.RedirectHost,
			RedirectPath:   lbr.RedirectPath,
		}
		if len(lbr.BackendGroupId) > 0 {
			group := lbr.GetLoadbalancerBackendGroup()
			if group == nil {
				return nil, fmt.Errorf("failed to find backend group for listener rule %s", lbr.Name)
			}
			cachedLbbg, err := models.OpenstackCachedLbbgManager.GetCachedBackendGroupByAssociateId(lbr.GetId())
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestCreateLoadbalancerListenerRule.GetCachedBackendGroupByAssociateId")
			}

			if cachedLbbg == nil {
				return nil, fmt.Errorf("usable cached backend group not found")
			}

			rule.BackendGroupID = cachedLbbg.ExternalId
			rule.BackendGroupType = group.Type
		}
		iListenerRule, err := iListener.CreateILoadBalancerListenerRule(rule)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lbr, userCred, iListenerRule.GetGlobalId()); err != nil {
			return nil, err
		}
		return nil, lbr.SyncWithCloudLoadbalancerListenerRule(ctx, userCred, iListenerRule, listener.GetOwnerId(), loadbalancer.GetCloudprovider())
	})
	return nil
}

func (self *SOpenStackRegionDriver) RequestDeleteLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		listener, err := lbr.GetLoadbalancerListener()
		if err != nil {
			return nil, err
		}
		loadbalancer, err := listener.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(listener.ExternalId)
		if err != nil {
			return nil, err
		}
		if len(lbr.ExternalId) == 0 {
			return nil, nil
		}
		iListenerRule, err := iListener.GetILoadBalancerListenerRuleById(lbr.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		err = iListenerRule.Delete(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "iListenerRule.Delete")
		}
		//delete associated backendgroup
		cachedlbbgs, err := models.OpenstackCachedLbbgManager.GetListenerRuleCachedBackendGroups(lbr.GetId())
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, errors.Wrap(fmt.Errorf("provider is nil"), "OpenstackRegionDriver.RequestSyncLoadbalancerBackendGroup.GetCachedBackendGroupByAssociateId")
			}
		}
		for i := 0; i < len(cachedlbbgs); i++ {
			// delete outdated cache
			if err := deleteOpenstackCachedLbbsByLbbg(ctx, userCred, cachedlbbgs[i].GetId()); err != nil {
				return nil, errors.Wrap(err, "deleteOpenstackCachedLbbg.deleteOpenstackCachedLbbsByLbbg")
			}
			err = db.DeleteModel(ctx, userCred, &cachedlbbgs[i])
			if err != nil {
				return nil, errors.Wrap(err, "deleteOpenstackCachedLbbg.DeleteModel")
			}
			// delete ilbbg
			ilbbg, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedlbbgs[i].ExternalId)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					ilbbg = nil
				} else {
					return nil, errors.Wrapf(err, "GetILoadBalancerBackendGroupById(%s)", cachedlbbgs[i].ExternalId)
				}
			} else {
				err := ilbbg.Delete(ctx)
				if err != nil {
					return nil, errors.Wrap(err, "iLoadbalancerBackendGroup.Delete(ctx)")
				}
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SOpenStackRegionDriver) createCachedLbbg(ctx context.Context, lb *models.SLoadbalancer, lblis *models.SLoadbalancerListener, lbr *models.SLoadbalancerListenerRule, lbbg *models.SLoadbalancerBackendGroup) (*models.SOpenstackCachedLbbg, error) {
	// create loadbalancer backendgroup cache
	cachedLbbg := &models.SOpenstackCachedLbbg{}
	cachedLbbg.ManagerId = lb.GetCloudproviderId()
	cachedLbbg.CloudregionId = lb.GetRegionId()
	cachedLbbg.LoadbalancerId = lb.GetId()
	cachedLbbg.BackendGroupId = lbbg.GetId()
	if lbr != nil {
		cachedLbbg.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
		cachedLbbg.AssociatedId = lbr.GetId()
		cachedLbbg.ProtocolType = lblis.ListenerType
	} else {
		cachedLbbg.AssociatedType = api.LB_ASSOCIATE_TYPE_LISTENER
		cachedLbbg.AssociatedId = lblis.GetId()
		cachedLbbg.ProtocolType = lblis.ListenerType
	}

	err := models.OpenstackCachedLbbgManager.TableSpec().Insert(ctx, cachedLbbg)
	if err != nil {
		return nil, err
	}

	cachedLbbg.SetModelManager(models.OpenstackCachedLbbgManager, cachedLbbg)
	return cachedLbbg, nil
}

func (self *SOpenStackRegionDriver) syncCloudlbbs(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, cachedLbbg *models.SOpenstackCachedLbbg, extlbbg cloudprovider.ICloudLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend) error {
	ibackends, err := extlbbg.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "OpenstackRegionDriver.syncCloudLoadbalancerBackends.GetILoadbalancerBackends")
	}

	m_ibackends := make(map[string]cloudprovider.ICloudLoadbalancerBackend)
	m_backends := make(map[string]*cloudprovider.SLoadbalancerBackend)
	for _, v := range ibackends {
		m_ibackends[v.GetBackendId()] = v
	}

	for i, v := range backends {
		m_backends[v.ExternalID] = &backends[i]
	}

	for _, ibackend := range ibackends {
		mbackend, _ := m_backends[ibackend.GetBackendId()]
		if mbackend == nil || mbackend.Port != ibackend.GetPort() {
			err = extlbbg.RemoveBackendServer(ibackend.GetId(), ibackend.GetWeight(), ibackend.GetPort())
			if err != nil {
				return errors.Wrap(err, "OpenstackRegionDriver.syncCloudLoadbalancerBackends.RemoveBackendServer")
			}
		} else {
			if ibackend.GetWeight() != mbackend.Weight {
				ibackend.SyncConf(ctx, mbackend.Port, mbackend.Weight)
			}
		}
	}

	for _, backend := range backends {
		ibackend, _ := m_ibackends[backend.ExternalID]
		if ibackend == nil {
			_, err = extlbbg.AddBackendServer(backend.ExternalID, backend.Weight, backend.Port)
			if err != nil {
				return errors.Wrap(err, "OpenstackRegionDriver.syncCloudLoadbalancerBackends.AddBackendServer")
			}
		}
	}

	return nil
}

func (self *SOpenStackRegionDriver) syncCachedLbbs(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, lbbg *models.SOpenstackCachedLbbg, extlbbg cloudprovider.ICloudLoadbalancerBackendGroup) error {
	iBackends, err := extlbbg.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "OpenstackRegionDriver.syncLoadbalancerBackendCaches.GetILoadbalancerBackends")
	}

	if len(iBackends) > 0 {
		provider := lb.GetCloudprovider()
		if provider == nil {
			return fmt.Errorf("failed to find cloudprovider for lb %s", lb.Name)
		}

		result := models.OpenstackCachedLbManager.SyncLoadbalancerBackends(ctx, userCred, provider, lbbg, iBackends, &models.SSyncRange{})
		if result.IsError() {
			return errors.Wrap(result.AllError(), "OpenstackRegionDriver.syncLoadbalancerBackendCaches.SyncLoadbalancerBackends")
		}
	}

	return nil
}

func (self *SOpenStackRegionDriver) updateCachedLbbg(ctx context.Context, cachedLbbg *models.SOpenstackCachedLbbg, backendGroupId string, externalBackendGroupId string, asscoicateId string, asscoicateType string) error {
	_, err := db.UpdateWithLock(ctx, cachedLbbg, func() error {
		if len(backendGroupId) > 0 {
			cachedLbbg.BackendGroupId = backendGroupId
		}

		if len(externalBackendGroupId) > 0 {
			cachedLbbg.ExternalId = externalBackendGroupId
		}

		if len(asscoicateId) > 0 {
			cachedLbbg.AssociatedId = asscoicateId
		}

		if len(asscoicateType) > 0 {
			cachedLbbg.AssociatedType = asscoicateType
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (self *SOpenStackRegionDriver) removeCachedLbbg(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SOpenstackCachedLbbg) error {
	backends, err := lbbg.GetCachedBackends()
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "openstackRegionDriver.GetCachedBackends")
	}
	for i := range backends {
		err = db.DeleteModel(ctx, userCred, &backends[i])
		if err != nil {
			return errors.Wrap(err, "openstackRegionDriver.DeleteModel")
		}
	}

	err = db.DeleteModel(ctx, userCred, lbbg)
	if err != nil {
		return errors.Wrap(err, "openstackRegionDriver.DeleteModel")
	}

	return nil
}

func (self *SOpenStackRegionDriver) createLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, lbr *models.SLoadbalancerListenerRule, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend) (jsonutils.JSONObject, error) {
	if len(lblis.ListenerType) == 0 {
		return nil, fmt.Errorf("loadbalancer backendgroup missing protocol type")
	}

	iRegion, err := lbbg.GetIRegion(ctx)
	if err != nil {
		return nil, err
	}
	lb, err := lbbg.GetLoadbalancer()
	if err != nil {
		return nil, err
	}
	iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
	if err != nil {
		return nil, err
	}

	cachedLbbg, err := self.createCachedLbbg(ctx, lb, lblis, lbr, lbbg)
	if err != nil {
		return nil, errors.Wrap(err, "OpenstackRegionDriver.createLoadbalancerBackendGroupCache")
	}

	group, err := lbbg.GetOpenstackBackendGroupParams(lblis, nil)
	if err != nil {
		return nil, err
	}
	// 避免和defaultpool冲突
	if lbr != nil {
		group.ListenerID = ""
	}

	iLoadbalancerBackendGroup, err := iLoadbalancer.CreateILoadBalancerBackendGroup(group)
	if err != nil {
		return nil, err
	}

	if err := db.SetExternalId(cachedLbbg, userCred, iLoadbalancerBackendGroup.GetGlobalId()); err != nil {
		return nil, err
	}

	err = self.syncCloudlbbs(ctx, userCred, lb, cachedLbbg, iLoadbalancerBackendGroup, backends)
	if err != nil {
		return nil, errors.Wrap(err, "OpenstackRegionDriver.createLoadbalancerBackendGroup.syncCloudLoadbalancerBackends")
	}

	err = self.syncCachedLbbs(ctx, userCred, lb, cachedLbbg, iLoadbalancerBackendGroup)
	if err != nil {
		return nil, errors.Wrap(err, "OpenstackRegionDriver.createLoadbalancerBackendGroup.syncLoadbalancerBackendCaches")
	}

	return nil, nil

}

func deleteOpenstackLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, ilb cloudprovider.ICloudLoadbalancer, irule cloudprovider.ICloudLoadbalancerListenerRule) error {
	err := irule.Refresh()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}

		return errors.Wrap(err, "OpenStackRegionDriver.Rule.Refresh")
	}

	lbbgId := irule.GetBackendGroupId()

	err = irule.Delete(ctx)
	if err != nil && err != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "OpenStackRegionDriver.Rule.Delete")
	}

	// delete backendgroup
	if len(lbbgId) > 0 {
		ilbbg, err := ilb.GetILoadBalancerBackendGroupById(lbbgId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil
			}

			return errors.Wrap(err, "OpenStackRegionDriver.Rule.GetILoadBalancerBackendGroupById")
		}

		err = deleteOpenstackLoadbalancerBackendGroup(ctx, userCred, ilb, ilbbg)
		if err != nil {
			return errors.Wrap(err, "OpenStackRegionDriver.Rule.deleteOpenstackLoadbalancerBackendGroup")
		}
	}

	return nil
}

func deleteOpenstackLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, ilb cloudprovider.ICloudLoadbalancer, ilbbg cloudprovider.ICloudLoadbalancerBackendGroup) error {
	err := ilbbg.Refresh()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}

		return errors.Wrap(err, "OpenStackRegionDriver.BackendGroup.Refresh")
	}

	ibackends, err := ilbbg.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "OpenStackRegionDriver.BackendGroup.GetILoadbalancerBackends")
	}

	for i := range ibackends {
		ilbb := ibackends[i]
		err = deleteOpenstackLoadbalancerBackend(ctx, userCred, ilb, ilbbg, ilbb)
		if err != nil {
			return errors.Wrap(err, "OpenStackRegionDriver.BackendGroup.deleteOpenstackLoadbalancerBackend")
		}
	}

	err = ilbbg.Delete(ctx)
	if err != nil && err != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "OpenStackRegionDriver.BackendGroup.Delete")
	}

	return nil
}

func deleteOpenstackLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, ilb cloudprovider.ICloudLoadbalancer, ilbbg cloudprovider.ICloudLoadbalancerBackendGroup, ilbb cloudprovider.ICloudLoadbalancerBackend) error {
	err := ilbbg.RemoveBackendServer(ilbb.GetId(), ilbb.GetWeight(), ilbb.GetPort())
	if err != nil && err != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "OpenStackRegionDriver.Backend.Delete")
	}

	return nil
}

func deleteOpenstackLblisRule(ctx context.Context, userCred mcclient.TokenCredential, ruleId string) error {
	rule, err := db.FetchById(models.LoadbalancerListenerRuleManager, ruleId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		return errors.Wrap(err, "deleteOpenstackLblisRule.FetchById")
	}

	err = deleteOpenstackCachedLbbg(ctx, userCred, ruleId)
	if err != nil {
		return errors.Wrap(err, "deleteOpenstackLblisRule.deleteOpenstackCachedLbbg")
	}

	err = db.DeleteModel(ctx, userCred, rule)
	if err != nil {
		return errors.Wrap(err, "deleteOpenstackLblisRule.DeleteModel")
	}

	return nil
}

func deleteOpenstackCachedLbbg(ctx context.Context, userCred mcclient.TokenCredential, associatedId string) error {
	lbbg, err := models.OpenstackCachedLbbgManager.GetCachedBackendGroupByAssociateId(associatedId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		return errors.Wrap(err, "deleteOpenstackCachedLbbg.GetCachedBackendGroupByAssociateId")
	}

	if err := deleteOpenstackCachedLbbsByLbbg(ctx, userCred, lbbg.GetId()); err != nil {
		return errors.Wrap(err, "deleteOpenstackCachedLbbg.deleteOpenstackCachedLbbsByLbbg")
	}

	err = db.DeleteModel(ctx, userCred, lbbg)
	if err != nil {
		return errors.Wrap(err, "deleteOpenstackCachedLbbg.DeleteModel")
	}

	return nil
}

func deleteOpenstackCachedLbbsByLbbg(ctx context.Context, userCred mcclient.TokenCredential, cachedLbbgId string) error {
	cachedLbbs := []models.SOpenstackCachedLb{}
	q := models.OpenstackCachedLbManager.Query().IsFalse("pending_deleted").Equals("cached_backend_group_id", cachedLbbgId)
	err := db.FetchModelObjects(models.OpenstackCachedLbManager, q, &cachedLbbs)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		return errors.Wrap(err, "deleteOpenstackCachedLbbsByLbbg.FetchModelObjects")
	}

	for i := range cachedLbbs {
		cachedLbb := cachedLbbs[i]
		err = db.DeleteModel(ctx, userCred, &cachedLbb)
		if err != nil {
			return errors.Wrap(err, "deleteOpenstackCachedLbbsByLbbg.DeleteModel")
		}
	}

	return nil
}

func (self *SOpenStackRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	// 未指定listenerId或ruleId情况下，跳过远端创建步骤
	listenerId, _ := task.GetParams().GetString("listenerId")
	ruleId, _ := task.GetParams().GetString("ruleId")
	if len(listenerId) == 0 && len(ruleId) == 0 {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return nil, nil
		})
		return nil
	}

	var rule *models.SLoadbalancerListenerRule
	var listener *models.SLoadbalancerListener
	if len(ruleId) > 0 {
		_rule, err := db.FetchById(models.LoadbalancerListenerRuleManager, ruleId)
		if err != nil {
			return err
		}

		rule = _rule.(*models.SLoadbalancerListenerRule)
		listener, _ = rule.GetLoadbalancerListener()
	} else {
		_listener, err := db.FetchById(models.LoadbalancerListenerManager, listenerId)
		if err != nil {
			return err
		}
		listener = _listener.(*models.SLoadbalancerListener)
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.createLoadbalancerBackendGroup(ctx, userCred, listener, rule, lbbg, backends)
	})
	return nil
}

func (self *SOpenStackRegionDriver) validatorCachedLbbgConflict(ilb cloudprovider.ICloudLoadbalancer, lblis *models.SLoadbalancerListener, olbbg *models.SOpenstackCachedLbbg, nlbbg *models.SOpenstackCachedLbbg, rlbbg *models.SOpenstackCachedLbbg) error {
	if rlbbg == nil {
		return nil
	}

	if len(rlbbg.AssociatedId) > 0 && lblis.GetId() != rlbbg.AssociatedId {
		return fmt.Errorf("sync required, backend group aready assoicate with %s %s", rlbbg.AssociatedType, rlbbg.AssociatedId)
	}

	if olbbg != nil && len(rlbbg.AssociatedId) > 0 && rlbbg.AssociatedId != olbbg.AssociatedId {
		// 与本地关联状态不一致
		return fmt.Errorf("sync required, backend group aready assoicate with %s %s", rlbbg.AssociatedType, rlbbg.AssociatedId)
	}

	if nlbbg != nil && len(nlbbg.ExternalId) > 0 {
		// 需绑定的服务器组，被人为删除了
		_, err := ilb.GetILoadBalancerBackendGroupById(nlbbg.GetExternalId())
		if err != nil {
			return fmt.Errorf("sync required, validatorCachedLbbgConflict.GetILoadBalancerBackendGroupById %s", err)
		}

		// 需绑定的服务器组，被人为关联了其他监听或者监听规则
		listeners, err := ilb.GetILoadBalancerListeners()
		if err != nil {
			return fmt.Errorf("sync required, validatorCachedLbbgConflict.GetILoadBalancerListeners %s", err)
		}

		for i := range listeners {
			listener := listeners[i]
			if bgId := listener.GetBackendGroupId(); len(bgId) > 0 && bgId == nlbbg.ExternalId && listener.GetGlobalId() != lblis.GetExternalId() {
				return fmt.Errorf("sync required, backend group aready assoicate with listener %s(external id )", listener.GetGlobalId())
			}

			rules, err := listener.GetILoadbalancerListenerRules()
			if err != nil {
				return fmt.Errorf("sync required, validatorCachedLbbgConflict.GetILoadbalancerListenerRules %s", err)
			}

			for j := range rules {
				rule := rules[j]
				if bgId := rule.GetBackendGroupId(); len(bgId) > 0 && bgId == nlbbg.ExternalId {
					return fmt.Errorf("sync required, backend group aready assoicate with rule %s(external id )", rule.GetGlobalId())
				}
			}
		}
	}

	return nil
}

func (self *SOpenStackRegionDriver) RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg := lblis.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			err := fmt.Errorf("failed to find lbbg for lblis %s", lblis.Name)
			return nil, errors.Wrap(err, "OpenstackRegionDriver.RequestSyncLoadbalancerbackendGroup.GetLoadbalancerBackendGroup")
		}

		iRegion, err := lbbg.GetIRegion(ctx)
		if err != nil {
			return nil, err
		}

		lb, _ := lbbg.GetLoadbalancer()
		ilb, err := iRegion.GetILoadBalancerById(lb.GetExternalId())
		if err != nil {
			return nil, err
		}

		provider := lb.GetCloudprovider()
		if provider == nil {
			return nil, errors.Wrap(fmt.Errorf("provider is nil"), "OpenstackRegionDriver.RequestSyncLoadbalancerBackendGroup.GetCloudprovider")
		}

		// new group params
		groupInput, err := lbbg.GetOpenstackBackendGroupParams(lblis, nil)
		if err != nil {
			return nil, errors.Wrap(err, "GetOpenstackBackendGroupParams")
		}

		// new backends params
		backendsInput, err := lbbg.GetBackendsParams()
		if err != nil {
			return nil, err
		}

		cachedlbbgs, err := models.OpenstackCachedLbbgManager.GetListenerCachedBackendGroups(lblis.GetId())
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, errors.Wrap(fmt.Errorf("provider is nil"), "OpenstackRegionDriver.RequestSyncLoadbalancerBackendGroup.GetCachedBackendGroupByAssociateId")
			}
		}
		var cachedlbbg *models.SOpenstackCachedLbbg
		ibackendGroups := make(map[string]cloudprovider.ICloudLoadbalancerBackendGroup)
		for i := 0; i < len(cachedlbbgs); i++ {
			// fetch ilbbg
			ilbbg, err := ilb.GetILoadBalancerBackendGroupById(cachedlbbgs[i].ExternalId)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					ilbbg = nil
				} else {
					return nil, errors.Wrapf(err, "GetILoadBalancerBackendGroupById(%s)", cachedlbbgs[i].ExternalId)
				}
			}
			ibackendGroups[cachedlbbgs[i].Id] = ilbbg
			if cachedlbbgs[i].BackendGroupId == lblis.BackendGroupId && ibackendGroups[cachedlbbgs[i].Id] != nil {
				cachedlbbg = &cachedlbbgs[i]
			} else {
				// delete outdated cache,不保留游离的后端组
				if err := deleteOpenstackCachedLbbsByLbbg(ctx, userCred, cachedlbbgs[i].GetId()); err != nil {
					return nil, errors.Wrap(err, "deleteOpenstackCachedLbbg.deleteOpenstackCachedLbbsByLbbg")
				}
				err = db.DeleteModel(ctx, userCred, &cachedlbbgs[i])
				if err != nil {
					return nil, errors.Wrap(err, "deleteOpenstackCachedLbbg.DeleteModel")
				}
				//关联listener的可以直接删除
				if ibackendGroups[cachedlbbgs[i].Id] != nil {
					err := ibackendGroups[cachedlbbgs[i].Id].Delete(ctx)
					if err != nil {
						return nil, errors.Wrap(err, "iLoadbalancerBackendGroup.Delete(ctx)")
					}
				}
			}
		}
		// create or update
		if cachedlbbg == nil {
			// 新创建前需要确保listener没有defaultpool(外部操作)
			{
				iListener, err := ilb.GetILoadBalancerListenerById(lblis.ExternalId)
				if err != nil {
					return nil, err
				}
				ilbbgId := iListener.GetBackendGroupId()
				if len(ilbbgId) > 0 {
					ilbbg, err := ilb.GetILoadBalancerBackendGroupById(ilbbgId)
					if err != nil {
						if errors.Cause(err) != cloudprovider.ErrNotFound {
							return nil, errors.Wrapf(err, "GetILoadBalancerBackendGroupById(%s)", ilbbgId)
						}
					} else {
						err = ilbbg.Delete(ctx)
						if err != nil {
							return nil, errors.Wrap(err, "iLoadbalancerBackendGroup.Delete(ctx)")
						}
					}
				}
			}

			_, err = self.createLoadbalancerBackendGroup(ctx, task.GetUserCred(), lblis, nil, lbbg, backendsInput)
			if err != nil {
				return nil, errors.Wrap(err, "OpenstackRegionDriver.Sync.Case1.createLoadbalancerBackendGroup")
			}
		} else {
			err = self.syncCloudlbbs(ctx, userCred, lb, cachedlbbg, ibackendGroups[cachedlbbg.Id], backendsInput)
			if err != nil {
				return nil, errors.Wrap(err, "OpenstackRegionDriver.Sync.Case6.syncCloudlbbs")
			}

			err = self.syncCachedLbbs(ctx, userCred, lb, cachedlbbg, ibackendGroups[cachedlbbg.Id])
			if err != nil {
				return nil, errors.Wrap(err, "OpenstackRegionDriver.Sync.Case6.syncCloudlbbs")
			}
		}
		// continue here
		cachedLbbg, err := models.OpenstackCachedLbbgManager.GetCachedBackendGroupByAssociateId(lblis.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "OpenstackRegionDriver.Sync.GetCachedBackendGroupByAssociateId")
		}

		ilbbg, err := ilb.GetILoadBalancerBackendGroupById(cachedLbbg.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "OpenstackRegionDriver.Sync.GetILoadBalancerBackendGroupById")
		}

		err = ilbbg.Sync(ctx, groupInput)
		if err != nil {
			return nil, errors.Wrap(err, "OpenstackRegionDriver.Sync.LoadbalancerBackendGroup")
		}

		if err := cachedLbbg.SyncWithCloudLoadbalancerBackendgroup(ctx, task.GetUserCred(), lb, ilbbg, lb.GetOwnerId(), lb.GetCloudprovider()); err != nil {
			return nil, errors.Wrap(err, "OpenstackRegionDriver.Sync.SyncWithCloudLoadbalancerBackendgroup")
		}

		return nil, nil
	})

	return nil
}

func (self *SOpenStackRegionDriver) ValidateDeleteLoadbalancerBackendGroupCondition(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup) error {
	// pool不能被l7policy关联，若要解除关联关系，可通过更新转发策略将转测策略的redirect_pool_id更新为null。
	count, err := lbbg.RefCount()
	if err != nil {
		return err
	}

	if count != 0 {
		return fmt.Errorf("backendgroup is binding with loadbalancer/listener/listenerrule.")
	}

	return nil
}

func (self *SOpenStackRegionDriver) RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		iRegion, err := lbbg.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.")
		}
		loadbalancer, err := lbbg.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}

			return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.GetILoadBalancerById")
		}

		cachedLbbgs, err := models.OpenstackCachedLbbgManager.GetCachedBackendGroups(lbbg.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.GetCachedBackendGroups")
		}

		for _, cachedLbbg := range cachedLbbgs {
			if len(cachedLbbg.ExternalId) == 0 {
				continue
			}

			iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedLbbg.ExternalId)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					if err := deleteOpenstackCachedLbbg(ctx, userCred, cachedLbbg.AssociatedId); err != nil {
						return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.deleteOpenstackCachedLbbg")
					}

					continue
				}
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.GetILoadBalancerBackendGroupById")
			}

			ilbbs, err := iLoadbalancerBackendGroup.GetILoadbalancerBackends()
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.GetILoadbalancerBackends")
			}

			for _, ilbb := range ilbbs {
				iLoadbalancerBackendGroup.RemoveBackendServer(ilbb.GetId(), ilbb.GetWeight(), ilbb.GetPort())

				_cachedLbb, err := db.FetchByExternalId(models.OpenstackCachedLbManager, ilbb.GetGlobalId())
				if err != nil {
					if err != sql.ErrNoRows {
						return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.FetchByExternalId")
					}
					continue
				}

				cachedLbb := _cachedLbb.(*models.SOpenstackCachedLb)
				err = db.DeleteModel(ctx, userCred, cachedLbb)
				if err != nil {
					return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.DeleteModel")
				}
			}

			err = iLoadbalancerBackendGroup.Delete(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.Delete")
			}

			cachedLbbg.SetModelManager(models.OpenstackCachedLbbgManager, &cachedLbbg)
			err = db.DeleteModel(ctx, userCred, &cachedLbbg)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestDeleteLoadbalancerBackendGroup.DeleteModel")
			}
		}

		return nil, nil
	})
	return nil
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendType string, lb *models.SLoadbalancer, backendGroup *models.SLoadbalancerBackendGroup, backend db.IModel) (*jsonutils.JSONDict, error) {
	ownerId := ctx.Value("ownerId").(mcclient.IIdentityProvider)
	man := models.LoadbalancerBackendManager
	backendTypeV := validators.NewStringChoicesValidator("backend_type", api.LB_BACKEND_TYPES)
	keyV := map[string]validators.IValidator{
		"backend_type": backendTypeV,
		"weight":       validators.NewRangeValidator("weight", 0, 100).Default(10),
		"port":         validators.NewPortValidator("port"),
		"send_proxy":   validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Default(api.LB_SENDPROXY_OFF),
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	var basename string
	switch backendType {
	case api.LB_BACKEND_GUEST:
		backendV := validators.NewModelIdOrNameValidator("backend", "server", ownerId)
		err := backendV.Validate(data)
		if err != nil {
			return nil, err
		}
		guest := backendV.Model.(*models.SGuest)
		err = man.ValidateBackendVpc(lb, guest, backendGroup)
		if err != nil {
			return nil, err
		}

		address, err := models.LoadbalancerBackendManager.GetGuestAddress(guest)
		if err != nil {
			return nil, errors.Wrap(err, "OpenstackRegionDriver.ValidateCreateLoadbalancerBackendData.GetGuestAddress")
		}

		data.Set("address", jsonutils.NewString(address))
		basename = guest.Name
		backend = backendV.Model
	case api.LB_BACKEND_HOST:
		if db.IsAdminAllowCreate(userCred, man).Result.IsDeny() {
			return nil, fmt.Errorf("only sysadmin can specify host as backend")
		}
		backendV := validators.NewModelIdOrNameValidator("backend", "host", userCred)
		err := backendV.Validate(data)
		if err != nil {
			return nil, err
		}
		host := backendV.Model.(*models.SHost)
		{
			if len(host.AccessIp) == 0 {
				return nil, fmt.Errorf("host %s has no access ip", host.GetId())
			}
			data.Set("address", jsonutils.NewString(host.AccessIp))
		}
		basename = host.Name
		backend = backendV.Model
	case api.LB_BACKEND_IP:
		if db.IsAdminAllowCreate(userCred, man).Result.IsDeny() {
			return nil, fmt.Errorf("only sysadmin can specify ip address as backend")
		}
		backendV := validators.NewIPv4AddrValidator("backend")
		err := backendV.Validate(data)
		if err != nil {
			return nil, err
		}
		ip := backendV.IP.String()
		data.Set("address", jsonutils.NewString(ip))
		basename = ip
	default:
		return nil, fmt.Errorf("internal error: unexpected backend type %s", backendType)
	}

	name, _ := data.GetString("name")
	if name == "" {
		name = fmt.Sprintf("%s-%s-%s-%s", backendGroup.Name, backendType, basename, rand.String(4))
	}

	data.Set("name", jsonutils.NewString(name))
	data.Set("manager_id", jsonutils.NewString(lb.GetCloudproviderId()))
	data.Set("cloudregion_id", jsonutils.NewString(lb.GetRegionId()))
	return data, nil
}

func (self *SOpenStackRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lbbg *models.SLoadbalancerBackendGroup) (*jsonutils.JSONDict, error) {
	keyV := map[string]validators.IValidator{
		"weight":     validators.NewRangeValidator("weight", 0, 100).Optional(true),
		"port":       validators.NewPortValidator("port").Optional(true),
		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Optional(true),
	}

	if err := RunValidators(keyV, data, true); err != nil {
		return nil, err
	}

	// 只能更新权重。不能更新端口
	port, err := data.Int("port")
	if err == nil && port != 0 {
		return data, fmt.Errorf("can not update backend port.")
	}

	return data, nil
}

func (self *SOpenStackRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg, err := lbb.GetLoadbalancerBackendGroup()
		if err != nil {
			return nil, err
		}
		lb, err := lbbg.GetLoadbalancer()
		if err != nil {
			return nil, err
		}

		cachedlbbgs, err := models.OpenstackCachedLbbgManager.GetCachedBackendGroups(lbbg.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "openstackRegionDriver.RequestCreateLoadbalancerBackend.GetCachedBackendGroups")
		}

		guest := lbb.GetGuest()
		if guest == nil {
			return nil, fmt.Errorf("failed to find guest for lbb %s", lbb.Name)
		}

		var ibackend cloudprovider.ICloudLoadbalancerBackend
		for _, cachedLbbg := range cachedlbbgs {
			iLoadbalancerBackendGroup, err := cachedLbbg.GetICloudLoadbalancerBackendGroup(ctx)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					if err := deleteOpenstackCachedLbbg(ctx, userCred, cachedLbbg.AssociatedId); err != nil {
						return nil, errors.Wrap(err, "openstackRegionDriver.RequestCreateLoadbalancerBackend.deleteOpenstackCachedLbbg")
					}

					continue
				}

				return nil, errors.Wrap(err, "openstackRegionDriver.RequestCreateLoadbalancerBackend.GetICloudLoadbalancerBackendGroup")
			}

			ibackend, err = iLoadbalancerBackendGroup.AddBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestCreateLoadbalancerBackend.AddBackendServer")
			}

			_, err = models.OpenstackCachedLbManager.CreateOpenstackCachedLb(ctx, userCred, lbb, &cachedLbbg, ibackend, cachedLbbg.GetOwnerId())
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestCreateLoadbalancerBackend.CreateOpenstackCachedLb")
			}
		}

		if ibackend != nil {
			if err := lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, ibackend, lbbg.GetOwnerId(), lb.GetCloudprovider()); err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestCreateLoadbalancerBackend.SyncWithCloudLoadbalancerBackend")
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SOpenStackRegionDriver) RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		cachedlbbs, err := models.OpenstackCachedLbManager.GetBackendsByLocalBackendId(lbb.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerBackend.GetBackendsByLocalBackendId")
		}

		for _, cachedlbb := range cachedlbbs {
			cachedlbbg, _ := cachedlbb.GetCachedBackendGroup()
			if cachedlbbg == nil {
				return nil, fmt.Errorf("failed to find lbbg for backend %s", cachedlbb.Name)
			}
			lb, err := cachedlbbg.GetLoadbalancer()
			if err != nil {
				return nil, errors.Wrap(err, "cachedlbbg.GetLoadbalancer()")
			}
			iRegion, err := lb.GetIRegion(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerBackend.GetIRegion")
			}
			iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerBackend.GetILoadBalancerById")
			}
			iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedlbbg.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerBackend.GetILoadBalancerBackendGroupById")
			}

			iBackend, err := iLoadbalancerBackendGroup.GetILoadbalancerBackendById(cachedlbb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerBackend.GetILoadbalancerBackendById")
			}

			err = iBackend.SyncConf(ctx, lbb.Port, lbb.Weight)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerBackend.SyncConf")
			}

			iBackend, err = iLoadbalancerBackendGroup.GetILoadbalancerBackendById(cachedlbb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerBackend.GetILoadbalancerBackendById")
			}

			err = cachedlbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iBackend, lbb.GetOwnerId())
			if err != nil {
				return nil, errors.Wrap(err, "openstackRegionDriver.RequestSyncLoadbalancerBackend.SyncWithCloudLoadbalancerBackend")
			}
		}

		return nil, nil
	})
	return nil
}

func (self *SOpenStackRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}

		cachedlbbs, err := models.OpenstackCachedLbManager.GetBackendsByLocalBackendId(lbb.GetId())
		if err != nil {
			return nil, err
		}

		for _, cachedlbb := range cachedlbbs {
			cachedlbbg, _ := cachedlbb.GetCachedBackendGroup()
			if cachedlbbg == nil {
				log.Warningf("failed to find lbbg for backend %s", cachedlbb.Name)
				continue
			}
			lb, err := cachedlbbg.GetLoadbalancer()
			if err != nil {
				return nil, errors.Wrap(err, "cachedlbbg.GetLoadbalancer()")
			}
			iRegion, err := lb.GetIRegion(ctx)
			if err != nil {
				return nil, err
			}
			iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
			if err != nil {
				return nil, err
			}
			iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedlbbg.ExternalId)
			if err != nil {
				return nil, err
			}

			err = iLoadbalancerBackendGroup.RemoveBackendServer(cachedlbb.ExternalId, lbb.Weight, lbb.Port)
			if err != nil {
				return nil, err
			}

			err = db.DeleteModel(ctx, userCred, &cachedlbb)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})
	return nil
}
