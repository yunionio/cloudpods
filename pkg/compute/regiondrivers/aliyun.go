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
	"strings"
	"time"
	"unicode"

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

func (self *SAliyunRegionDriver) validateCreateLBCommonData(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*validators.ValidatorModelIdOrName, *jsonutils.JSONDict, error) {
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", ownerId)
	chargeTypeV := validators.NewStringChoicesValidator("charge_type", choices.NewChoices(api.LB_CHARGE_TYPE_BY_BANDWIDTH, api.LB_CHARGE_TYPE_BY_TRAFFIC))
	chargeTypeV.Default(api.LB_CHARGE_TYPE_BY_TRAFFIC)
	addressTypeV := validators.NewStringChoicesValidator("address_type", api.LB_ADDR_TYPES)
	loadbalancerSpecV := validators.NewStringChoicesValidator("loadbalancer_spec", api.LB_ALIYUN_SPECS)
	loadbalancerSpecV.Default(api.LB_ALIYUN_SPEC_SHAREABLE)

	keyV := map[string]validators.IValidator{
		"status":            validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),
		"charge_type":       chargeTypeV,
		"address_type":      addressTypeV.Default(api.LB_ADDR_TYPE_INTRANET),
		"zone":              zoneV,
		"manager":           managerIdV,
		"loadbalancer_spec": loadbalancerSpecV,
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, nil, err
	}

	if chargeTypeV.Value == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		egressMbps := validators.NewRangeValidator("egress_mbps", 1, 5000)
		if err := egressMbps.Validate(data); err != nil {
			return nil, nil, err
		}
	}

	region := zoneV.Model.(*models.SZone).GetRegion()
	if region == nil {
		return nil, nil, fmt.Errorf("getting region failed")
	}

	data.Set("network_type", jsonutils.NewString(api.LB_NETWORK_TYPE_VPC))
	data.Set("cloudregion_id", jsonutils.NewString(region.GetId()))
	return managerIdV, data, nil
}

func (self *SAliyunRegionDriver) validateCreateIntranetLBData(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	managerIdV, data, err := self.validateCreateLBCommonData(ownerId, data)
	if err != nil {
		return nil, err
	}

	networkV := validators.NewModelIdOrNameValidator("network", "network", ownerId)
	if err := networkV.Validate(data); err != nil {
		return nil, err
	}

	network := networkV.Model.(*models.SNetwork)
	region, zone, vpc, _, err := network.ValidateElbNetwork(nil)
	if err != nil {
		return nil, err
	}

	chargeType, _ := data.GetString("charge_type")
	if chargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		return nil, httperrors.NewUnsupportOperationError("intranet loadbalancer not support bandwidth charge type")
	}

	managerId, _ := data.GetString("manager_id")
	if managerId != vpc.ManagerId {
		return nil, httperrors.NewInputParameterError("Loadbalancer's manager (%s(%s)) does not match vpc's(%s(%s)) (%s)", managerIdV.Model.GetName(), managerIdV.Model.GetId(), vpc.GetName(), vpc.GetId(), vpc.ManagerId)
	}

	data.Set("vpc_id", jsonutils.NewString(vpc.Id))
	data.Set("cloudregion_id", jsonutils.NewString(region.GetId()))
	data.Set("zone_id", jsonutils.NewString(zone.GetId()))
	data.Set("address_type", jsonutils.NewString(api.LB_ADDR_TYPE_INTRANET))
	return data, nil
}

func (self *SAliyunRegionDriver) validateCreateInternetLBData(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	_, data, err := self.validateCreateLBCommonData(ownerId, data)
	if err != nil {
		return nil, err
	}

	// 公网 lb 实例和vpc、network无关联
	data.Set("vpc_id", jsonutils.NewString(""))
	data.Set("address", jsonutils.NewString(""))
	data.Set("network_id", jsonutils.NewString(""))
	data.Set("address_type", jsonutils.NewString(api.LB_ADDR_TYPE_INTERNET))
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ownerId := ctx.Value("ownerId").(mcclient.IIdentityProvider)
	addressTypeV := validators.NewStringChoicesValidator("address_type", api.LB_ADDR_TYPES)
	if err := addressTypeV.Validate(data); err != nil {
		return nil, err
	}

	var validator func(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	if addressTypeV.Value == api.LB_ADDR_TYPE_INTRANET {
		validator = self.validateCreateIntranetLBData
	} else {
		validator = self.validateCreateInternetLBData
	}

	if _, err := validator(ownerId, data); err != nil {
		return nil, err
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, data)
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("certificate") || data.Contains("private_key") {
		return nil, httperrors.NewUnsupportOperationError("Aliyun not allow to change certificate")
	}
	return data, nil
}

func (self *SAliyunRegionDriver) GetBackendStatusForAdd() []string {
	return []string{api.VM_RUNNING}
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backends []cloudprovider.SLoadbalancerBackend) (*jsonutils.JSONDict, error) {
	groupType, _ := data.GetString("type")
	switch groupType {
	case "", api.LB_BACKENDGROUP_TYPE_NORMAL:
		break
	case api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
		if len(backends) != 2 {
			return nil, httperrors.NewInputParameterError("master slave backendgorup must contain two backend")
		}
	default:
		return nil, httperrors.NewInputParameterError("Unsupport backendgorup type %s", groupType)
	}
	for _, backend := range backends {
		if len(backend.ExternalID) == 0 {
			return nil, httperrors.NewInputParameterError("invalid guest %s", backend.Name)
		}
		if backend.Weight < 0 || backend.Weight > 100 {
			return nil, httperrors.NewInputParameterError("Aliyun instance weight must be in the range of 0 ~ 100")
		}
	}
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendType string, lb *models.SLoadbalancer, backendGroup *models.SLoadbalancerBackendGroup, backend db.IModel) (*jsonutils.JSONDict, error) {
	ownerId := ctx.Value("ownerId").(mcclient.IIdentityProvider)
	man := models.LoadbalancerBackendManager
	backendTypeV := validators.NewStringChoicesValidator("backend_type", api.LB_BACKEND_TYPES)
	keyV := map[string]validators.IValidator{
		"backend_type": backendTypeV,
		"weight":       validators.NewRangeValidator("weight", 0, 100).Default(1),
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

	if backendType != api.LB_BACKEND_GUEST {
		return nil, httperrors.NewUnsupportOperationError("internal error: unexpected backend type %s", backendType)
	}
	if !utils.IsInStringArray(backendGroup.Type, []string{api.LB_BACKENDGROUP_TYPE_DEFAULT, api.LB_BACKENDGROUP_TYPE_NORMAL}) {
		return nil, httperrors.NewUnsupportOperationError("backendgroup %s not support this operation", backendGroup.Name)
	}

	guest := backend.(*models.SGuest)
	host := guest.GetHost()
	if host == nil {
		return nil, fmt.Errorf("error getting host of guest %s", guest.GetId())
	}
	if lb == nil {
		return nil, fmt.Errorf("error loadbalancer of backend group %s", backendGroup.GetId())
	}

	hostRegion := host.GetRegion()
	lbRegion := lb.GetRegion()
	if hostRegion.Id != lbRegion.Id {
		return nil, httperrors.NewInputParameterError("region of host %q (%s) != region of loadbalancer %q (%s))",
			host.Name, host.ZoneId, lb.Name, lb.ZoneId)
	}

	address, err := models.LoadbalancerBackendManager.GetGuestAddress(guest)
	if err != nil {
		return nil, err
	}

	data.Set("address", jsonutils.NewString(address))
	data.Set("name", jsonutils.NewString(name))
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lbbg *models.SLoadbalancerBackendGroup) (*jsonutils.JSONDict, error) {
	keyV := map[string]validators.IValidator{
		"weight":     validators.NewRangeValidator("weight", 1, 100).Optional(true),
		"port":       validators.NewPortValidator("port").Optional(true),
		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Optional(true),
	}

	if err := RunValidators(keyV, data, true); err != nil {
		return nil, err
	}

	switch lbbg.Type {
	case api.LB_BACKENDGROUP_TYPE_DEFAULT:
		if data.Contains("port") {
			return nil, httperrors.NewInputParameterError("%s backend group not support change port", lbbg.Type)
		}
	case api.LB_BACKENDGROUP_TYPE_NORMAL:
		return data, nil
	case api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
		if data.Contains("port") || data.Contains("weight") {
			return data, httperrors.NewInputParameterError("%s backend group not support change port or weight", lbbg.Type)
		}
	default:
		return nil, httperrors.NewInputParameterError("Unknown backend group type %s", lbbg.Type)
	}

	return data, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
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

	backendgroup, ok := backendGroup.(*models.SLoadbalancerBackendGroup)
	if !ok {
		return nil, httperrors.NewMissingParameterError("backend_group")
	}
	if backendgroup.Type != api.LB_BACKENDGROUP_TYPE_NORMAL {
		return nil, httperrors.NewInputParameterError("backend group type must be normal")
	}

	data.Set("cloudregion_id", jsonutils.NewString(listener.CloudregionId))
	data.Set("manager_id", jsonutils.NewString(listener.ManagerId))
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	lbr := ctx.Value("lbr").(*models.SLoadbalancerListenerRule)
	keyV := map[string]validators.IValidator{
		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate"),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src"),
	}
	for _, v := range keyV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
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

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
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

		"client_request_timeout":  validators.NewRangeValidator("client_request_timeout", 0, 600).Default(10),
		"client_idle_timeout":     validators.NewRangeValidator("client_idle_timeout", 0, 600).Default(90),
		"backend_connect_timeout": validators.NewRangeValidator("backend_connect_timeout", 0, 180).Default(5),
		"backend_idle_timeout":    validators.NewRangeValidator("backend_idle_timeout", 0, 600).Default(90),

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES).Default(api.LB_BOOL_OFF),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", api.LB_STICKY_SESSION_TYPES).Default(api.LB_STICKY_SESSION_TYPE_INSERT),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)).Optional(true),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout").Optional(true),

		"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for").Default(true),
		"gzip":            validators.NewBoolValidator("gzip").Default(false),

		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate").Default(0),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src").Default(0),
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

	//  检查带宽限制
	maxEgressMbps := 5000
	if lb.ChargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		maxEgressMbps = lb.EgressMbps
	}

	listeners, err := lb.GetLoadbalancerListeners()
	if err != nil {
		return nil, err
	}
	for _, listener := range listeners {
		if listener.EgressMbps > 0 {
			maxEgressMbps -= listener.EgressMbps
		}
	}

	egressMbpsV := validators.NewRangeValidator("egress_mbps", 0, int64(maxEgressMbps)).Optional(true)
	if err := egressMbpsV.Validate(data); err != nil {
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

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10).Default(3),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 2, 10).Default(3),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 300).Default(5),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Default(2),
	}

	if err := RunValidators(keyVHealth, data, false); err != nil {
		return nil, err
	}

	// acl check
	if err := models.LoadbalancerListenerManager.ValidateAcl(aclStatusV, aclTypeV, aclV, data, api.CLOUD_PROVIDER_ALIYUN); err != nil {
		return nil, err
	}

	backendgroup, ok := backendGroup.(*models.SLoadbalancerBackendGroup)
	if !ok {
		return nil, httperrors.NewMissingParameterError("backend_group")
	}

	// http&https listenerType limitation check
	if utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) && !utils.IsInStringArray(backendgroup.Type, []string{api.LB_BACKENDGROUP_TYPE_DEFAULT, api.LB_BACKENDGROUP_TYPE_NORMAL}) {
		return nil, httperrors.NewUnsupportOperationError("http or https listener only supportd default or normal backendgroup")
	}

	if tlsCipherPolicy, _ := data.GetString("tls_cipher_policy"); len(tlsCipherPolicy) > 0 && len(lb.LoadbalancerSpec) == 0 {
		data.Set("tls_cipher_policy", jsonutils.NewString(""))
	}

	if healthCheckDomain, _ := data.GetString("health_check_domain"); len(healthCheckDomain) > 80 {
		return nil, httperrors.NewInputParameterError("health_check_domain must be in the range of 1 ~ 80")
	}

	// 阿里云协议限制
	V := map[string]validators.IValidator{}
	V["scheduler"] = validators.NewStringChoicesValidator("scheduler", api.LB_ALIYUN_COMMON_SCHEDULER_TYPES)
	switch listenerType {
	case api.LB_LISTENER_TYPE_UDP:
		V["health_check_interval"] = validators.NewRangeValidator("health_check_interval", 1, 50).Default(5)
		V["scheduler"] = validators.NewStringChoicesValidator("scheduler", api.LB_ALIYUN_UDP_SCHEDULER_TYPES)
		for _, _key := range []string{"health_check_req", "health_check_exp"} {
			if key, _ := data.GetString(_key); len(key) > 500 {
				return nil, httperrors.NewInputParameterError("%s length must less 500 letters", key)
			}
		}
	case api.LB_LISTENER_TYPE_HTTP:
		V["client_idle_timeout"] = validators.NewRangeValidator("client_idle_timeout", 1, 60).Default(15)
		V["client_request_timeout"] = validators.NewRangeValidator("client_request_timeout", 1, 180).Default(60)
		if strickySession, _ := data.GetString("sticky_session"); strickySession == api.LB_BOOL_ON {
			strickySessionType, _ := data.GetString("sticky_session_type")
			switch strickySessionType {
			case api.LB_STICKY_SESSION_TYPE_INSERT:
				V["sticky_session_cookie_timeout"] = validators.NewRangeValidator("sticky_session_cookie_timeout", 1, 86400).Default(1000)
			case api.LB_STICKY_SESSION_TYPE_SERVER:
				cookie, _ := data.GetString("sticky_session_cookie")
				if len(cookie) < 1 || len(cookie) > 200 {
					return nil, httperrors.NewInputParameterError("sticky_session_cookie length must within 1~200")
				}
				//只能包含字母、数字、‘_’和‘-’
				regexpCookie := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
				if !regexpCookie.MatchString(cookie) {
					return nil, httperrors.NewInputParameterError("sticky_session_cookie can only contain letters, Numbers, '_' and '-'")
				}
			default:
				return nil, httperrors.NewInputParameterError("Unknown sticky_session_type, only support %s or %s", api.LB_STICKY_SESSION_TYPE_INSERT, api.LB_STICKY_SESSION_TYPE_SERVER)
			}
		}
	case api.LB_LISTENER_TYPE_HTTPS:
		V["client_idle_timeout"] = validators.NewRangeValidator("client_idle_timeout", 1, 60).Default(15)
		V["client_request_timeout"] = validators.NewRangeValidator("client_request_timeout", 1, 180).Default(60)
	}

	if backendgroup.Type == api.LB_BACKENDGROUP_TYPE_DEFAULT {
		V["backend_server_port"] = validators.NewPortValidator("backend_server_port")
	}

	if err := RunValidators(V, data, false); err != nil {
		return nil, err
	}

	// check scheduler limiations
	cloudregion := lb.GetRegion()
	if cloudregion == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find loadbalancer's %s(%s) region", lb.Name, lb.Id)
	}

	if scheduler, _ := data.GetString("scheduler"); utils.IsInStringArray(scheduler, []string{api.LB_SCHEDULER_SCH, api.LB_SCHEDULER_TCH, api.LB_SCHEDULER_QCH}) {
		if len(lb.LoadbalancerSpec) == 0 {
			return nil, httperrors.NewInputParameterError("The specified Scheduler %s is invalid for performance sharing loadbalancer", scheduler)
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
		if !utils.IsInStringArray(cloudregion.ExternalId, supportRegions) {
			return nil, httperrors.NewUnsupportOperationError("cloudregion %s(%d) not support %s scheduler", cloudregion.Name, cloudregion.Id, scheduler)
		}
	}

	data.Set("cloudregion_id", jsonutils.NewString(cloudregion.GetId()))
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerListenerData(ctx, userCred, ownerId, data, lb, backendGroup)
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lblis *models.SLoadbalancerListener, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	ownerId := lblis.GetOwnerId()
	aclStatusV := validators.NewStringChoicesValidator("acl_status", api.LB_BOOL_VALUES)
	defaultAclStatus := lblis.AclStatus
	if defaultAclStatus == "" {
		defaultAclStatus = api.LB_BOOL_OFF
	}

	aclStatusV.Default(defaultAclStatus)
	certV := validators.NewModelIdOrNameValidator("certificate", "loadbalancercertificate", ownerId)
	tlsCipherPolicyV := validators.NewStringChoicesValidator("tls_cipher_policy", api.LB_TLS_CIPHER_POLICIES).Default(api.LB_TLS_CIPHER_POLICY_1_2)
	keyV := map[string]validators.IValidator{
		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES),

		"acl_status": aclStatusV,

		"client_idle_timeout":     validators.NewRangeValidator("client_idle_timeout", 0, 600),
		"backend_connect_timeout": validators.NewRangeValidator("backend_connect_timeout", 0, 180),
		"backend_idle_timeout":    validators.NewRangeValidator("backend_idle_timeout", 0, 600),

		"sticky_session":        validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES),
		"sticky_session_type":   validators.NewStringChoicesValidator("sticky_session_type", api.LB_STICKY_SESSION_TYPES),
		"sticky_session_cookie": validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)),

		"health_check":      validators.NewStringChoicesValidator("health_check", api.LB_BOOL_VALUES),
		"health_check_type": models.LoadbalancerListenerManager.CheckTypeV(lblis.ListenerType),

		"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true),
		"health_check_path":      validators.NewURLPathValidator("health_check_path"),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(","),

		"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for"),
		"gzip":            validators.NewBoolValidator("gzip"),

		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate"),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src"),

		"certificate":       certV,
		"tls_cipher_policy": tlsCipherPolicyV,
		"enable_http2":      validators.NewBoolValidator("enable_http2"),
	}

	if err := RunValidators(keyV, data, true); err != nil {
		return nil, err
	}

	{
		aclTypeV := validators.NewStringChoicesValidator("acl_type", api.LB_ACL_TYPES)
		if api.LB_ACL_TYPES.Has(lblis.AclType) {
			aclTypeV.Default(lblis.AclType)
		}

		aclV := validators.NewModelIdOrNameValidator("acl", "loadbalanceracl", ownerId)
		if len(lblis.AclId) > 0 {
			aclV.Default(lblis.AclId)
		}

		if acl_status, _ := data.GetString("acl_status"); acl_status == api.LB_BOOL_ON {
			aclKeyV := map[string]validators.IValidator{
				"acl_type": aclTypeV,
				"acl":      aclV,
			}

			if err := RunValidators(aclKeyV, data, false); err != nil {
				return nil, err
			}
		}

		if err := models.LoadbalancerListenerManager.ValidateAcl(aclStatusV, aclTypeV, aclV, data, lblis.GetProviderName()); err != nil {
			return nil, err
		}
	}

	{
		if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != lblis.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, lblis.LoadbalancerId)
		}
	}

	listenerType, _ := data.GetString("listener_type")

	lb := lblis.GetLoadbalancer()
	if lb == nil {
		return nil, httperrors.NewInternalServerError("failed to found loadbalancer for listener %s(%s)", lblis.Name, lblis.Id)
	}

	egressMbps := 5000
	if lb.ChargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		egressMbps = lb.EgressMbps
	}

	listeners, err := lb.GetLoadbalancerListeners()
	if err != nil {
		return nil, err
	}
	for _, listener := range listeners {
		if listener.EgressMbps > 0 && listener.Id != lblis.Id {
			egressMbps -= listener.EgressMbps
		}
	}

	V := map[string]validators.IValidator{
		"egress_mbps": validators.NewRangeValidator("egress_mbps", 0, int64(egressMbps)).Optional(true),

		"client_request_timeout": validators.NewRangeValidator("client_request_timeout", 1, 180).Optional(true),

		"sticky_session_cookie_timeout": validators.NewRangeValidator("sticky_session_cookie_timeout", 1, 86400).Optional(true),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10).Optional(true),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 2, 10).Optional(true),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 300).Optional(true),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Optional(true),
		"scheduler":             validators.NewStringChoicesValidator("scheduler", api.LB_ALIYUN_COMMON_SCHEDULER_TYPES).Optional(true),
	}
	if lblis.ListenerType == api.LB_LISTENER_TYPE_UDP {
		V["scheduler"] = validators.NewStringChoicesValidator("scheduler", api.LB_ALIYUN_UDP_SCHEDULER_TYPES).Optional(true)
	}

	if scheduler, _ := data.GetString("scheduler"); utils.IsInStringArray(scheduler, []string{api.LB_SCHEDULER_SCH, api.LB_SCHEDULER_TCH, api.LB_SCHEDULER_QCH}) {
		if len(lb.LoadbalancerSpec) == 0 {
			return nil, httperrors.NewInputParameterError("The specified Scheduler %s is invalid for performance sharing loadbalancer", scheduler)
		}
		cloudregion := lb.GetRegion()
		if cloudregion == nil {
			return nil, httperrors.NewResourceNotFoundError("failed to find loadbalancer's %s(%s) region", lb.Name, lb.Id)
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
		if !utils.IsInStringArray(cloudregion.ExternalId, supportRegions) {
			return nil, httperrors.NewUnsupportOperationError("cloudregion %s(%d) not support %s scheduler", cloudregion.Name, cloudregion.Id, scheduler)
		}
	}

	if healthCheck, _ := data.GetString("health_check"); len(healthCheck) > 0 {
		switch healthCheck {
		case api.LB_BOOL_ON:
			for key, lisValue := range map[string]int{"health_check_rise": lblis.HealthCheckRise, "health_check_fall": lblis.HealthCheckFall, "health_check_timeout": lblis.HealthCheckTimeout, "health_check_interval": lblis.HealthCheckInterval} {
				if value, err := data.Int(key); data.Contains(key) && err != nil {
					return nil, httperrors.NewInputParameterError("invalid %s,required int", key)
				} else if err == nil && value == 0 && lisValue == 0 {
					return nil, httperrors.NewInputParameterError("%s cannot be set to 0", key)
				}
			}
		case api.LB_BOOL_OFF:
			if utils.IsInStringArray(lblis.ListenerType, []string{api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP}) {
				return nil, httperrors.NewUnsupportOperationError("%s not support close tcp or udp loadbalancer listener health check", self.GetProvider())
			}
		}
	}

	if healthCheckDomain, _ := data.GetString("health_check_domain"); len(healthCheckDomain) > 80 {
		return nil, httperrors.NewInputParameterError("health_check_domain must be in the range of 1 ~ 80")
	}

	for _, _key := range []string{"health_check_req", "health_check_exp"} {
		if key, _ := data.GetString(_key); len(key) > 500 {
			return nil, httperrors.NewInputParameterError("%s length must less 500 letters", key)
		}
	}

	backendgroup, ok := backendGroup.(*models.SLoadbalancerBackendGroup)
	if ok {
		if utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) && !utils.IsInStringArray(backendgroup.Type, []string{api.LB_BACKENDGROUP_TYPE_DEFAULT, api.LB_BACKENDGROUP_TYPE_NORMAL}) {
			return nil, httperrors.NewUnsupportOperationError("http or https listener only supportd default or normal backendgroup")
		}

		if backendgroup.Type == api.LB_BACKENDGROUP_TYPE_DEFAULT {
			if lblis.BackendServerPort == 0 {
				V["backend_server_port"] = validators.NewPortValidator("backend_server_port")
			}
		}

		lb := backendgroup.GetLoadbalancer()
		if tlsCipherPolicy, _ := data.GetString("tls_cipher_policy"); len(tlsCipherPolicy) > 0 && len(lb.LoadbalancerSpec) == 0 {
			data.Set("tls_cipher_policy", jsonutils.NewString(""))
		}
	}

	if !utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_UDP, api.LB_LISTENER_TYPE_TCP}) {
		if lblis.ClientIdleTimeout == 0 {
			V["client_idle_timeout"] = validators.NewRangeValidator("client_idle_timeout", 1, 60)
		}
	}

	if err := RunValidators(V, data, true); err != nil {
		return nil, err
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateLoadbalancerListenerData(ctx, userCred, data, lblis, backendGroup)
}

func (self *SAliyunRegionDriver) ValidateCreateSnapshopolicyDiskData(ctx context.Context,
	userCred mcclient.TokenCredential, disk *models.SDisk, snapshotPolicy *models.SSnapshotPolicy) error {
	//err := self.SManagedVirtualizationRegionDriver.ValidateCreateSnapshopolicyDiskData(ctx, userCred, disk, snapshotPolicy)
	//if err != nil {
	//	return nil
	//}
	//// In Aliyun, One disk only apply one snapshot policy
	//ret, err := models.SnapshotPolicyDiskManager.FetchAllByDiskID(ctx, userCred, disk.GetId())
	//if err != nil {
	//	return err
	//}
	//if len(ret) != 0 {
	//	return httperrors.NewBadRequestError("One disk could't attach two snapshot policy in aliyun; please detach last one first.")
	//}
	//return nil
	return nil
}

func (self *SAliyunRegionDriver) ValidateSnapshotCreate(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, data *jsonutils.JSONDict) error {
	name, _ := data.GetString("name")
	if strings.HasPrefix(name, "auto") || strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		return httperrors.NewBadRequestError(
			"Snapshot for %s name can't start with auto, http:// or https://", self.GetProvider())
	}
	return nil
}

func (self *SAliyunRegionDriver) DealNatGatewaySpec(spec string) string {
	switch spec {
	case "Small":
		return api.NAT_SPEC_SMALL
	case "Midele":
		return api.NAT_SPEC_MIDDLE
	case "Large":
		return api.NAT_SPEC_LARGE
	case "XLarge.1":
		return api.NAT_SPEC_XLARGE
	}
	//can't arrive
	return ""
}

// RequestBindIPToNatgateway in aliyun don't need to check eip again which is different from SManagerResongDriver.
// RequestBindIPToNatgateway because func ieip.Associate will fail if eip has been associate
func (self *SAliyunRegionDriver) RequestBindIPToNatgateway(ctx context.Context, task taskman.ITask, natgateway *models.SNatGateway,
	eipId string) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		model, err := models.ElasticipManager.FetchById(eipId)
		if err != nil {
			return nil, err
		}
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)
		eip := model.(*models.SElasticip)
		iregion, err := natgateway.GetIRegion()
		if err != nil {
			return nil, err
		}
		ieip, err := iregion.GetIEipById(eip.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "fetch eip failed")
		}
		err = ieip.Associate(natgateway.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "fail to bind eip to natgateway")
		}

		err = cloudprovider.WaitStatus(ieip, api.EIP_STATUS_READY, 5*time.Second, 100*time.Second)
		if err != nil {
			return nil, err
		}

		// database
		_, err = db.Update(eip, func() error {
			eip.AssociateType = api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
			eip.AssociateId = natgateway.GetId()
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "fail to update eip '%s' in database", eip.Id)
		}
		return nil, nil
	})
	return nil
}

func (self *SAliyunRegionDriver) RequestUnBindIPFromNatgateway(ctx context.Context, task taskman.ITask,
	nat models.INatHelper, natgateway *models.SNatGateway) error {

	count, err := nat.CountByEIP()
	if err != nil {
		return errors.Wrapf(err, "fail to count by eip")
	}
	if count > 0 {
		return nil
	}
	eip := &models.SElasticip{}
	err = models.ElasticipManager.Query().Equals("associate_id", natgateway.Id).First(eip)
	if err != nil {
		return errors.Wrapf(err, "fail to fetch eip associate with natgateway %s", natgateway.Id)
	}
	eip.SetModelManager(models.ElasticipManager, eip)
	lockman.LockObject(ctx, eip)
	defer lockman.ReleaseObject(ctx, eip)
	iregion, err := eip.GetIRegion()
	if err != nil {
		return errors.Wrapf(err, "fail to fetch iregion")
	}
	ieip, err := iregion.GetIEipById(eip.GetExternalId())
	if err != nil {
		return errors.Wrap(err, "fetch eip failed")
	}
	err = ieip.Dissociate()
	if err != nil {
		return errors.Wrap(err, "fail to unbind eip from natgateway")
	}

	err = cloudprovider.WaitStatus(ieip, api.EIP_STATUS_READY, 5*time.Second, 100*time.Second)
	if err != nil {
		return err
	}

	// database
	_, err = db.Update(eip, func() error {
		eip.AssociateType = ""
		eip.AssociateId = ""
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "fail to update eip '%s' in database", eip.Id)
	}
	return nil
}

func (self *SAliyunRegionDriver) BindIPToNatgatewayRollback(ctx context.Context, eipId string) error {
	return nil
}

func (self *SAliyunRegionDriver) IsSecurityGroupBelongVpc() bool {
	return true
}

func (self *SAliyunRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.SDBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (*api.SDBInstanceCreateInput, error) {
	if input.BillingType == billing_api.BILLING_TYPE_PREPAID && len(input.MasterInstanceId) > 0 {
		return nil, httperrors.NewInputParameterError("slave dbinstance not support prepaid billing type")
	}

	wire := network.GetWire()
	if wire == nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("failed to found wire for network %s(%s)", network.Name, network.Id))
	}
	zone := wire.GetZone()
	if zone == nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("failed to found zone for wire %s(%s)", wire.Name, wire.Id))
	}

	match := false
	for _, sku := range skus {
		if utils.IsInStringArray(zone.Id, []string{sku.Zone1, sku.Zone2, sku.Zone3}) {
			match = true
			break
		}
	}

	if !match {
		return nil, httperrors.NewInputParameterError("failed to match any skus in the network %s(%s) zone %s(%s)", network.Name, network.Id, zone.Name, zone.Id)
	}

	var master *models.SDBInstance
	var slaves []models.SDBInstance
	var err error
	if len(input.MasterInstanceId) > 0 {
		_master, _ := models.DBInstanceManager.FetchById(input.MasterInstanceId)
		master = _master.(*models.SDBInstance)
		slaves, err = master.GetSlaveDBInstances()
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}

		switch master.Engine {
		case api.DBINSTANCE_TYPE_MYSQL:
			switch master.EngineVersion {
			case "5.6":
				break
			case "5.7", "8.0":
				if master.Category != api.ALIYUN_DBINSTANCE_CATEGORY_HA {
					return nil, httperrors.NewInputParameterError("Not support create readonly dbinstance for MySQL %s %s", master.EngineVersion, master.Category)
				}
				if master.StorageType != api.ALIYUN_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD {
					return nil, httperrors.NewInputParameterError("Not support create readonly dbinstance for MySQL %s %s with storage type %s, only support %s", master.EngineVersion, master.Category, master.StorageType, api.ALIYUN_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD)
				}
			default:
				return nil, httperrors.NewInputParameterError("Not support create readonly dbinstance for MySQL %s", master.EngineVersion)
			}
		case api.DBINSTANCE_TYPE_SQLSERVER:
			if master.Category != api.ALIYUN_DBINSTANCE_CATEGORY_ALWAYSON || master.EngineVersion != "2017_ent" {
				return nil, httperrors.NewInputParameterError("SQL Server only support create readonly dbinstance for 2017_ent")
			}
			if len(slaves) >= 7 {
				return nil, httperrors.NewInputParameterError("SQL Server cannot have more than seven read-only dbinstances")
			}
		default:
			return nil, httperrors.NewInputParameterError("Not support create readonly dbinstance which master dbinstance engine is", master.Engine)
		}
	}

	switch input.Engine {
	case api.DBINSTANCE_TYPE_MYSQL:
		if input.VmemSizeMb/1024 >= 64 && len(slaves) >= 10 {
			return nil, httperrors.NewInputParameterError("Master dbinstance memory ≥64GB, up to 10 read-only instances are allowed to be created")
		} else if input.VmemSizeMb/1024 < 64 && len(slaves) >= 5 {
			return nil, httperrors.NewInputParameterError("Master dbinstance memory <64GB, up to 5 read-only instances are allowed to be created")
		}
	case api.DBINSTANCE_TYPE_SQLSERVER:
		if input.Category == api.ALIYUN_DBINSTANCE_CATEGORY_ALWAYSON {
			vpc := network.GetVpc()
			count, err := vpc.GetNetworkCount()
			if err != nil {
				return nil, httperrors.NewGeneralError(err)
			}
			if count < 2 {
				return nil, httperrors.NewInputParameterError("At least two networks are required under vpc %s(%s) whith aliyun %s(%s)", vpc.Name, vpc.Id, input.Engine, input.Category)
			}
		}
	}

	if len(input.Name) > 0 {
		if strings.HasPrefix(input.Description, "http://") || strings.HasPrefix(input.Description, "https://") {
			return nil, httperrors.NewInputParameterError("Description can not start with http:// or https://")
		}
	}

	return input, nil
}

func (self *SAliyunRegionDriver) IsSupportedBillingCycle(bc billing.SBillingCycle, resource string) bool {
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

func (self *SAliyunRegionDriver) RequestCreateDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRds, err := instance.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrap(err, "instance.GetIDBInstance")
		}

		desc := &cloudprovider.SDBInstanceBackupCreateConfig{
			Name: backup.Name,
		}
		if len(backup.DBNames) > 0 {
			desc.Databases = strings.Split(backup.DBNames, ",")
		}

		_, err = iRds.CreateIBackup(desc)
		if err != nil {
			return nil, errors.Wrap(err, "iRds.CreateBackup")
		}

		backups, err := iRds.GetIDBInstanceBackups()
		if err != nil {
			return nil, errors.Wrap(err, "iRds.GetIDBInstanceBackups")
		}
		result := models.DBInstanceBackupManager.SyncDBInstanceBackups(ctx, userCred, backup.GetCloudprovider(), instance, backup.GetRegion(), backups)
		log.Infof("SyncDBInstanceBackups for dbinstance %s(%s) result: %s", instance.Name, instance.Id, result.Result())
		return nil, nil
	})
	return nil
}

func (self *SAliyunRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input *api.SDBInstanceAccountCreateInput) (*api.SDBInstanceAccountCreateInput, error) {
	if len(input.Name) < 2 || len(input.Name) > 16 {
		return nil, httperrors.NewInputParameterError("Aliyun DBInstance account name length shoud be 2~16 characters")
	}

	DENY_KEY := map[string][]string{
		api.DBINSTANCE_TYPE_MYSQL:     api.ALIYUN_MYSQL_DENY_KEYWORKD,
		api.DBINSTANCE_TYPE_SQLSERVER: api.ALIYUN_SQL_SERVER_DENY_KEYWORD,
	}

	if keys, ok := DENY_KEY[instance.Engine]; ok && utils.IsInStringArray(input.Name, keys) {
		return nil, httperrors.NewInputParameterError("%s is reserved for aliyun %s, please use another", input.Name, instance.Engine)
	}

	for i, s := range input.Name {
		if !unicode.IsLetter(s) && !unicode.IsDigit(s) && s != '_' {
			return nil, httperrors.NewInputParameterError("invalid character %s for account name", s)
		}
		if s == '_' && (i == 0 || i == len(input.Name)) {
			return nil, httperrors.NewInputParameterError("account name can not start or end with _")
		}
	}

	for _, privilege := range input.Privileges {
		err := self.ValidateDBInstanceAccountPrivilege(ctx, userCred, instance, privilege.Privilege)
		if err != nil {
			return nil, err
		}
	}

	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input *api.SDBInstanceDatabaseCreateInput) (*api.SDBInstanceDatabaseCreateInput, error) {
	if len(input.CharacterSet) == 0 {
		return nil, httperrors.NewMissingParameterError("character_set")
	}

	for _, account := range input.Accounts {
		err := self.ValidateDBInstanceAccountPrivilege(ctx, userCred, instance, account.Privilege)
		if err != nil {
			return nil, err
		}
	}

	return input, nil
}

func (self *SAliyunRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input *api.SDBInstanceBackupCreateInput) (*api.SDBInstanceBackupCreateInput, error) {
	return input, nil
}

func (self *SAliyunRegionDriver) ValidateDBInstanceAccountPrivilege(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, privilege string) error {
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

func (self *SAliyunRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	networkV := validators.NewModelIdOrNameValidator("network", "network", ownerId)

	// billing cycle
	billingTypeV := validators.NewStringChoicesValidator("billing_type", choices.NewChoices(billing_api.BILLING_TYPE_PREPAID, billing_api.BILLING_TYPE_POSTPAID))
	networkTypeV := validators.NewStringChoicesValidator("network_type", choices.NewChoices(api.LB_NETWORK_TYPE_VPC, api.LB_NETWORK_TYPE_CLASSIC)).Default(api.LB_NETWORK_TYPE_VPC).Optional(true)
	engineV := validators.NewStringChoicesValidator("engine", choices.NewChoices("redis", "memcache"))
	engineVersionV := validators.NewStringChoicesValidator("engine_version", choices.NewChoices("2.8", "4.0", "5.0"))
	privateIpV := validators.NewIPv4AddrValidator("private_ip").Optional(true)

	keyV := map[string]validators.IValidator{
		"zone":           zoneV,
		"billing_type":   billingTypeV,
		"network_type":   networkTypeV,
		"network":        networkV,
		"engine":         engineV,
		"engine_version": engineVersionV,
		"private_ip":     privateIpV,
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
	billingType, _ := data.GetString("billing_type")
	zone := zoneV.Model.(*models.SZone)
	if sku, err := data.GetString("instance_type"); err != nil || len(sku) == 0 {
		return nil, httperrors.NewMissingParameterError("instance_type")
	} else {
		_skuModel, err := db.FetchByIdOrName(models.ElasticcacheSkuManager, userCred, sku)
		if err != nil {
			return nil, err
		}

		skuModel := _skuModel.(*models.SElasticcacheSku)
		if err := ValidateElasticcacheSku(zone.Id, billingType, skuModel); err != nil {
			return nil, err
		} else {
			data.Set("instance_type", jsonutils.NewString(skuModel.InstanceSpec))
			data.Set("node_type", jsonutils.NewString(skuModel.NodeType))
			data.Set("local_category", jsonutils.NewString(skuModel.LocalCategory))
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
	data.Set("cloudregion_id", jsonutils.NewString(zone.GetCloudRegionId()))
	return data, nil
}

func (self *SAliyunRegionDriver) RequestCreateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := ec.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.GetIRegion")
		}

		iprovider, err := db.FetchById(models.CloudproviderManager, ec.ManagerId)
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.GetProvider")
		}

		provider := iprovider.(*models.SCloudprovider)

		params, err := ec.GetCreateAliyunElasticcacheParams()
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.GetCreateAliyunElasticcacheParams")
		}

		iec, err := iRegion.CreateIElasticcaches(params)
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.CreateIElasticcaches")
		}

		err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 30*time.Second, 15*time.Second, 600*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.WaitStatusWithDelay")
		}

		ec.SetModelManager(models.ElasticcacheManager, ec)
		if err := db.SetExternalId(ec, userCred, iec.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.SetExternalId")
		}

		{
			// todo: 开启外网访问
		}

		if err := ec.SyncWithCloudElasticcache(ctx, userCred, provider, iec); err != nil {
			return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.SyncWithCloudElasticcache")
		}

		// sync accounts
		{
			iaccounts, err := iec.GetICloudElasticcacheAccounts()
			if err != nil {
				return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.GetICloudElasticcacheAccounts")
			}

			result := models.ElasticcacheAccountManager.SyncElasticcacheAccounts(ctx, userCred, ec, iaccounts)
			log.Debugf("aliyunRegionDriver.CreateElasticcache.SyncElasticcacheAccounts %s", result.Result())

			account, err := ec.GetAdminAccount()
			if err != nil {
				return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.GetAdminAccount")
			}

			err = account.SavePassword(params.Password)
			if err != nil {
				return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.SavePassword")
			}
		}

		// sync acl
		{
			iacls, err := iec.GetICloudElasticcacheAcls()
			if err != nil {
				return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.GetICloudElasticcacheAcls")
			}

			result := models.ElasticcacheAclManager.SyncElasticcacheAcls(ctx, userCred, ec, iacls)
			log.Debugf("aliyunRegionDriver.CreateElasticcache.SyncElasticcacheAcls %s", result.Result())
		}

		// sync parameters
		{
			iparams, err := iec.GetICloudElasticcacheParameters()
			if err != nil {
				return nil, errors.Wrap(err, "aliyunRegionDriver.CreateElasticcache.GetICloudElasticcacheParameters")
			}

			result := models.ElasticcacheParameterManager.SyncElasticcacheParameters(ctx, userCred, ec, iparams)
			log.Debugf("aliyunRegionDriver.CreateElasticcache.SyncElasticcacheParameters %s", result.Result())
		}

		return nil, nil
	})
	return nil
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
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}

	passwd, _ := data.GetString("password")
	if !seclib2.MeetComplxity(passwd) {
		return nil, httperrors.NewWeakPasswordError()
	}

	if accountPrivilegeV.Value == "repl" && elasticCacheV.Model.(*models.SElasticcache).EngineVersion != "4.0" {
		return nil, httperrors.NewInputParameterError("account_privilege %s only support redis version 4.0")
	}

	return data, nil
}

func (self *SAliyunRegionDriver) RequestCreateElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
		if err != nil {
			return nil, errors.Wrap(nil, "aliyunRegionDriver.CreateElasticcacheAccount.GetElasticcache")
		}

		ec := _ec.(*models.SElasticcache)
		iregion, err := ec.GetIRegion()
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

		if err := ea.SyncWithCloudElasticcacheAccount(ctx, userCred, iea); err != nil {
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
		iregion, err := ec.GetIRegion()
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

		_, err = iec.CreateBackup()
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
	iregion, err := ea.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "aliyunRegionDriver.RequestElasticcacheAccountResetPassword.FetchById")
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err == cloudprovider.ErrNotFound {
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
