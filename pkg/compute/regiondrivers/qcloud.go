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
	"regexp"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rand"
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

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ownerId := ctx.Value("ownerId").(mcclient.IIdentityProvider)
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	vpcV := validators.NewModelIdOrNameValidator("vpc", "vpc", ownerId)
	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", ownerId)
	addressTypeV := validators.NewStringChoicesValidator("address_type", api.LB_ADDR_TYPES)

	keyV := map[string]validators.IValidator{
		"status":       validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),
		"address_type": addressTypeV.Default(api.LB_ADDR_TYPE_INTRANET),
		"vpc":          vpcV,
		"zone":         zoneV,
		"manager":      managerIdV,
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	// 内网ELB需要增加network
	if addressTypeV.Value == api.LB_ADDR_TYPE_INTRANET {
		networkV := validators.NewModelIdOrNameValidator("network", "network", ownerId)
		if err := networkV.Validate(data); err != nil {
			return nil, err
		}

		network := networkV.Model.(*models.SNetwork)
		_, _, vpc, _, err := network.ValidateElbNetwork(nil)
		if err != nil {
			return nil, err
		}

		if managerIdV.Model.GetId() != vpc.ManagerId {
			return nil, httperrors.NewInputParameterError("Loadbalancer's manager (%s(%s)) does not match vpc's(%s(%s)) (%s)", managerIdV.Model.GetName(), managerIdV.Model.GetId(), vpc.GetName(), vpc.GetId(), vpc.ManagerId)
		}
	}

	region := zoneV.Model.(*models.SZone).GetRegion()
	if region == nil {
		return nil, fmt.Errorf("getting region failed")
	}

	data.Set("network_type", jsonutils.NewString(api.LB_NETWORK_TYPE_VPC))
	data.Set("cloudregion_id", jsonutils.NewString(region.GetId()))
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, data)
}

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	listenerTypeV := validators.NewStringChoicesValidator("listener_type", api.LB_LISTENER_TYPES)
	listenerPortV := validators.NewPortValidator("listener_port")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),

		"listener_type": listenerTypeV,
		"listener_port": listenerPortV,

		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Default(api.LB_SENDPROXY_OFF),
		"scheduler":  validators.NewStringChoicesValidator("scheduler", api.LB_SCHEDULER_TYPES),

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES).Default(api.LB_BOOL_OFF),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", api.LB_STICKY_SESSION_TYPES).Default(api.LB_STICKY_SESSION_TYPE_INSERT),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)).Optional(true),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout").Optional(true),

		"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for").Default(true),
		"gzip":            validators.NewBoolValidator("gzip").Default(false),
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	//  listener uniqueness
	listenerType := listenerTypeV.Value
	err := models.LoadbalancerListenerManager.CheckListenerUniqueness(ctx, lb, listenerType, listenerPortV.Value)
	if err != nil {
		return nil, err
	}

	// backendgroup check
	if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != lb.Id {
		return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
			lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, lb.Id)
	} else {
		if lbbg != nil {
			count, err := lbbg.RefCount()
			if err != nil {
				return nil, httperrors.NewInternalServerError("get lbbg RefCount fail %s", err)
			}
			if count > 0 {
				return nil, httperrors.NewResourceBusyError("backendgroup aready related with other listener/rule")
			}
		}
	}

	// https additional certificate check
	if listenerType == api.LB_LISTENER_TYPE_HTTPS {
		certV := validators.NewModelIdOrNameValidator("certificate", "loadbalancercertificate", ownerId)
		tlsCipherPolicyV := validators.NewStringChoicesValidator("tls_cipher_policy", api.LB_TLS_CIPHER_POLICIES).Default(api.LB_TLS_CIPHER_POLICY_1_2)
		httpsV := map[string]validators.IValidator{
			"certificate":       certV,
			"tls_cipher_policy": tlsCipherPolicyV,
			"enable_http2":      validators.NewBoolValidator("enable_http2").Default(true),
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
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 10).Default(3),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 1, 10).Default(3),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 2, 60).Default(2),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 5, 300).Default(5),
	}

	if err := RunValidators(keyVHealth, data, false); err != nil {
		return nil, err
	}

	data.Set("acl_status", jsonutils.NewString(api.LB_BOOL_OFF))
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerListenerData(ctx, userCred, ownerId, data, lb, backendGroup)
}

func (self *SQcloudRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lbbg.GetIRegion()
		if err != nil {
			return nil, err
		}

		// 腾讯云本身没有后端服务器组，因此不需要在qcloud端执行创建操作
		if iRegion.GetProvider() == api.CLOUD_PROVIDER_QCLOUD {
			return nil, nil
		}

		loadbalancer := lbbg.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		group := &cloudprovider.SLoadbalancerBackendGroup{
			Name:      lbbg.Name,
			GroupType: lbbg.Type,
			Backends:  backends,
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.CreateILoadBalancerBackendGroup(group)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lbbg, userCred, iLoadbalancerBackendGroup.GetGlobalId()); err != nil {
			return nil, err
		}
		iBackends, err := iLoadbalancerBackendGroup.GetILoadbalancerBackends()
		if err != nil {
			return nil, err
		}
		if len(iBackends) > 0 {
			provider := loadbalancer.GetCloudprovider()
			if provider == nil {
				return nil, fmt.Errorf("failed to find cloudprovider for lb %s", loadbalancer.Name)
			}
			models.LoadbalancerBackendManager.SyncLoadbalancerBackends(ctx, userCred, provider, lbbg, iBackends, &models.SSyncRange{})
		}
		return nil, nil
	})
	return nil
}

func (self *SQcloudRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg := lbb.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			return nil, fmt.Errorf("failed to find lbbg for backend %s", lbb.Name)
		}

		guest := lbb.GetGuest()
		if guest == nil {
			return nil, fmt.Errorf("failed to find guest for lbb %s", lbb.Name)
		}

		// 兼容腾讯云，在fake的backend group 关联具体的转发策略之前。不需要同步后端服务器
		if lbbg.GetProviderName() == api.CLOUD_PROVIDER_QCLOUD {
			cnt, err := lbbg.RefCount()
			if err != nil {
				return nil, err
			}
			if cnt == 0 {
				return nil, nil
			}
		}

		lb := lbbg.GetLoadbalancer()
		if lb == nil {
			return nil, fmt.Errorf("failed to find lb for backendgroup %s", lbbg.Name)
		}
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			return nil, err
		}
		iLoadbalancerBackend, err := iLoadbalancerBackendGroup.AddBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lbb, userCred, iLoadbalancerBackend.GetGlobalId()); err != nil {
			return nil, err
		}
		return nil, lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iLoadbalancerBackend, nil)
	})
	return nil
}

func (self *SQcloudRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		lbbg := lbb.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			return nil, fmt.Errorf("failed to find lbbg for backend %s", lbb.Name)
		}
		lb := lbbg.GetLoadbalancer()
		if lb == nil {
			return nil, fmt.Errorf("failed to find lb for backendgroup %s", lbbg.Name)
		}
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}

		// ===========兼容腾讯云,未关联具体转发规则时，直接删除本地数据即可===============
		if iRegion.GetProvider() == api.CLOUD_PROVIDER_QCLOUD {
			count, err := lbbg.RefCount()
			if err != nil {
				return nil, err
			}
			if count == 0 {
				return nil, nil
			}
		}

		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			return nil, err
		}
		guest := lbb.GetGuest()
		if guest == nil {
			return nil, fmt.Errorf("failed to find guest for lbb %s", lbb.Name)
		}
		return nil, iLoadbalancerBackendGroup.RemoveBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
	})
	return nil
}

func (self *SQcloudRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		{
			certId, _ := task.GetParams().GetString("certificate_id")
			if len(certId) > 0 {
				provider := models.CloudproviderManager.FetchCloudproviderById(lblis.ManagerId)
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
					err = self.RequestCreateLoadbalancerCertificate(ctx, userCred, lbcert, task)
					if err != nil {
						return nil, err
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedCertificateId = lbcert.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "QcloudRegionDriver.RequestCreateLoadbalancerListener.UpdateCachedCertificateId")
				}
			}
		}

		{
			aclId, _ := task.GetParams().GetString("acl_id")
			if len(aclId) > 0 {
				provider := models.CloudproviderManager.FetchCloudproviderById(lblis.ManagerId)
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				acl, err := models.LoadbalancerAclManager.FetchById(aclId)
				if err != nil {
					return nil, err
				}

				lbacl, err := models.CachedLoadbalancerAclManager.GetOrCreateCachedAcl(ctx, userCred, provider, lblis, acl.(*models.SLoadbalancerAcl))
				if err != nil {
					return nil, err
				}

				if len(lbacl.ExternalId) == 0 {
					err = self.RequestCreateLoadbalancerAcl(ctx, userCred, lbacl, task)
					if err != nil {
						return nil, err
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedAclId = lbacl.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "QcloudRegionDriver.RequestCreateLoadbalancerListener.UpdateCachedAclId")
				}
			}
		}

		params, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, err
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.CreateILoadBalancerListener(params)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lblis, userCred, iListener.GetGlobalId()); err != nil {
			return nil, err
		}

		// ====腾讯云添加后端服务器=====
		if iRegion.GetProvider() == api.CLOUD_PROVIDER_QCLOUD {
			group := lblis.GetLoadbalancerBackendGroup()
			if group != nil {
				backends, err := group.GetBackends()
				if err != nil {
					return nil, fmt.Errorf("failed to find backends for backend group  %s: %s", group.GetId(), err)
				}

				extBgID := iListener.GetBackendGroupId()
				if len(extBgID) == 0 {
					return nil, fmt.Errorf("the backend group external id of loadbalancer listener  %s is empty", lblis.GetId())
				}

				ilbbg, err := iLoadbalancer.GetILoadBalancerBackendGroupById(extBgID)
				if err != nil {
					return nil, fmt.Errorf("failed to find backend group for loadbalancer listener  %s: %s", lblis.GetId(), err)
				}

				for _, backend := range backends {
					guest := backend.GetGuest()
					if guest == nil {
						return nil, fmt.Errorf("failed to find instance for loadbalancer backend  %s", backend.GetId())
					}
					_, err := ilbbg.AddBackendServer(guest.GetExternalId(), backend.Weight, backend.Port)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, nil)
	})
	return nil
}

func (self *SQcloudRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		listener := lbr.GetLoadbalancerListener()
		if listener == nil {
			return nil, fmt.Errorf("failed to find listener for listnener rule %s", lbr.Name)
		}
		loadbalancer := listener.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for listener %s", listener.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
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
		}
		if len(lbr.BackendGroupId) > 0 {
			group := lbr.GetLoadbalancerBackendGroup()
			if group == nil {
				return nil, fmt.Errorf("failed to find backend group for listener rule %s", lbr.Name)
			}
			rule.BackendGroupID = group.ExternalId
			rule.BackendGroupType = group.Type
		}
		iListenerRule, err := iListener.CreateILoadBalancerListenerRule(rule)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lbr, userCred, iListenerRule.GetGlobalId()); err != nil {
			return nil, err
		}
		// ====腾讯云添加后端服务器=====
		if listener.GetProviderName() == api.CLOUD_PROVIDER_QCLOUD && len(rule.BackendGroupID) > 0 {
			ilbbg, err := iLoadbalancer.GetILoadBalancerBackendGroupById(rule.BackendGroupID)
			if err != nil {
				return nil, fmt.Errorf("failed to find backend group for listener rule %s: %s", lbr.Name, err)
			}

			group := lbr.GetLoadbalancerBackendGroup()
			backends, err := group.GetBackends()
			if err != nil {
				return nil, fmt.Errorf("failed to find backends for backend group  %s: %s", group.GetId(), err)
			}

			for _, backend := range backends {
				guest := backend.GetGuest()
				if guest == nil {
					return nil, fmt.Errorf("failed to find instance for loadbalancer backend  %s", backend.GetId())
				}
				_, err := ilbbg.AddBackendServer(guest.GetExternalId(), backend.Weight, backend.Port)
				if err != nil {
					return nil, err
				}
			}

		}
		return nil, lbr.SyncWithCloudLoadbalancerListenerRule(ctx, userCred, iListenerRule, nil)
	})
	return nil
}

func (self *SQcloudRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	cidrV := validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(data); err != nil {
		return nil, err
	}
	if cidrV.Value.MaskLen < 16 || cidrV.Value.MaskLen > 28 {
		return nil, httperrors.NewInputParameterError("%s request the mask range should be between 16 and 28", self.GetProvider())
	}
	return data, nil
}

func (self *SQcloudRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lblis *models.SLoadbalancerListener, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	ownerId := lblis.GetOwnerId()
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

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", api.LB_STICKY_SESSION_TYPES),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout"),

		"health_check":      validators.NewStringChoicesValidator("health_check", api.LB_BOOL_VALUES),
		"health_check_type": models.LoadbalancerListenerManager.CheckTypeV(lblis.ListenerType),

		"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true).Default(""),
		"health_check_path":      validators.NewURLPathValidator("health_check_path").Default(""),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 10).Default(3),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 1, 10).Default(3),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 2, 60).Default(2),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 5, 300).Default(5),

		"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for"),
		"gzip":            validators.NewBoolValidator("gzip"),

		"certificate":       certV,
		"tls_cipher_policy": tlsCipherPolicyV,
		"enable_http2":      validators.NewBoolValidator("enable_http2"),
	}

	if err := RunValidators(keyV, data, true); err != nil {
		return nil, err
	}

	{
		if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != lblis.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, lblis.LoadbalancerId)
		}
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateLoadbalancerListenerData(ctx, userCred, data, lblis, backendGroup)
}

func (self *SQcloudRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lbbg *models.SLoadbalancerBackendGroup) (*jsonutils.JSONDict, error) {
	keyV := map[string]validators.IValidator{
		"weight":     validators.NewRangeValidator("weight", 1, 256).Optional(true),
		"port":       validators.NewPortValidator("port").Optional(true),
		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Optional(true),
	}

	if err := RunValidators(keyV, data, true); err != nil {
		return nil, err
	}
	return data, nil
}

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	domainV := validators.NewDomainNameValidator("domain")
	pathV := validators.NewURLPathValidator("path")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),
		"domain": domainV.AllowEmpty(true).Default(""),
		"path":   pathV.Default(""),

		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate").Default(0),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src").Default(0),
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	listenerId, err := data.GetString("listener_id")
	if err != nil {
		return nil, err
	}

	ilistener, err := db.FetchById(models.LoadbalancerListenerManager, listenerId)
	if err != nil {
		return nil, err
	}

	listener := ilistener.(*models.SLoadbalancerListener)
	listenerType := listener.ListenerType
	if listenerType != api.LB_LISTENER_TYPE_HTTP && listenerType != api.LB_LISTENER_TYPE_HTTPS {
		return nil, httperrors.NewInputParameterError("listener type must be http/https, got %s", listenerType)
	}

	if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != listener.LoadbalancerId {
		return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
			lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, listener.LoadbalancerId)
	}

	err = models.LoadbalancerListenerRuleCheckUniqueness(ctx, listener, domainV.Value, pathV.Value)
	if err != nil {
		return nil, err
	}

	data.Set("cloudregion_id", jsonutils.NewString(listener.CloudregionId))
	data.Set("manager_id", jsonutils.NewString(listener.ManagerId))
	return data, nil
}

func (self *SQcloudRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	lbr := ctx.Value("lbr").(*models.SLoadbalancerListenerRule)
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

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendType string, lb *models.SLoadbalancer, backendGroup *models.SLoadbalancerBackendGroup, backend db.IModel) (*jsonutils.JSONDict, error) {
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
		basename = guest.Name
		backend = backendV.Model
	case api.LB_BACKEND_HOST:
		if !db.IsAdminAllowCreate(userCred, man) {
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
		if !db.IsAdminAllowCreate(userCred, man) {
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
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	return data, nil
}

func (self *SQcloudRegionDriver) RequestPreSnapshotPolicyApply(ctx context.Context, userCred mcclient.
	TokenCredential, task taskman.ITask, disk *models.SDisk, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) error {

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		if sp == nil {
			return data, nil
		}
		spcache, err := models.SnapshotPolicyCacheManager.FetchSnapshotPolicyCache(sp.GetId(),
			disk.GetStorage().GetRegion().GetId(), disk.GetStorage().ManagerId)

		iRegion, err := spcache.GetIRegion()
		if err != nil {
			return nil, err
		}
		err = iRegion.CancelSnapshotPolicyToDisks(spcache.GetExternalId(), disk.GetExternalId())
		if err != nil {
			return nil, err
		}
		return data, nil
	})
	return nil
}
