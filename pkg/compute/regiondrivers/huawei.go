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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/choices"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/seclib2"
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
	data.Set("vpc_id", jsonutils.NewString(vpc.GetId()))
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

		address, err := models.LoadbalancerBackendManager.GetGuestAddress(guest)
		if err != nil {
			return nil, errors.Wrap(err, "huaWeiRegionDriver.ValidateCreateLoadbalancerBackendData.GetGuestAddress")
		}

		data.Set("address", jsonutils.NewString(address))
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
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 50).Default(10),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Default(5),
	}

	if err := RunValidators(keyVHealth, data, false); err != nil {
		return nil, err
	}

	if t, _ := data.Int("health_check_rise"); t > 0 {
		data.Set("health_check_fall", jsonutils.NewInt(t))
	}

	// acl check
	if err := models.LoadbalancerListenerManager.ValidateAcl(aclStatusV, aclTypeV, aclV, data, api.CLOUD_PROVIDER_HUAWEI); err != nil {
		return nil, err
	}

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
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 10).Default(3),
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

func (self *SHuaWeiRegionDriver) createCachedLbbg(lb *models.SLoadbalancer, lblis *models.SLoadbalancerListener, lbr *models.SLoadbalancerListenerRule, lbbg *models.SLoadbalancerBackendGroup) (*models.SHuaweiCachedLbbg, error) {
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

	err := models.HuaweiCachedLbbgManager.TableSpec().Insert(cachedLbbg)
	if err != nil {
		return nil, err
	}

	cachedLbbg.SetModelManager(models.HuaweiCachedLbbgManager, cachedLbbg)
	return cachedLbbg, nil
}

func (self *SHuaWeiRegionDriver) syncCloudlbbs(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, cachedLbbg *models.SHuaweiCachedLbbg, extlbbg cloudprovider.ICloudLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend) error {
	ibackends, err := extlbbg.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "HuaWeiRegionDriver.syncCloudLoadbalancerBackends.GetILoadbalancerBackends")
	}

	for i := range ibackends {
		ibackend := ibackends[i]
		err = extlbbg.RemoveBackendServer(ibackend.GetId(), ibackend.GetWeight(), ibackend.GetPort())
		if err != nil {
			return errors.Wrap(err, "HuaWeiRegionDriver.syncCloudLoadbalancerBackends.RemoveBackendServer")
		}
	}

	for _, backend := range backends {
		_, err = extlbbg.AddBackendServer(backend.ExternalID, backend.Weight, backend.Port)
		if err != nil {
			return errors.Wrap(err, "HuaWeiRegionDriver.syncCloudLoadbalancerBackends.AddBackendServer")
		}
	}

	return nil
}

func (self *SHuaWeiRegionDriver) syncCachedLbbs(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, lbbg *models.SHuaweiCachedLbbg, extlbbg cloudprovider.ICloudLoadbalancerBackendGroup) error {
	iBackends, err := extlbbg.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "HuaWeiRegionDriver.syncLoadbalancerBackendCaches.GetILoadbalancerBackends")
	}

	if len(iBackends) > 0 {
		provider := lb.GetCloudprovider()
		if provider == nil {
			return fmt.Errorf("failed to find cloudprovider for lb %s", lb.Name)
		}

		result := models.HuaweiCachedLbManager.SyncLoadbalancerBackends(ctx, userCred, provider, lbbg, iBackends, &models.SSyncRange{})
		if result.IsError() {
			return errors.Wrap(result.AllError(), "HuaWeiRegionDriver.syncLoadbalancerBackendCaches.SyncLoadbalancerBackends")
		}
	}

	return nil
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

	cachedLbbg, err := self.createCachedLbbg(lb, lblis, lbr, lbbg)
	if err != nil {
		return nil, errors.Wrap(err, "HuaWeiRegionDriver.createLoadbalancerBackendGroupCache")
	}

	group, err := lbbg.GetHuaweiBackendGroupParams(lblis, lbr)
	if err != nil {
		return nil, err
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
		return nil, errors.Wrap(err, "HuaWeiRegionDriver.createLoadbalancerBackendGroup.syncCloudLoadbalancerBackends")
	}

	err = self.syncCachedLbbs(ctx, userCred, lb, cachedLbbg, iLoadbalancerBackendGroup)
	if err != nil {
		return nil, errors.Wrap(err, "HuaWeiRegionDriver.createLoadbalancerBackendGroup.syncLoadbalancerBackendCaches")
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
		AccessControlEnable: listener.AclStatus == api.LB_BOOL_ON,
	}

	_originAcl, err := db.FetchById(models.LoadbalancerAclManager, lbacl.AclId)
	if err != nil {
		return nil, errors.Wrap(err, "huaWeiRegionDriver.FetchAcl")
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
	return nil, lbacl.SyncWithCloudLoadbalancerAcl(ctx, userCred, iLoadbalancerAcl, listener.GetOwnerId())
}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.createLoadbalancerAcl(ctx, userCred, lbacl)
	})
	return nil
}

func (self *SHuaWeiRegionDriver) updateCachedLbbg(ctx context.Context, cachedLbbg *models.SHuaweiCachedLbbg, backendGroupId string, externalBackendGroupId string, asscoicateId string, asscoicateType string) error {
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

func (self *SHuaWeiRegionDriver) validatorCachedLbbgConflict(ilb cloudprovider.ICloudLoadbalancer, lblis *models.SLoadbalancerListener, olbbg *models.SHuaweiCachedLbbg, nlbbg *models.SHuaweiCachedLbbg, rlbbg *models.SHuaweiCachedLbbg) error {
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

func (self *SHuaWeiRegionDriver) removeCachedLbbg(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SHuaweiCachedLbbg) error {
	backends, err := lbbg.GetCachedBackends()
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "huaweiRegionDriver.GetCachedBackends")
	}
	for i := range backends {
		err = db.DeleteModel(ctx, userCred, &backends[i])
		if err != nil {
			return errors.Wrap(err, "huaweiRegionDriver.DeleteModel")
		}
	}

	err = db.DeleteModel(ctx, userCred, lbbg)
	if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.DeleteModel")
	}

	return nil
}

func (self *SHuaWeiRegionDriver) RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg := lblis.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			err := fmt.Errorf("failed to find lbbg for lblis %s", lblis.Name)
			return nil, errors.Wrap(err, "HuaWeiRegionDriver.RequestSyncLoadbalancerbackendGroup.GetLoadbalancerBackendGroup")
		}

		iRegion, err := lbbg.GetIRegion()
		if err != nil {
			return nil, err
		}

		lb := lbbg.GetLoadbalancer()
		ilb, err := iRegion.GetILoadBalancerById(lb.GetExternalId())
		if err != nil {
			return nil, err
		}

		provider := lb.GetCloudprovider()
		if provider == nil {
			return nil, errors.Wrap(fmt.Errorf("provider is nil"), "HuaWeiRegionDriver.RequestSyncLoadbalancerBackendGroup.GetCloudprovider")
		}

		ilisten, err := ilb.GetILoadBalancerListenerById(lblis.GetExternalId())
		if err != nil {
			return nil, err
		}

		// new group params
		groupInput, err := lbbg.GetHuaweiBackendGroupParams(lblis, nil)
		if err != nil {
			return nil, errors.Wrap(err, "GetHuaweiBackendGroupParams")
		}

		// new backends params
		backendsInput, err := lbbg.GetBackendsParams()
		if err != nil {
			return nil, err
		}

		{
			var olbbg, nlbbg, rlbbg *models.SHuaweiCachedLbbg
			// current related cachedLbbg
			olbbg, err := models.HuaweiCachedLbbgManager.GetCachedBackendGroupByAssociateId(lblis.GetId())
			if err != nil {
				if err != sql.ErrNoRows {
					return nil, errors.Wrap(fmt.Errorf("provider is nil"), "HuaWeiRegionDriver.RequestSyncLoadbalancerBackendGroup.GetCachedBackendGroupByAssociateId")
				}
			}

			// new related backendgroup
			if olbbg == nil || olbbg.BackendGroupId != lbbg.GetId() {
				nlbbg, err = models.HuaweiCachedLbbgManager.GetUsableCachedBackendGroup(lbbg.GetId(), lblis.ListenerType)
				if err != nil {
					if err != sql.ErrNoRows {
						return nil, errors.Wrap(fmt.Errorf("provider is nil"), "HuaWeiRegionDriver.RequestSyncLoadbalancerBackendGroup.GetUsableCachedBackendGroup")
					}
				}
			}

			// remote relasted backendgroup
			rlbbgId := ilisten.GetBackendGroupId()
			if len(rlbbgId) > 0 {
				_rlbbg, err := db.FetchByExternalId(models.HuaweiCachedLbbgManager, rlbbgId)
				if err != nil {
					if err != sql.ErrNoRows {
						return nil, errors.Wrap(err, "HuaWeiRegionDriver.RequestSyncLoadbalancerBackendGroup.FetchByExternalId")
					}
				}

				rlbbg = _rlbbg.(*models.SHuaweiCachedLbbg)
			}

			// validator confilct
			err = self.validatorCachedLbbgConflict(ilb, lblis, olbbg, nlbbg, rlbbg)
			if err != nil {
				return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.validatorCachedLbbgConflict")
			}

			// case 1：新绑定,创建本地缓存，并创建远端服务器组
			if olbbg == nil && nlbbg == nil && rlbbg == nil {
				if _, err := self.createLoadbalancerBackendGroup(ctx, task.GetUserCred(), lblis, nil, lbbg, backendsInput); err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case1.createLoadbalancerBackendGroup")
				}
			}

			// case 2：新绑定,创建本地缓存
			if olbbg == nil && nlbbg == nil && rlbbg != nil {
				cachedLbbg, err := self.createCachedLbbg(lb, lblis, nil, lbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case2.createCachedLbbg")
				}

				err = db.SetExternalId(cachedLbbg, userCred, rlbbgId)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case2.SetExternalId")
				}

				ilbbg, err := ilb.GetILoadBalancerBackendGroupById(rlbbgId)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case2.GetILoadBalancerBackendGroupById")
				}

				err = self.syncCloudlbbs(ctx, userCred, lb, cachedLbbg, ilbbg, backendsInput)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case2.syncCloudlbbs")
				}

				err = self.syncCachedLbbs(ctx, userCred, lb, cachedLbbg, ilbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case2.syncCloudlbbs")
				}
			}

			// case 3：新绑定, 关联并同步
			if olbbg == nil && nlbbg != nil && rlbbg == nil {
				ilbbg, err := ilb.GetILoadBalancerBackendGroupById(nlbbg.GetExternalId())
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case4.GetILoadBalancerBackendGroupById")
				}

				err = db.SetExternalId(nlbbg, userCred, ilbbg.GetGlobalId())
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case3.SetExternalId")
				}

				err = self.syncCloudlbbs(ctx, userCred, lb, nlbbg, ilbbg, backendsInput)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case3.syncCloudlbbs")
				}

				err = self.syncCachedLbbs(ctx, userCred, lb, nlbbg, ilbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case3.syncCloudlbbs")
				}
			}

			// case 4：新绑定, 关联rlbbg并同步
			if olbbg == nil && nlbbg != nil && rlbbg != nil {
				ilbbg, err := ilb.GetILoadBalancerBackendGroupById(rlbbgId)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case4.GetILoadBalancerBackendGroupById")
				}

				err = db.SetExternalId(nlbbg, userCred, ilbbg.GetGlobalId())
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case4.SetExternalId")
				}

				err = self.syncCloudlbbs(ctx, userCred, lb, nlbbg, ilbbg, backendsInput)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case4.syncCloudlbbs")
				}

				err = self.syncCachedLbbs(ctx, userCred, lb, nlbbg, ilbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case4.syncCloudlbbs")
				}
			}

			// case 5：已绑定,更新原有缓存
			if olbbg != nil && nlbbg == nil && rlbbg == nil {
				ilbbg, err := ilb.CreateILoadBalancerBackendGroup(groupInput)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case5.CreateILoadBalancerBackendGroup")
				}

				err = self.updateCachedLbbg(ctx, olbbg, lbbg.GetId(), ilbbg.GetGlobalId(), "", "")
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case5.updateCachedLbbg")
				}

				err = self.syncCloudlbbs(ctx, userCred, lb, olbbg, ilbbg, backendsInput)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case4.syncCloudlbbs")
				}

				err = self.syncCachedLbbs(ctx, userCred, lb, olbbg, ilbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case4.syncCloudlbbs")
				}
			}

			// case 6：已绑定, 更新olbbg并同步
			if olbbg != nil && nlbbg == nil && rlbbg != nil {
				ilbbg, err := ilb.GetILoadBalancerBackendGroupById(rlbbgId)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case6.GetILoadBalancerBackendGroupById")
				}

				err = self.updateCachedLbbg(ctx, olbbg, lbbg.GetId(), ilbbg.GetGlobalId(), "", "")
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case6.updateCachedLbbg")
				}

				err = self.syncCloudlbbs(ctx, userCred, lb, olbbg, ilbbg, backendsInput)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case6.syncCloudlbbs")
				}

				err = self.syncCachedLbbs(ctx, userCred, lb, olbbg, ilbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case6.syncCloudlbbs")
				}
			}

			// case 7：已绑定, 更新olbbg并同步
			if olbbg != nil && nlbbg != nil && rlbbg == nil {
				err := self.removeCachedLbbg(ctx, userCred, olbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case7.removeCachedLbbg")
				}

				err = self.updateCachedLbbg(ctx, nlbbg, lbbg.GetId(), "", lblis.GetId(), api.LB_ASSOCIATE_TYPE_LISTENER)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case7.updateCachedLbbg")
				}

				ilbbg, err := ilb.GetILoadBalancerBackendGroupById(nlbbg.GetExternalId())
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case7.GetILoadBalancerBackendGroupById")
				}

				err = self.syncCloudlbbs(ctx, userCred, lb, nlbbg, ilbbg, backendsInput)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case6.syncCloudlbbs")
				}

				err = self.syncCachedLbbs(ctx, userCred, lb, nlbbg, ilbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case6.syncCloudlbbs")
				}
			}

			// case 8：已绑定, 更新olbbg并同步
			if olbbg != nil && nlbbg != nil && rlbbg != nil {
				ilbbg, err := ilb.GetILoadBalancerBackendGroupById(rlbbgId)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case8.GetILoadBalancerBackendGroupById")
				}

				err = deleteHuaweiLoadbalancerBackendGroup(ctx, userCred, ilb, ilbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case8.deleteHuaweiLoadbalancerBackendGroup")
				}

				err = deleteHuaweiCachedLbbg(ctx, userCred, lblis.GetId())
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case8.deleteHuaweiCachedLbbg")
				}

				err = self.updateCachedLbbg(ctx, nlbbg, lbbg.GetId(), "", lblis.GetId(), api.LB_ASSOCIATE_TYPE_LISTENER)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case8.updateCachedLbbg")
				}

				err = self.syncCloudlbbs(ctx, userCred, lb, nlbbg, ilbbg, backendsInput)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case6.syncCloudlbbs")
				}

				err = self.syncCachedLbbs(ctx, userCred, lb, nlbbg, ilbbg)
				if err != nil {
					return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.Case6.syncCloudlbbs")
				}
			}
		}

		// continue here
		cachedLbbg, err := models.HuaweiCachedLbbgManager.GetCachedBackendGroupByAssociateId(lblis.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.GetCachedBackendGroupByAssociateId")
		}

		ilbbg, err := ilb.GetILoadBalancerBackendGroupById(cachedLbbg.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.GetILoadBalancerBackendGroupById")
		}

		err = ilbbg.Sync(groupInput)
		if err != nil {
			return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.LoadbalancerBackendGroup")
		}

		if err := cachedLbbg.SyncWithCloudLoadbalancerBackendgroup(ctx, task.GetUserCred(), lb, ilbbg, lb.GetOwnerId()); err != nil {
			return nil, errors.Wrap(err, "HuaWeiRegionDriver.Sync.SyncWithCloudLoadbalancerBackendgroup")
		}

		return nil, nil
	})

	return nil
}

func (self *SHuaWeiRegionDriver) RequestPullRegionLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) error {
	return nil
}

func (self *SHuaWeiRegionDriver) RequestPullLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localLoadbalancer *models.SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *models.SSyncRange) error {
	models.SyncHuaweiLoadbalancerBackendgroups(ctx, userCred, syncResults, provider, localLoadbalancer, remoteLoadbalancer, syncRange)
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
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.FetchById")
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.GetOrCreateCachedCertificate")
				}

				if len(lbcert.ExternalId) == 0 {
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.createLoadbalancerCertificate")
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedCertificateId = lbcert.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.UpdateCachedCertificateId")
				}
			}
		}

		params, err := lblis.GetHuaweiLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.GetHuaweiLoadbalancerListenerParams")
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.GetILoadBalancerById")
		}
		iListener, err := iLoadbalancer.CreateILoadBalancerListener(params)
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.CreateILoadBalancerListener")
		}

		lblis.SetModelManager(models.LoadbalancerListenerManager, lblis)
		if err := db.SetExternalId(lblis, userCred, iListener.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.SetExternalId")
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
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.FetchAclById")
				}

				lbacl, err := models.CachedLoadbalancerAclManager.GetOrCreateCachedAcl(ctx, userCred, provider, lblis, acl.(*models.SLoadbalancerAcl))
				if err != nil {
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.GetOrCreateCachedAcl")
				}

				if len(lbacl.ExternalId) == 0 {
					_, err = self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.createLoadbalancerAcl")
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedAclId = lbacl.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListener.UpdateCachedAclId")
				}
			}
		}

		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, loadbalancer.GetOwnerId())
	})
	return nil
}

func (self *SHuaWeiRegionDriver) syncLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl) (jsonutils.JSONObject, error) {
	iRegion, err := lbacl.GetIRegion()
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
		return nil, fmt.Errorf("huaweiRegionDriver.syncLoadbalancerAcl %s", err)
	}

	_localAcl, err := db.FetchById(models.LoadbalancerAclManager, lbacl.AclId)
	if err != nil {
		return nil, errors.Wrap(err, "huaweiRegionDriver.FetchById.LoaclAcl")
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

				_, err = db.Update(lblis, func() error {
					lblis.CachedCertificateId = lbcert.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerListener.UpdateCachedCertificateId")
				}
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

				_, err = db.Update(lblis, func() error {
					lblis.CachedAclId = lbacl.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerListener.UpdateCachedAclId")
				}
			}
		}

		if err := iListener.Refresh(); err != nil {
			return nil, err
		}
		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, lblis.GetOwnerId())
	})
	return nil
}

func deleteHuaweiLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, ilb cloudprovider.ICloudLoadbalancer, irule cloudprovider.ICloudLoadbalancerListenerRule) error {
	err := irule.Refresh()
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			return nil
		}

		return errors.Wrap(err, "HuaWeiRegionDriver.Rule.Refresh")
	}

	lbbgId := irule.GetBackendGroupId()

	err = irule.Delete()
	if err != nil && err != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "HuaWeiRegionDriver.Rule.Delete")
	}

	// delete backendgroup
	if len(lbbgId) > 0 {
		ilbbg, err := ilb.GetILoadBalancerBackendGroupById(lbbgId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil
			}

			return errors.Wrap(err, "HuaWeiRegionDriver.Rule.GetILoadBalancerBackendGroupById")
		}

		err = deleteHuaweiLoadbalancerBackendGroup(ctx, userCred, ilb, ilbbg)
		if err != nil {
			return errors.Wrap(err, "HuaWeiRegionDriver.Rule.deleteHuaweiLoadbalancerBackendGroup")
		}
	}

	return nil
}

func deleteHuaweiLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, ilb cloudprovider.ICloudLoadbalancer, ilbbg cloudprovider.ICloudLoadbalancerBackendGroup) error {
	err := ilbbg.Refresh()
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			return nil
		}

		return errors.Wrap(err, "HuaWeiRegionDriver.BackendGroup.Refresh")
	}

	ibackends, err := ilbbg.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "HuaWeiRegionDriver.BackendGroup.GetILoadbalancerBackends")
	}

	for i := range ibackends {
		ilbb := ibackends[i]
		err = deleteHuaweiLoadbalancerBackend(ctx, userCred, ilb, ilbbg, ilbb)
		if err != nil {
			return errors.Wrap(err, "HuaWeiRegionDriver.BackendGroup.deleteHuaweiLoadbalancerBackend")
		}
	}

	err = ilbbg.Delete()
	if err != nil && err != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "HuaWeiRegionDriver.BackendGroup.Delete")
	}

	return nil
}

func deleteHuaweiLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, ilb cloudprovider.ICloudLoadbalancer, ilbbg cloudprovider.ICloudLoadbalancerBackendGroup, ilbb cloudprovider.ICloudLoadbalancerBackend) error {
	err := ilbbg.RemoveBackendServer(ilbb.GetId(), ilbb.GetWeight(), ilbb.GetPort())
	if err != nil && err != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "HuaWeiRegionDriver.Backend.Delete")
	}

	return nil
}

func deleteHuaweiLblisRule(ctx context.Context, userCred mcclient.TokenCredential, ruleId string) error {
	rule, err := db.FetchById(models.LoadbalancerListenerRuleManager, ruleId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		return errors.Wrap(err, "deleteHuaweiLblisRule.FetchById")
	}

	err = deleteHuaweiCachedLbbg(ctx, userCred, ruleId)
	if err != nil {
		return errors.Wrap(err, "deleteHuaweiLblisRule.deleteHuaweiCachedLbbg")
	}

	err = db.DeleteModel(ctx, userCred, rule)
	if err != nil {
		return errors.Wrap(err, "deleteHuaweiLblisRule.DeleteModel")
	}

	return nil
}

func deleteHuaweiCachedLbbg(ctx context.Context, userCred mcclient.TokenCredential, associatedId string) error {
	lbbg, err := models.HuaweiCachedLbbgManager.GetCachedBackendGroupByAssociateId(associatedId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		return errors.Wrap(err, "deleteHuaweiCachedLbbg.GetCachedBackendGroupByAssociateId")
	}

	if err := deleteHuaweiCachedLbbsByLbbg(ctx, userCred, lbbg.GetId()); err != nil {
		return errors.Wrap(err, "deleteHuaweiCachedLbbg.deleteHuaweiCachedLbbsByLbbg")
	}

	err = db.DeleteModel(ctx, userCred, lbbg)
	if err != nil {
		return errors.Wrap(err, "deleteHuaweiCachedLbbg.DeleteModel")
	}

	return nil
}

func deleteHuaweiCachedLbbsByLbbg(ctx context.Context, userCred mcclient.TokenCredential, cachedLbbgId string) error {
	cachedLbbs := []models.SHuaweiCachedLb{}
	q := models.HuaweiCachedLbManager.Query().IsFalse("pending_deleted").Equals("cached_backend_group_id", cachedLbbgId)
	err := db.FetchModelObjects(models.HuaweiCachedLbManager, q, &cachedLbbs)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		return errors.Wrap(err, "deleteHuaweiCachedLbbsByLbbg.FetchModelObjects")
	}

	for i := range cachedLbbs {
		cachedLbb := cachedLbbs[i]
		err = db.DeleteModel(ctx, userCred, &cachedLbb)
		if err != nil {
			return errors.Wrap(err, "deleteHuaweiCachedLbbsByLbbg.DeleteModel")
		}
	}

	return nil
}

func deleteHuaweiCachedLbb(ctx context.Context, userCred mcclient.TokenCredential, cachedLbbId string) error {
	lbb, err := db.FetchById(models.HuaweiCachedLbManager, cachedLbbId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		return errors.Wrap(err, "deleteHuaweiCachedLbb.FetchById")
	}

	err = db.DeleteModel(ctx, userCred, lbb)
	if err != nil {
		return errors.Wrap(err, "deleteHuaweiCachedLbb.DeleteModel")
	}

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
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}

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
			} else {
				iListener.Refresh()
			}

			// 删除后端服务器组
			ilbbg, err := iLoadbalancer.GetILoadBalancerBackendGroupById(backendgroupId)
			if err != nil {
				return nil, errors.Wrap(err, "HuaWeiRegionDriver.RequestDeleteLoadbalancerListener.GetILoadBalancerBackendGroup")
			}

			err = deleteHuaweiLoadbalancerBackendGroup(ctx, userCred, iLoadbalancer, ilbbg)
			if err != nil {
				return nil, errors.Wrap(err, "HuaWeiRegionDriver.RequestDeleteLoadbalancerListener.DeleteBackendGroup")
			}
		}

		err = deleteHuaweiCachedLbbg(ctx, userCred, lblis.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "HuaWeiRegionDriver.RequestDeleteLoadbalancerListener.deleteHuaweiCachedLbbg")
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
			return nil, errors.Wrap(err, "HuaWeiRegionDriver.GetILoadbalancerListenerRules")
		}

		for i := range irules {
			irule := irules[i]
			err = deleteHuaweiLoadbalancerListenerRule(ctx, userCred, iLoadbalancer, irule)
			if err != nil {
				return nil, errors.Wrap(err, "HuaWeiRegionDriver.deleteHuaweiLoadbalancerListenerRule")
			}
		}

		rules, err := lblis.GetLoadbalancerListenerRules()
		if err != nil && err != sql.ErrNoRows {
			return nil, errors.Wrap(err, "HuaWeiRegionDriver.GetLoadbalancerListenerRules")
		}

		for i := range rules {
			rule := rules[i]
			if err := deleteHuaweiLblisRule(ctx, userCred, rule.GetId()); err != nil {
				return nil, errors.Wrap(err, "HuaWeiRegionDriver.Rule.deleteHuaweiCachedLbbg")
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
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.")
		}
		loadbalancer := lbbg.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}

			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.GetILoadBalancerById")
		}

		cachedLbbgs, err := models.HuaweiCachedLbbgManager.GetCachedBackendGroups(lbbg.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.GetCachedBackendGroups")
		}

		for _, cachedLbbg := range cachedLbbgs {
			if len(cachedLbbg.ExternalId) == 0 {
				continue
			}

			iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedLbbg.ExternalId)
			if err != nil {
				if err == cloudprovider.ErrNotFound {
					if err := deleteHuaweiCachedLbbg(ctx, userCred, cachedLbbg.AssociatedId); err != nil {
						return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.deleteHuaweiCachedLbbg")
					}

					continue
				}
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.GetILoadBalancerBackendGroupById")
			}

			ilbbs, err := iLoadbalancerBackendGroup.GetILoadbalancerBackends()
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.GetILoadbalancerBackends")
			}

			for _, ilbb := range ilbbs {
				iLoadbalancerBackendGroup.RemoveBackendServer(ilbb.GetId(), ilbb.GetWeight(), ilbb.GetPort())

				_cachedLbb, err := db.FetchByExternalId(models.HuaweiCachedLbManager, ilbb.GetGlobalId())
				if err != nil {
					if err != sql.ErrNoRows {
						return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.FetchByExternalId")
					}
					continue
				}

				cachedLbb := _cachedLbb.(*models.SHuaweiCachedLb)
				err = db.DeleteModel(ctx, userCred, cachedLbb)
				if err != nil {
					return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.DeleteModel")
				}
			}

			err = iLoadbalancerBackendGroup.Delete()
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.Delete")
			}

			cachedLbbg.SetModelManager(models.HuaweiCachedLbbgManager, &cachedLbbg)
			err = db.DeleteModel(ctx, userCred, &cachedLbbg)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestDeleteLoadbalancerBackendGroup.DeleteModel")
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
				log.Warningf("failed to find lbbg for backend %s", cachedlbb.Name)
				continue
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

				conf := &cloudprovider.AssociateConfig{
					InstanceId:    iLoadbalancer.GetGlobalId(),
					AssociateType: api.EIP_ASSOCIATE_TYPE_LOADBALANCER,
				}

				err = ieip.Associate(conf)
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
		return httperrors.NewConflictError("loadbalancer is using by %d backendgroup.", len(lbbgs))
	}
	return nil
}

func (self *SHuaWeiRegionDriver) RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		cachedlbbs, err := models.HuaweiCachedLbManager.GetBackendsByLocalBackendId(lbb.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerBackend.GetBackendsByLocalBackendId")
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
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerBackend.GetIRegion")
			}
			iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerBackend.GetILoadBalancerById")
			}
			iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedlbbg.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerBackend.GetILoadBalancerBackendGroupById")
			}

			iBackend, err := iLoadbalancerBackendGroup.GetILoadbalancerBackendById(cachedlbb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerBackend.GetILoadbalancerBackendById")
			}

			err = iBackend.SyncConf(lbb.Port, lbb.Weight)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerBackend.SyncConf")
			}

			iBackend, err = iLoadbalancerBackendGroup.GetILoadbalancerBackendById(cachedlbb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerBackend.GetILoadbalancerBackendById")
			}

			err = cachedlbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iBackend, nil)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestSyncLoadbalancerBackend.SyncWithCloudLoadbalancerBackend")
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
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerBackend.GetCachedBackendGroups")
		}

		guest := lbb.GetGuest()
		if guest == nil {
			return nil, fmt.Errorf("failed to find guest for lbb %s", lbb.Name)
		}

		var ibackend cloudprovider.ICloudLoadbalancerBackend
		for _, cachedLbbg := range cachedlbbgs {
			iLoadbalancerBackendGroup, err := cachedLbbg.GetICloudLoadbalancerBackendGroup()
			if err != nil {
				if err == cloudprovider.ErrNotFound {
					if err := deleteHuaweiCachedLbbg(ctx, userCred, cachedLbbg.AssociatedId); err != nil {
						return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerBackend.deleteHuaweiCachedLbbg")
					}

					continue
				}

				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerBackend.GetICloudLoadbalancerBackendGroup")
			}

			ibackend, err = iLoadbalancerBackendGroup.AddBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerBackend.AddBackendServer")
			}

			_, err = models.HuaweiCachedLbManager.CreateHuaweiCachedLb(ctx, userCred, lbb, &cachedLbbg, ibackend, cachedLbbg.GetOwnerId())
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerBackend.CreateHuaweiCachedLb")
			}
		}

		if ibackend != nil {
			if err := lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, ibackend, nil); err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerBackend.SyncWithCloudLoadbalancerBackend")
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
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListenerRule.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListenerRule.GetILoadBalancerById")
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(listener.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListenerRule.GetILoadBalancerListenerById")
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
				return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListenerRule.GetCachedBackendGroupByAssociateId")
			}

			if cachedLbbg == nil {
				return nil, fmt.Errorf("usable cached backend group not found")
			}

			rule.BackendGroupID = cachedLbbg.ExternalId
			rule.BackendGroupType = group.Type
		}
		iListenerRule, err := iListener.CreateILoadBalancerListenerRule(rule)
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListenerRule.CreateILoadBalancerListenerRule")
		}
		//
		lbr.SetModelManager(models.LoadbalancerListenerRuleManager, lbr)
		if err := db.SetExternalId(lbr, userCred, iListenerRule.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.RequestCreateLoadbalancerListenerRule.SetExternalId")
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
	// can't arrive
	return ""
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

	return input, nil
}

func (self *SHuaWeiRegionDriver) InitDBInstanceUser(instance *models.SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error {
	if len(desc.Password) == 0 {
		desc.Password = seclib2.RandomPassword2(12)
	}

	user := "root"
	if desc.Engine == api.DBINSTANCE_TYPE_SQLSERVER {
		user = "rdsuser"
	}

	account := models.SDBInstanceAccount{
		DBInstanceId: instance.Id,
	}
	account.Name = user
	account.Status = api.DBINSTANCE_USER_AVAILABLE
	account.ExternalId = user
	account.SetModelManager(models.DBInstanceAccountManager, &account)
	err := models.DBInstanceAccountManager.TableSpec().Insert(&account)
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
	if len(input.Category) > 0 && input.Category != instance.Category {
		return httperrors.NewUnsupportOperationError("Huawei DBInstance category cannot change")
	}
	if len(input.StorageType) > 0 && input.StorageType != instance.StorageType {
		return httperrors.NewUnsupportOperationError("Huawei DBInstance storage type cannot change")
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

func validatorSlaveZones(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, optional bool) error {
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
	for _, zone := range zones {
		_data := jsonutils.NewDict()
		_data.Add(jsonutils.NewString(zone), "zone")
		if err := zoneV.Validate(_data); err != nil {
			return errors.Wrap(err, "validatorSlaveZones")
		} else {
			ret = append(ret, zoneV.Model.GetId())
		}
	}

	if sku, err := data.GetString("sku"); err != nil || len(sku) == 0 {
		return httperrors.NewMissingParameterError("sku")
	} else {
		chargeType, _ := data.GetString("charge_type")

		_skuModel, err := db.FetchByIdOrName(models.ElasticcacheSkuManager, ownerId, sku)
		if err != nil {
			return err
		}

		skuModel := _skuModel.(*models.SElasticcacheSku)
		for _, zoneId := range zones {
			if err := ValidateElasticcacheSku(zoneId, chargeType, skuModel); err != nil {
				return err
			}
		}
	}

	data.Set("slave_zones", jsonutils.NewString(strings.Join(ret, ",")))
	return nil
}

func ValidateElasticcacheSku(zoneId string, chargeType string, sku *models.SElasticcacheSku) error {
	if sku.ZoneId != zoneId {
		return httperrors.NewResourceNotFoundError("zone mismatch, elastic cache sku zone %s != %s", sku.ZoneId, zoneId)
	}

	if chargeType == billing_api.BILLING_TYPE_PREPAID {
		if sku.PrepaidStatus != api.SkuStatusAvailable {
			return httperrors.NewOutOfResourceError("sku %s is soldout", sku.Name)
		}
	} else {
		if sku.PostpaidStatus != api.SkuStatusAvailable {
			return httperrors.NewOutOfResourceError("sku %s is soldout", sku.Name)
		}
	}

	return nil
}

func (self *SHuaWeiRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	networkV := validators.NewModelIdOrNameValidator("network", "network", ownerId)
	// secgroupV := validators.NewModelIdOrNameValidator("security_group", "secgroup", ownerId)

	// todo: fix me
	instanceTypeV := validators.NewModelIdOrNameValidator("instance_type", "elasticcachesku", ownerId)
	chargeTypeV := validators.NewStringChoicesValidator("billing_type", choices.NewChoices(billing_api.BILLING_TYPE_PREPAID, billing_api.BILLING_TYPE_POSTPAID))
	networkTypeV := validators.NewStringChoicesValidator("network_type", choices.NewChoices(api.LB_NETWORK_TYPE_VPC, api.LB_NETWORK_TYPE_CLASSIC)).Default(api.LB_NETWORK_TYPE_VPC).Optional(true)
	engineV := validators.NewStringChoicesValidator("engine", choices.NewChoices("redis", "memcache"))
	engineVersionV := validators.NewStringChoicesValidator("engine_version", choices.NewChoices("3.0", "4.0", "5.0"))
	privateIpV := validators.NewIPv4AddrValidator("private_ip").Optional(true)
	maintainStartTimeV := validators.NewStringChoicesValidator("maintain_start_time", choices.NewChoices("22:00:00", "02:00:00", "06:00:00", "10:00:00", "14:00:00", "18:00:00")).Default("02:00:00").Optional(true)

	keyV := map[string]validators.IValidator{
		"zone":          zoneV,
		"billing_type":  chargeTypeV,
		"network_type":  networkTypeV,
		"network":       networkV,
		"instance_type": instanceTypeV,
		// "security_group":     secgroupV,
		"engine":             engineV,
		"engine_version":     engineVersionV,
		"private_ip":         privateIpV,
		"mantain_start_time": maintainStartTimeV,
	}
	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	// validate password
	if password, _ := data.GetString("password"); len(password) > 0 {
		if !seclib2.MeetComplxity(password) {
			return nil, httperrors.NewWeakPasswordError()
		}
	}

	// validate sku
	sku := instanceTypeV.Model.(*models.SElasticcacheSku)
	zoneId, _ := data.GetString("zone_id")
	billingType, _ := data.GetString("billing_type")

	if err := ValidateElasticcacheSku(zoneId, billingType, sku); err != nil {
		return nil, err
	} else {
		data.Set("instance_type", jsonutils.NewString(sku.InstanceSpec))
		data.Set("node_type", jsonutils.NewString(sku.NodeType))
		data.Set("local_category", jsonutils.NewString(sku.LocalCategory))
		data.Set("capacity_mb", jsonutils.NewInt(int64(sku.MemorySizeMB)))
	}

	// validate slave zones
	if err := validatorSlaveZones(ownerId, data, true); err != nil {
		return nil, err
	}

	// validate capacity
	if capacityMB, err := data.Int("capacity_mb"); err != nil {
		return nil, errors.Wrap(err, "invalid parameter capacity_mb")
	} else {
		// todo: fix me
		// Redis引擎：单机和主备类型实例取值：2、4、8、16、32、64。集群实例规格支持64G、128G、256G、512G和1024G。
		data.Set("capacity_mb", jsonutils.NewInt(capacityMB))
	}

	// MaintainEndTime
	// 开始时间必须为22:00:00、02:00:00、06:00:00、10:00:00、14:00:00和18:00:00。
	data.Remove("mantain_end_time")
	if startTime, err := data.GetString("maintain_start_time"); err == nil {
		maintainTimes := []string{"22:00:00", "02:00:00", "06:00:00", "10:00:00", "14:00:00", "18:00:00", "22:00:00"}

		for i := range maintainTimes {
			if maintainTimes[i] == startTime {
				data.Add(jsonutils.NewString(maintainTimes[i+1]), "mantain_end_time")
				break
			}
		}
	}

	// billing cycle
	if billingType == billing_api.BILLING_TYPE_PREPAID {
		billingCycle, err := data.GetString("billing_cycle")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("billing_cycle")
		}

		cycle, err := billing.ParseBillingCycle(billingCycle)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid billing_cycle %s", billingCycle)
		}

		data.Set("billing_cycle", jsonutils.NewString(cycle.String()))
	}

	network := networkV.Model.(*models.SNetwork)
	vpc := network.GetVpc()
	if vpc == nil {
		return nil, httperrors.NewNotFoundError("network %s related vpc not found", network.GetId())
	}
	data.Set("vpc_id", jsonutils.NewString(vpc.Id))
	data.Set("manager_id", jsonutils.NewString(vpc.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(sku.CloudregionId))
	return data, nil
}

func (self *SHuaWeiRegionDriver) RequestCreateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := ec.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.GetIRegion")
		}

		iprovider, err := db.FetchById(models.CloudproviderManager, ec.ManagerId)
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.GetProvider")
		}

		provider := iprovider.(*models.SCloudprovider)

		params, err := ec.GetCreateHuaweiElasticcacheParams(task.GetParams())
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.GetCreateHuaweiElasticcacheParams")
		}

		iec, err := iRegion.CreateIElasticcaches(params)
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.CreateIElasticcaches")
		}

		err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 30*time.Second, 15*time.Second, 600*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.WaitStatusWithDelay")
		}

		ec.SetModelManager(models.ElasticcacheManager, ec)
		if err := db.SetExternalId(ec, userCred, iec.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.SetExternalId")
		}

		{
			// todo: 开启外网访问
		}

		if err := ec.SyncWithCloudElasticcache(ctx, userCred, provider, iec); err != nil {
			return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.SyncWithCloudElasticcache")
		}

		// sync accounts
		{
			iaccounts, err := iec.GetICloudElasticcacheAccounts()
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.GetICloudElasticcacheAccounts")
			}

			result := models.ElasticcacheAccountManager.SyncElasticcacheAccounts(ctx, userCred, ec, iaccounts)
			log.Debugf("huaweiRegionDriver.CreateElasticcache.SyncElasticcacheAccounts %s", result.Result())

			account, err := ec.GetAdminAccount()
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.GetAdminAccount")
			}

			err = account.SavePassword(params.Password)
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.SavePassword")
			}
		}

		// sync acl
		{
			iacls, err := iec.GetICloudElasticcacheAcls()
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.GetICloudElasticcacheAcls")
			}

			result := models.ElasticcacheAclManager.SyncElasticcacheAcls(ctx, userCred, ec, iacls)
			log.Debugf("huaweiRegionDriver.CreateElasticcache.SyncElasticcacheAcls %s", result.Result())
		}

		// sync parameters
		{
			iparams, err := iec.GetICloudElasticcacheParameters()
			if err != nil {
				return nil, errors.Wrap(err, "huaweiRegionDriver.CreateElasticcache.GetICloudElasticcacheParameters")
			}

			result := models.ElasticcacheParameterManager.SyncElasticcacheParameters(ctx, userCred, ec, iparams)
			log.Debugf("huaweiRegionDriver.CreateElasticcache.SyncElasticcacheParameters %s", result.Result())
		}

		return nil, nil
	})
	return nil
}

func (self *SHuaWeiRegionDriver) ValidateCreateElasticcacheAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewUnsupportOperationError("% not support create account", self.GetProvider())
}

func (self *SHuaWeiRegionDriver) RequestElasticcacheAccountResetPassword(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	iregion, err := ea.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "huaweiRegionDriver.RequestElasticcacheAccountResetPassword.FetchById")
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err == cloudprovider.ErrNotFound {
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

	return ea.SetStatus(userCred, api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE, "")
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

func (self *SHuaWeiRegionDriver) GetBackendStatusForAdd() []string {
	return []string{api.VM_RUNNING, api.VM_READY}
}
