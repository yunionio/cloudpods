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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	loadbalancerSpecV := validators.NewStringChoicesValidator("loadbalancer_spec", api.LB_ALIYUN_SPECS)
	loadbalancerSpecV.Default(api.LB_ALIYUN_SPEC_S1_SMALL)
	if err := loadbalancerSpecV.Validate(data); err != nil {
		return nil, err
	}
	chargeType, _ := data.GetString("charge_type")
	if len(chargeType) == 0 {
		chargeType = api.LB_CHARGE_TYPE_BY_TRAFFIC
		data.Set("charge_type", jsonutils.NewString(chargeType))
	}
	if !utils.IsInStringArray(chargeType, []string{api.LB_CHARGE_TYPE_BY_BANDWIDTH, api.LB_CHARGE_TYPE_BY_TRAFFIC}) {
		return nil, httperrors.NewInputParameterError("Unsupport charge type %s, only support traffic or bandwidth")
	}
	if chargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		egressMbps := validators.NewRangeValidator("egress_mbps", 1, 5000)
		if err := egressMbps.Validate(data); err != nil {
			return nil, err
		}
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, data)
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("certificate") || data.Contains("private_key") {
		return nil, httperrors.NewUnsupportOperationError("Aliyun not allow to change certificate")
	}
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateDeleteLoadbalancerBackendCondition(ctx context.Context, lbb *models.SLoadbalancerBackend) error {
	backendGroup := lbb.GetLoadbalancerBackendGroup()
	if backendGroup.Type == api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE {
		return httperrors.NewUnsupportOperationError("backend %s belong master slave backendgroup, not allow delete", lbb.Name)
	}
	return nil
}

func (self *SAliyunRegionDriver) ValidateDeleteLoadbalancerBackendGroupCondition(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup) error {
	if lbbg.Type == api.LB_BACKENDGROUP_TYPE_DEFAULT {
		return httperrors.NewUnsupportOperationError("not allow to delete default backend group")
	}
	return nil
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
	weight, _ := data.Int("weight")
	if weight < 0 || weight > 100 {
		return nil, httperrors.NewInputParameterError("Aliyun instance weight must be in the range of 0 ~ 100")
	}
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	backendgroup, ok := backendGroup.(*models.SLoadbalancerBackendGroup)
	if !ok {
		return nil, httperrors.NewMissingParameterError("backend_group")
	}
	if backendgroup.Type != api.LB_BACKENDGROUP_TYPE_NORMAL {
		return nil, httperrors.NewInputParameterError("backend group type must be normal")
	}
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	backendgroup, ok := backendGroup.(*models.SLoadbalancerBackendGroup)
	if !ok {
		return nil, httperrors.NewMissingParameterError("backend_group")
	}
	listenerType, _ := data.GetString("listener_type")
	if utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) && !utils.IsInStringArray(backendgroup.Type, []string{api.LB_BACKENDGROUP_TYPE_DEFAULT, api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE}) {
		return nil, httperrors.NewUnsupportOperationError("http or https listener only supportd default or master_slave backendgroup")
	}

	lb := backendgroup.GetLoadbalancer()
	if tlsCipherPolicy, _ := data.GetString("tls_cipher_policy"); len(tlsCipherPolicy) > 0 && len(lb.LoadbalancerSpec) == 0 {
		data.Set("tls_cipher_policy", jsonutils.NewString(""))
	}
	if healthCheckDomain, _ := data.GetString("health_check_domain"); len(healthCheckDomain) > 80 {
		return nil, httperrors.NewInputParameterError("health_check_domain must be in the range of 1 ~ 80")
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
		if listener.EgressMbps > 0 {
			egressMbps -= listener.EgressMbps
		}
	}

	keyV := map[string]validators.IValidator{
		"egress_mbps": validators.NewRangeValidator("egress_mbps", -1, int64(egressMbps)).Optional(true),

		"client_request_timeout": validators.NewRangeValidator("client_request_timeout", 1, 180),

		"sticky_session_cookie_timeout": validators.NewRangeValidator("sticky_session_cookie_timeout", 1, 86400),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 2, 10),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 300),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50),
	}
	if !utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_UDP, api.LB_LISTENER_TYPE_TCP}) {
		keyV["client_idle_timeout"] = validators.NewRangeValidator("client_idle_timeout", 1, 60)
	}

	if backendgroup.Type == api.LB_BACKENDGROUP_TYPE_DEFAULT {
		keyV["backend_server_port"] = validators.NewPortValidator("backend_server_port")
	}

	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	listenerType, _ := data.GetString("listener_type")

	keyV := map[string]validators.IValidator{
		"bandwidth": validators.NewRangeValidator("bandwidth", 1, 5000),

		"client_request_timeout": validators.NewRangeValidator("client_request_timeout", 1, 180),

		"sticky_session_cookie_timeout": validators.NewRangeValidator("sticky_session_cookie_timeout", 1, 86400),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 2, 10),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 300),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50),
	}

	if healthCheckDomain, _ := data.GetString("health_check_domain"); len(healthCheckDomain) > 80 {
		return nil, httperrors.NewInputParameterError("health_check_domain must be in the range of 1 ~ 80")
	}

	backendgroup, ok := backendGroup.(*models.SLoadbalancerBackendGroup)
	if ok {
		if utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) && !utils.IsInStringArray(backendgroup.Type, []string{api.LB_BACKENDGROUP_TYPE_DEFAULT, api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE}) {
			return nil, httperrors.NewUnsupportOperationError("http or https listener only supportd default or master_slave backendgroup")
		}

		if backendgroup.Type == api.LB_BACKENDGROUP_TYPE_DEFAULT {
			keyV["backend_server_port"] = validators.NewPortValidator("backend_server_port")
		}

		lb := backendgroup.GetLoadbalancer()
		if tlsCipherPolicy, _ := data.GetString("tls_cipher_policy"); len(tlsCipherPolicy) > 0 && len(lb.LoadbalancerSpec) == 0 {
			data.Set("tls_cipher_policy", jsonutils.NewString(""))
		}
	}

	if !utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_UDP, api.LB_LISTENER_TYPE_TCP}) {
		keyV["client_idle_timeout"] = validators.NewRangeValidator("client_idle_timeout", 1, 60)
	}

	for _, v := range keyV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	return data, nil
}
