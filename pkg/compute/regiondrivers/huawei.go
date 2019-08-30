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

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ownerId := ctx.Value("ownerId").(mcclient.IIdentityProvider)
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", ownerId)
	addressTypeV := validators.NewStringChoicesValidator("address_type", api.LB_ADDR_TYPES)
	networkV := validators.NewModelIdOrNameValidator("network", "network", ownerId)

	keyV := map[string]validators.IValidator{
		"status":       validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),
		"address_type": addressTypeV.Default(api.LB_ADDR_TYPE_INTRANET),
		"network":      networkV,
		"zone":         zoneV,
		"manager":      managerIdV,
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	//  检查网络可用
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

	region := zoneV.Model.(*models.SZone).GetRegion()
	if region == nil {
		return nil, fmt.Errorf("getting region failed")
	}

	data.Set("network_type", jsonutils.NewString(api.LB_NETWORK_TYPE_VPC))
	data.Set("cloudregion_id", jsonutils.NewString(region.GetId()))
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, data)
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0143878053.html
func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// 访问控制： 与listener是1v1的
	// 关系，创建时即需要与具体的listener绑定，不能再变更listner。
	// required: listener_id, acl_type: "white", acl_status: "on", manager,cloudregion,acl_entries
	data, err := self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerAclData(ctx, userCred, data)
	if err != nil {
		return data, err
	}

	// todo: ownId ??
	listenerV := validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", nil)
	err = listenerV.Validate(data)
	if err != nil {
		return data, err
	}

	return data, nil
}

// func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
// 	// required：certificate （PEM格式），private_key（PEM格式），name
// 	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer certificate", self.GetProvider())
// }

// func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backends []cloudprovider.SLoadbalancerBackend) (*jsonutils.JSONDict, error) {
// 	data, err := self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerBackendGroupData(ctx, userCred, data, lb, backends)
// 	if err != nil {
// 		return data, err
// 	}
//
// 	listener_id, _ := data.GetString("listener_id")
// 	if len(listener_id) > 0 {
// 		ilistener, err := models.LoadbalancerListenerManager.FetchById(listener_id)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		_lbbgId := ilistener.(*models.SLoadbalancerListener).BackendGroupId
// 		if len(_lbbgId) > 0 {
// 			return nil, fmt.Errorf("loadbalancer listener %s aready binding with backendgroup %s", listener_id, _lbbgId)
// 		}
// 	}
//
// 	{
// 		protocolTypeV := validators.NewStringChoicesValidator("protocol_type", api.HUAWEI_LBBG_PROTOCOL_TYPES)
// 		keyV := map[string]validators.IValidator{
// 			"protocol_type":                 protocolTypeV,
// 			"scheduler":                     validators.NewStringChoicesValidator("scheduler", api.HUAWEI_LBBG_SCHDULERS),
// 			"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES).Default(api.LB_BOOL_OFF),
// 			"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", api.LB_STICKY_SESSION_TYPES).Default(api.LB_STICKY_SESSION_TYPE_INSERT),
// 			"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)).Optional(true),
// 			"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout").Optional(true),
// 		}
//
// 		for _, v := range keyV {
// 			if err := v.Validate(data); err != nil {
// 				return nil, err
// 			}
// 		}
// 	}
//
// 	{
// 		// health check default depends on input parameters
// 		_t, _ := data.GetString("protocol_type")
// 		checkTypeV := models.LoadbalancerListenerManager.CheckTypeV(_t)
// 		keyVHealth := map[string]validators.IValidator{
// 			"health_check":      validators.NewStringChoicesValidator("health_check", api.LB_BOOL_VALUES).Default(api.LB_BOOL_OFF),
// 			"health_check_type": checkTypeV,
//
// 			"health_check_domain":   validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true).Default(""),
// 			"health_check_path":     validators.NewURLPathValidator("health_check_path").Default(""),
// 			"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 10).Default(3),
// 			"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 50).Default(5),
// 			"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Default(5),
// 		}
// 		for _, v := range keyVHealth {
// 			if err := v.Validate(data); err != nil {
// 				return nil, err
// 			}
// 		}
// 	}
// 	return data, nil
// }

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendType string, lb *models.SLoadbalancer, backendGroup *models.SLoadbalancerBackendGroup, backend db.IModel) (*jsonutils.JSONDict, error) {
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

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
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
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 50).Default(10),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Default(5),
	}

	if err := RunValidators(keyVHealth, data, false); err != nil {
		return nil, err
	}

	// acl check
	if err := models.LoadbalancerListenerManager.ValidateAcl(aclStatusV, aclTypeV, aclV, data, api.CLOUD_PROVIDER_HUAWEI); err != nil {
		return nil, err
	}

	data.Set("acl_status", jsonutils.NewString(api.LB_BOOL_OFF))
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerListenerData(ctx, userCred, ownerId, data, lb, backendGroup)
}

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
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

	_, err = self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerListenerRuleData(ctx, userCred, ownerId, data, backendGroup)
	if err != nil {
		return data, err
	}

	domain, _ := data.GetString("domain")
	path, _ := data.GetString("path")
	if domain == "" && path == "" {
		return data, fmt.Errorf("'domain' or 'path' should not be empty.")
	}

	data.Set("cloudregion_id", jsonutils.NewString(listener.CloudregionId))
	data.Set("manager_id", jsonutils.NewString(listener.ManagerId))
	return data, nil
}

func (self *SHuaWeiRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
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

// func (self *SHuaWeiRegionDriver) ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
// 	// required：certificate （PEM格式），private_key（PEM格式），name， id
// 	return nil, nil
// }

func (self *SHuaWeiRegionDriver) ValidateDeleteLoadbalancerBackendGroupCondition(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup) error {
	//删除pool之前必须删除pool上的所有member和healthmonitor，并且pool不能被l7policy关联，若要解除关联关系，可通过更新转发策略将转测策略的redirect_pool_id更新为null。
	count, err := lbbg.RefCount()
	if err != nil {
		return err
	}

	if count != 0 {
		return fmt.Errorf("backendgroup is binding with loadbalancer/listener/listenerrule.")
	}

	return nil
}

func (self *SHuaWeiRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lbbg *models.SLoadbalancerBackendGroup) (*jsonutils.JSONDict, error) {
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

// func (self *SHuaWeiRegionDriver) ValidateDeleteLoadbalancerBackendCondition(ctx context.Context, lbb *models.SLoadbalancerBackend) error {
// 	// required：backendgroup id, serverId
// 	return nil
// }

func (self *SHuaWeiRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lblis *models.SLoadbalancerListener, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	ownerId := lblis.GetOwnerId()
	aclStatusV := validators.NewStringChoicesValidator("acl_status", api.LB_BOOL_VALUES)
	aclStatusV.Default(lblis.AclStatus)
	aclTypeV := validators.NewStringChoicesValidator("acl_type", api.LB_ACL_TYPES)
	if api.LB_ACL_TYPES.Has(lblis.AclType) {
		aclTypeV.Default(lblis.AclType)
	}
	var aclV *validators.ValidatorModelIdOrName
	if _acl, _ := data.GetString("acl"); len(_acl) > 0 {
		aclV = validators.NewModelIdOrNameValidator("acl", "loadbalanceracl", ownerId)
	} else {
		aclV = validators.NewModelIdOrNameValidator("acl", "cachedloadbalanceracl", ownerId)
		if len(lblis.AclId) > 0 {
			aclV.Default(lblis.AclId)
		}
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
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 50).Default(10),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Default(5),

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

func (self *SHuaWeiRegionDriver) createLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, lbr *models.SLoadbalancerListenerRule, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend) (jsonutils.JSONObject, error) {
	if len(lblis.ListenerType) == 0 {
		return nil, fmt.Errorf("loadbalancer backendgroup missing protocol type")
	}

	iRegion, err := lbbg.GetIRegion()
	if err != nil {
		return nil, err
	}
	lb := lbbg.GetLoadbalancer()
	if lb == nil {
		return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
	}
	iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
	if err != nil {
		return nil, err
	}

	// create loadbalancer backendgroup cache
	cachedLbbg := &models.SHuaweiCachedLbbg{}
	cachedLbbg.ManagerId = lb.ManagerId
	cachedLbbg.CloudregionId = lb.CloudregionId
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

	err = models.HuaweiCachedLbbgManager.TableSpec().Insert(cachedLbbg)
	if err != nil {
		return nil, err
	}

	group, err := lbbg.GetHuaweiBackendGroupParams(lblis, lbr)
	if err != nil {
		return nil, err
	}

	iLoadbalancerBackendGroup, err := iLoadbalancer.CreateILoadBalancerBackendGroup(group)
	if err != nil {
		return nil, err
	}

	cachedLbbg.SetModelManager(models.HuaweiCachedLbbgManager, cachedLbbg)
	if err := db.SetExternalId(cachedLbbg, userCred, iLoadbalancerBackendGroup.GetGlobalId()); err != nil {
		return nil, err
	}

	for _, backend := range backends {
		cachedlbb := &models.SHuaweiCachedLb{}
		cachedlbb.ManagerId = lb.ManagerId
		cachedlbb.CloudregionId = lb.CloudregionId
		cachedlbb.CachedBackendGroupId = cachedLbbg.GetId()
		cachedlbb.BackendId = backend.ID
		err = models.HuaweiCachedLbManager.TableSpec().Insert(cachedlbb)
		if err != nil {
			return nil, err
		}

		ibackend, err := iLoadbalancerBackendGroup.AddBackendServer(backend.ExternalID, backend.Weight, backend.Port)
		if err != nil {
			return nil, err
		}

		cachedlbb.SetModelManager(models.HuaweiCachedLbManager, cachedlbb)
		err = db.SetExternalId(cachedlbb, userCred, ibackend.GetGlobalId())
		if err != nil {
			return nil, err
		}
	}

	iBackends, err := iLoadbalancerBackendGroup.GetILoadbalancerBackends()
	if err != nil {
		return nil, err
	}
	if len(iBackends) > 0 {
		provider := lb.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to find cloudprovider for lb %s", lb.Name)
		}
		models.HuaweiCachedLbManager.SyncLoadbalancerBackends(ctx, userCred, provider, cachedLbbg, iBackends, &models.SSyncRange{})
	}
	return nil, nil

}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
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
		listener = rule.GetLoadbalancerListener()
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

func (self *SHuaWeiRegionDriver) createLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl) (jsonutils.JSONObject, error) {
	iRegion, err := lbacl.GetIRegion()
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
		AccessControlEnable: (listener.AclStatus == api.LB_BOOL_ON),
	}
	if lbacl.AclEntries != nil {
		for _, entry := range *lbacl.AclEntries {
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
	return nil, lbacl.SyncWithCloudLoadbalancerAcl(ctx, userCred, iLoadbalancerAcl, listener.GetOwnerId())
}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.createLoadbalancerAcl(ctx, userCred, lbacl)
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lbbg.GetIRegion()
		if err != nil {
			return nil, err
		}

		lb := lbbg.GetLoadbalancer()
		ilb, err := iRegion.GetILoadBalancerById(lb.GetExternalId())
		if err != nil {
			return nil, err
		}

		ilisten, err := ilb.GetILoadBalancerListenerById(lblis.GetExternalId())
		if err != nil {
			return nil, err
		}

		olbbg := ilisten.GetBackendGroupId()
		if len(olbbg) > 0 {
			p, err := lblis.GetHuaweiLoadbalancerListenerParams()
			if err != nil {
				return nil, err
			}

			err = ilisten.Sync(p)
			if err != nil {
				return nil, err
			}

			_t, err := db.FetchByExternalId(models.HuaweiCachedLbbgManager, olbbg)
			if err != nil {
				return nil, err
			}

			tmpLbbg := _t.(*models.SHuaweiCachedLbbg)
			_, err = db.UpdateWithLock(ctx, tmpLbbg, func() error {
				tmpLbbg.AssociatedId = ""
				tmpLbbg.AssociatedType = ""
				return nil
			})

			if err != nil {
				return nil, err
			}
		}

		backends, err := lbbg.GetBackendsParams()
		if err != nil {
			return nil, err
		}

		nlbbg, err := models.HuaweiCachedLbbgManager.GetUsableCachedBackendGroup(lbbg.GetId(), lblis.ListenerType)
		if nlbbg == nil {
			if _, err := self.createLoadbalancerBackendGroup(ctx, task.GetUserCred(), lblis, nil, lbbg, backends); err != nil {
				return nil, err
			}
		} else {
			ilbbg, err := ilb.GetILoadBalancerBackendGroupById(nlbbg.GetExternalId())
			if err != nil {
				return nil, err
			}

			group, err := lbbg.GetHuaweiBackendGroupParams(lblis, nil)
			if err != nil {
				return nil, err
			}

			if err := ilbbg.Sync(group); err != nil {
				return nil, err
			}

			nlbbg.SetModelManager(models.HuaweiCachedLbbgManager, nlbbg)
			_, err = db.UpdateWithLock(ctx, nlbbg, func() error {
				nlbbg.AssociatedId = lblis.GetId()
				nlbbg.AssociatedType = api.LB_ASSOCIATE_TYPE_LISTENER
				return nil
			})
		}
		// continue here
		cachedLbbg, err := models.HuaweiCachedLbbgManager.GetCachedBackendGroupByAssociateId(lblis.GetId())
		if err != nil {
			return nil, err
		}

		ilbbg, err := ilb.GetILoadBalancerBackendGroupById(cachedLbbg.GetExternalId())
		if err != nil {
			return nil, err
		}

		if err := cachedLbbg.SyncWithCloudLoadbalancerBackendgroup(ctx, task.GetUserCred(), lb, ilbbg, lb.GetOwnerId()); err != nil {
			return nil, err
		}

		return nil, nil
	})

	return nil
}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
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
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, err
					}
				}

				// lblis.CertificateId = lbcert.ExternalId
			}
		}

		params, err := lblis.GetHuaweiLoadbalancerListenerParams()
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

		lblis.SetModelManager(models.LoadbalancerListenerManager, lblis)
		if err := db.SetExternalId(lblis, userCred, iListener.GetGlobalId()); err != nil {
			return nil, err
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
					_, err = self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, loadbalancer.GetOwnerId())
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
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
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, err
					}
				}

				lblis.ExternalId = lbcert.ExternalId
			}
		}

		params, err := lblis.GetHuaweiLoadbalancerListenerParams()
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
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, err
		}
		if err := iListener.Sync(params); err != nil {
			return nil, err
		}

		{
			aclId, _ := task.GetParams().GetString("acl_id")
			if len(aclId) > 0 {
				provider := models.CloudproviderManager.FetchCloudproviderById(lblis.ManagerId)
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

				lblis.AclId = lbacl.ExternalId
			}
		}

		if err := iListener.Refresh(); err != nil {
			return nil, err
		}
		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, lblis.GetOwnerId())
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
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
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}

		// 取消服务器组关联
		backendgroupId := iListener.GetBackendGroupId()
		if len(backendgroupId) > 0 {
			params, err := lblis.GetHuaweiLoadbalancerListenerParams()
			if err != nil {
				return nil, err
			}

			params.BackendGroupID = ""
			err = iListener.Sync(params)
			if err != nil {
				return nil, err
			}

			_cachedLbbg, err := db.FetchByExternalId(models.HuaweiCachedLbbgManager, backendgroupId)
			if err != nil {
				return nil, err
			}

			cachedLbbg := _cachedLbbg.(*models.SHuaweiCachedLbbg)
			_, err = db.UpdateWithLock(ctx, cachedLbbg, func() error {
				cachedLbbg.AssociatedId = ""
				cachedLbbg.AssociatedType = ""
				return nil
			})
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

			acl := lblis.GetLoadbalancerAcl()
			if acl != nil {
				err := db.DeleteModel(ctx, userCred, acl)
				if err != nil {
					return nil, err
				}
			}
		}

		return nil, iListener.Delete()
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		iRegion, err := lbbg.GetIRegion()
		if err != nil {
			return nil, err
		}
		loadbalancer := lbbg.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}

		cachedLbbgs, err := models.HuaweiCachedLbbgManager.GetCachedBackendGroups(lbbg.GetId())
		if err != nil {
			return nil, err
		}

		for _, cachedLbbg := range cachedLbbgs {
			if len(cachedLbbg.ExternalId) == 0 {
				continue
			}

			iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedLbbg.ExternalId)
			if err != nil {
				if err == cloudprovider.ErrNotFound {
					continue
				}
				return nil, err
			}

			cachedLbbs, err := cachedLbbg.GetCachedBackends()
			if err != nil {
				return nil, err
			}

			for _, cachedLbb := range cachedLbbs {
				if len(cachedLbb.ExternalId) == 0 {
					continue
				}

				_lbb, err := db.FetchById(models.LoadbalancerBackendManager, cachedLbb.BackendId)
				if err != nil {
					return nil, err
				}

				lbb := _lbb.(*models.SLoadbalancerBackend)
				iLoadbalancerBackendGroup.RemoveBackendServer(cachedLbb.ExternalId, lbb.Weight, lbb.Port)

				cachedLbb.SetModelManager(models.HuaweiCachedLbManager, &cachedLbb)
				err = db.DeleteModel(ctx, userCred, &cachedLbb)
				if err != nil {
					return nil, err
				}
			}

			err = iLoadbalancerBackendGroup.Delete()
			if err != nil {
				return nil, err
			}

			cachedLbbg.SetModelManager(models.HuaweiCachedLbbgManager, &cachedLbbg)
			err = db.DeleteModel(ctx, userCred, &cachedLbbg)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}

		cachedlbbs, err := models.HuaweiCachedLbManager.GetBackendsByLocalBackendId(lbb.GetId())
		if err != nil {
			return nil, err
		}

		for _, cachedlbb := range cachedlbbs {
			cachedlbbg, _ := cachedlbb.GetCachedBackendGroup()
			if cachedlbbg == nil {
				return nil, fmt.Errorf("failed to find lbbg for backend %s", cachedlbb.Name)
			}
			lb := cachedlbbg.GetLoadbalancer()
			if lb == nil {
				return nil, fmt.Errorf("failed to find lb for backendgroup %s", cachedlbbg.Name)
			}
			iRegion, err := lb.GetIRegion()
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

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		params, err := lb.GetCreateLoadbalancerParams(iRegion)
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.CreateILoadBalancer(params)
		if err != nil {
			return nil, err
		}

		lb.SetModelManager(models.LoadbalancerManager, lb)
		if err := db.SetExternalId(lb, userCred, iLoadbalancer.GetGlobalId()); err != nil {
			return nil, err
		}

		{
			// bind eip
			eipId, _ := task.GetParams().GetString("eip_id")
			if len(eipId) > 0 {
				ieip, err := iRegion.GetIEipById(eipId)
				if err != nil {
					return nil, err
				}

				err = ieip.Associate(iLoadbalancer.GetGlobalId())
				if err != nil {
					return nil, err
				}

				eip, err := db.FetchByExternalId(models.ElasticipManager, ieip.GetGlobalId())
				if err != nil {
					return nil, err
				}

				err = eip.(*models.SElasticip).SyncWithCloudEip(ctx, userCred, lb.GetCloudprovider(), ieip, lb.GetOwnerId())
				if err != nil {
					return nil, err
				}
			}
		}

		if err := lb.SyncWithCloudLoadbalancer(ctx, userCred, iLoadbalancer, nil); err != nil {
			return nil, err
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

func (self *SHuaWeiRegionDriver) ValidateDeleteLoadbalancerCondition(ctx context.Context, lb *models.SLoadbalancer) error {
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
		return httperrors.NewConflictError("loadbalancer is using by %d backendgroup.", len(listeners))
	}
	return nil
}

func (self *SHuaWeiRegionDriver) RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		cachedlbbs, err := models.HuaweiCachedLbManager.GetBackendsByLocalBackendId(lbb.GetId())
		if err != nil {
			return nil, err
		}

		for _, cachedlbb := range cachedlbbs {
			cachedlbbg, _ := cachedlbb.GetCachedBackendGroup()
			if cachedlbbg == nil {
				return nil, fmt.Errorf("failed to find lbbg for backend %s", cachedlbb.Name)
			}
			lb := cachedlbbg.GetLoadbalancer()
			if lb == nil {
				return nil, fmt.Errorf("failed to find lb for backendgroup %s", cachedlbbg.Name)
			}
			iRegion, err := lb.GetIRegion()
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

			iBackend, err := iLoadbalancerBackendGroup.GetILoadbalancerBackendById(cachedlbb.ExternalId)
			if err != nil {
				return nil, err
			}

			err = iBackend.SyncConf(lbb.Port, lbb.Weight)
			if err != nil {
				return nil, err
			}

			iBackend, err = iLoadbalancerBackendGroup.GetILoadbalancerBackendById(cachedlbb.ExternalId)
			if err != nil {
				return nil, err
			}

			err = cachedlbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iBackend, nil)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg := lbb.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			return nil, fmt.Errorf("failed to find lbbg for backend %s", lbb.Name)
		}
		lb := lbbg.GetLoadbalancer()
		if lb == nil {
			return nil, fmt.Errorf("failed to find lb for backendgroup %s", lbbg.Name)
		}

		cachedlbbgs, err := models.HuaweiCachedLbbgManager.GetCachedBackendGroups(lbbg.GetId())
		if err != nil {
			return nil, err
		}

		guest := lbb.GetGuest()
		if guest == nil {
			return nil, fmt.Errorf("failed to find guest for lbb %s", lbb.Name)
		}

		var ibackend cloudprovider.ICloudLoadbalancerBackend
		for _, cachedLbbg := range cachedlbbgs {
			iLoadbalancerBackendGroup, err := cachedLbbg.GetICloudLoadbalancerBackendGroup()
			if err != nil {
				return nil, err
			}

			ibackend, err = iLoadbalancerBackendGroup.AddBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
			if err != nil {
				return nil, err
			}

			_, err = models.HuaweiCachedLbManager.CreateHuaweiCachedLb(ctx, userCred, lbb, &cachedLbbg, ibackend, cachedLbbg.GetOwnerId())
			if err != nil {
				return nil, err
			}
		}

		if ibackend != nil {
			if err := lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, ibackend, nil); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
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

			cachedLbbg, err := models.HuaweiCachedLbbgManager.GetCachedBackendGroupByAssociateId(lbr.GetId())
			if err != nil {
				return nil, err
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

		lbr.SetModelManager(models.LoadbalancerListenerRuleManager, lbr)
		if err := db.SetExternalId(lbr, userCred, iListenerRule.GetGlobalId()); err != nil {
			return nil, err
		}
		return nil, lbr.SyncWithCloudLoadbalancerListenerRule(ctx, userCred, iListenerRule, nil)
	})
	return nil
}

func (self *SHuaWeiRegionDriver) DealNatGatewaySpec(spec string) string {
	switch spec {
	case "1":
		return api.NAT_SPEC_SMALL
	case "2":
		return api.NAT_SPEC_MIDDLE
	case "3":
		return api.NAT_SPEC_LARGE
	case "4":
		return api.NAT_SPEC_XLARGE
	}
	//can't arrive
	return ""
}
