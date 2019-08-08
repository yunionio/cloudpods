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
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
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
	data.Set("vpc_id", jsonutils.NewString(""))
	data.Set("address", jsonutils.NewString(""))
	data.Set("network_id", jsonutils.NewString(""))
	loadbalancerSpecV := validators.NewStringChoicesValidator("loadbalancer_spec", api.LB_ALIYUN_SPECS)
	loadbalancerSpecV.Default(api.LB_ALIYUN_SPEC_SHAREABLE)
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

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lbbg *models.SLoadbalancerBackendGroup) (*jsonutils.JSONDict, error) {
	switch lbbg.Type {
	case api.LB_BACKENDGROUP_TYPE_DEFAULT:
		if data.Contains("port") {
			return nil, httperrors.NewInputParameterError("%s backend group not support change port", lbbg.Type)
		}
	case api.LB_BACKENDGROUP_TYPE_NORMAL:
		weightV := validators.NewRangeValidator("weight", 1, 100).Optional(true)
		return data, weightV.Validate(data)
	case api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
		if data.Contains("port") || data.Contains("weight") {
			return data, httperrors.NewInputParameterError("%s backend group not support change port or weight", lbbg.Type)
		}
	default:
		return nil, httperrors.NewInputParameterError("Unknown backend group type %s", lbbg.Type)
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
	if utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) && !utils.IsInStringArray(backendgroup.Type, []string{api.LB_BACKENDGROUP_TYPE_DEFAULT, api.LB_BACKENDGROUP_TYPE_NORMAL}) {
		return nil, httperrors.NewUnsupportOperationError("http or https listener only supportd default or normal backendgroup")
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
		"egress_mbps": validators.NewRangeValidator("egress_mbps", 0, int64(egressMbps)).Optional(true),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10).Default(3),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 2, 10).Default(3),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 300).Default(5),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 50).Default(2),
		"scheduler":             validators.NewStringChoicesValidator("scheduler", api.LB_ALIYUN_COMMON_SCHEDULER_TYPES),
	}

	strickySession, _ := data.GetString("sticky_session")

	switch listenerType {
	case api.LB_LISTENER_TYPE_UDP:
		keyV["health_check_interval"] = validators.NewRangeValidator("health_check_interval", 1, 50).Default(5)
		keyV["scheduler"] = validators.NewStringChoicesValidator("scheduler", api.LB_ALIYUN_UDP_SCHEDULER_TYPES)
		for _, _key := range []string{"health_check_req", "health_check_exp"} {
			if key, _ := data.GetString(_key); len(key) > 500 {
				return nil, httperrors.NewInputParameterError("%s length must less 500 letters", key)
			}
		}
	case api.LB_LISTENER_TYPE_HTTP:
		if strickySession == api.LB_BOOL_ON {
			strickySessionType, _ := data.GetString("sticky_session_type")
			switch strickySessionType {
			case api.LB_STICKY_SESSION_TYPE_INSERT:
				keyV["sticky_session_cookie_timeout"] = validators.NewRangeValidator("sticky_session_cookie_timeout", 1, 86400).Default(1000)
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
		keyV["client_idle_timeout"] = validators.NewRangeValidator("client_idle_timeout", 1, 60).Default(15)
		keyV["client_request_timeout"] = validators.NewRangeValidator("client_request_timeout", 1, 180).Default(60)
	case api.LB_LISTENER_TYPE_HTTPS:
		keyV["client_idle_timeout"] = validators.NewRangeValidator("client_idle_timeout", 1, 60).Default(15)
		keyV["client_request_timeout"] = validators.NewRangeValidator("client_request_timeout", 1, 180).Default(60)
	}

	if backendgroup.Type == api.LB_BACKENDGROUP_TYPE_DEFAULT {
		keyV["backend_server_port"] = validators.NewPortValidator("backend_server_port")
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

	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerListenerData(ctx, userCred, data, backendGroup)
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lblis *models.SLoadbalancerListener, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
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

	keyV := map[string]validators.IValidator{
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
		keyV["scheduler"] = validators.NewStringChoicesValidator("scheduler", api.LB_ALIYUN_UDP_SCHEDULER_TYPES).Optional(true)
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
				if value, _ := data.Int(key); value == 0 && lisValue == 0 {
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
				keyV["backend_server_port"] = validators.NewPortValidator("backend_server_port")
			}
		}

		lb := backendgroup.GetLoadbalancer()
		if tlsCipherPolicy, _ := data.GetString("tls_cipher_policy"); len(tlsCipherPolicy) > 0 && len(lb.LoadbalancerSpec) == 0 {
			data.Set("tls_cipher_policy", jsonutils.NewString(""))
		}
	}

	if !utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_UDP, api.LB_LISTENER_TYPE_TCP}) {
		if lblis.ClientIdleTimeout == 0 {
			keyV["client_idle_timeout"] = validators.NewRangeValidator("client_idle_timeout", 1, 60)
		}
	}

	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	return self.SManagedVirtualizationRegionDriver.ValidateUpdateLoadbalancerListenerData(ctx, userCred, data, lblis, backendGroup)
}

func daysValidate(days []int, min, max int) ([]int, error) {
	if len(days) == 0 {
		return days, nil
	}
	sort.Ints(days)

	var tmp *int
	for i := 0; i < len(days); i++ {
		if days[i] < min || days[i] > max {
			return days, fmt.Errorf("Day %d out of range", days[i])
		}
		if tmp != nil && *tmp == days[i] {
			return days, fmt.Errorf("Has repeat day %v", days)
		} else {
			tmp = &days[i]
		}
	}
	return days, nil
}

func (self *SAliyunRegionDriver) ValidateCreateSnapshotPolicyData(ctx context.Context, userCred mcclient.TokenCredential, data *compute.SSnapshotPolicyCreateInput) error {
	var err error
	if data.RetentionDays < -1 || data.RetentionDays == 0 || data.RetentionDays > 65535 {
		return httperrors.NewInputParameterError("Retention days must in 1~65535 or -1")
	}

	if len(data.RepeatWeekdays) == 0 {
		return httperrors.NewMissingParameterError("repeat_weekdays")
	}
	data.RepeatWeekdays, err = daysValidate(data.RepeatWeekdays, 1, 7)
	if err != nil {
		return httperrors.NewInputParameterError(err.Error())
	}

	if len(data.TimePoints) == 0 {
		return httperrors.NewInputParameterError("time_points")
	}
	data.TimePoints, err = daysValidate(data.TimePoints, 0, 23)
	if err != nil {
		return httperrors.NewInputParameterError(err.Error())
	}
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
