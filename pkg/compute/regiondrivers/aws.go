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

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/tasks"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/choices"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
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

func validateAwsLbNetwork(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, requiredMin int) (*jsonutils.JSONDict, error) {
	var networkIds []string
	if ns, err := data.GetString("network"); err != nil {
		return nil, httperrors.NewMissingParameterError("network")
	} else {
		networkIds = strings.Split(ns, ",")
	}

	var regionId string
	var vpcId string
	secondNet := &models.SNetwork{}
	zones := []string{}
	networkV := validators.NewModelIdOrNameValidator("network", "network", ownerId)
	for _, networkId := range networkIds {
		networkObj := jsonutils.NewDict()
		networkObj.Set("network", jsonutils.NewString(networkId))
		if err := networkV.Validate(networkObj); err != nil {
			return nil, err
		}

		network := networkV.Model.(*models.SNetwork)
		region, zone, vpc, _, err := network.ValidateElbNetwork(nil)
		if err != nil {
			return nil, err
		} else {
			//随机选择一个子网
			if requiredMin == 2 && len(networkIds) == 1 {
				wires := models.WireManager.Query().SubQuery()
				q := models.NetworkManager.Query().IsFalse("pending_deleted")
				q = q.Join(wires, sqlchemy.Equals(q.Field("wire_id"), wires.Field("id")))
				q = q.Filter(sqlchemy.Equals(wires.Field("vpc_id"), vpc.GetId()))
				q = q.Filter(sqlchemy.NotEquals(wires.Field("zone_id"), zone.GetId()))
				err := q.First(secondNet)
				if err != nil {
					return nil, httperrors.NewInputParameterError("required at least %d subnet.", requiredMin)
				}
			}
		}

		if vpcId == "" {
			vpcId = vpc.GetId()
			regionId = region.GetId()
			// 检查manager id 和 VPC manager id 是否匹配
			managerId, _ := data.GetString("manager_id")
			if managerId != vpc.ManagerId {
				return nil, httperrors.NewInputParameterError("Loadbalancer's manager %s does not match vpc's(%s(%s)) (%s)", managerId, vpc.GetName(), vpc.GetId(), vpc.ManagerId)
			}
		}

		if vpcId != vpc.GetId() {
			return nil, httperrors.NewInputParameterError("all networks should in the same vpc. (%s).", network.GetId())
		}

		if utils.IsInStringArray(zone.GetId(), zones) {
			return nil, httperrors.NewInputParameterError("already has one network in the zone %s. (%s).", zone.GetName(), network.GetId())
		}
	}

	if len(secondNet.Id) > 0 {
		networkIds = append(networkIds, secondNet.Id)
	}

	if len(networkIds) < requiredMin {
		return nil, httperrors.NewInputParameterError("required at least %d subnet.", requiredMin)
	}

	data.Set("vpc_id", jsonutils.NewString(vpcId))
	data.Set("network_id", jsonutils.NewString(strings.Join(networkIds, ",")))
	data.Set("cloudregion_id", jsonutils.NewString(regionId))
	return data, nil
}

func (self *SAwsRegionDriver) validateCreateLBCommonData(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*validators.ValidatorModelIdOrName, *jsonutils.JSONDict, error) {
	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", ownerId)
	addressTypeV := validators.NewStringChoicesValidator("address_type", api.LB_ADDR_TYPES)
	loadbalancerSpecV := validators.NewStringChoicesValidator("loadbalancer_spec", api.LB_AWS_SPECS)

	keyV := map[string]validators.IValidator{
		"status":            validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),
		"address_type":      addressTypeV.Default(api.LB_ADDR_TYPE_INTRANET),
		"manager":           managerIdV,
		"loadbalancer_spec": loadbalancerSpecV,
	}

	if err := RunValidators(keyV, data); err != nil {
		return nil, nil, err
	}

	data.Set("network_type", jsonutils.NewString(api.LB_NETWORK_TYPE_VPC))
	return managerIdV, data, nil
}

func (self *SAwsRegionDriver) validateCreateApplicationLBData(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	_, data, err := self.validateCreateLBCommonData(ownerId, data)
	if err != nil {
		return nil, err
	}

	if _, err := validateAwsLbNetwork(ownerId, data, 2); err != nil {
		return nil, err
	}
	return data, nil
}

func (self *SAwsRegionDriver) validateCreateNetworkLBData(ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	_, data, err := self.validateCreateLBCommonData(ownerId, data)
	if err != nil {
		return nil, err
	}

	if _, err := validateAwsLbNetwork(ownerId, data, 1); err != nil {
		return nil, err
	}

	return data, nil
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ownerId := ctx.Value("ownerId").(mcclient.IIdentityProvider)
	if spec, _ := data.GetString("loadbalancer_spec"); spec == api.LB_AWS_SPEC_APPLICATION {
		if _, err := self.validateCreateApplicationLBData(ownerId, data); err != nil {
			return nil, err
		}
	} else if spec == api.LB_AWS_SPEC_NETWORK {
		if _, err := self.validateCreateNetworkLBData(ownerId, data); err != nil {
			return nil, err
		}
	} else {
		return nil, httperrors.NewInputParameterError("invalid parameter loadbalancer_spec %s", spec)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerData(ctx, userCred, data)
}

func (self *SAwsRegionDriver) validateCreateApplicationListenerData(ctx context.Context, ownerId mcclient.IIdentityProvider, lb *models.SLoadbalancer, backendGroup *models.SLoadbalancerBackendGroup, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	listenerTypeV := validators.NewStringChoicesValidator("listener_type", api.AWS_APPLICATION_LB_LISTENER_TYPES)
	listenerPortV := validators.NewPortValidator("listener_port")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),

		"listener_type": listenerTypeV,
		"listener_port": listenerPortV,

		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Default(api.LB_SENDPROXY_OFF),

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES).Default(api.LB_BOOL_OFF),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", choices.NewChoices(api.LB_STICKY_SESSION_TYPE_INSERT)).Default(api.LB_STICKY_SESSION_TYPE_INSERT),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)).Optional(true),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout").Optional(true),
	}

	if err := RunValidators(keyV, data); err != nil {
		return nil, err
	}

	//  listener uniqueness
	listenerType := listenerTypeV.Value
	err := models.LoadbalancerListenerManager.CheckAwsListenerUniqueness(ctx, lb, nil, listenerType, listenerPortV.Value)
	if err != nil {
		return nil, err
	}

	// backendgroup check & vpc
	if backendGroup != nil {
		if backendGroup.LoadbalancerId == "" {
			_, err := db.Update(backendGroup, func() error {
				backendGroup.LoadbalancerId = lb.Id
				return nil
			})
			if err != nil {
				return nil, err
			}
		}

		if backendGroup.LoadbalancerId != lb.Id {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, lb.Id)
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

		if err := RunValidators(httpsV, data); err != nil {
			return nil, err
		}
	}

	// health check default depends on input parameters
	healthTypeChoices := choices.NewChoices(api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS)
	keyVHealth := map[string]validators.IValidator{
		"health_check":      validators.NewStringChoicesValidator("health_check", choices.NewChoices(api.LB_BOOL_ON)).Default(api.LB_BOOL_ON),
		"health_check_type": validators.NewStringChoicesValidator("health_check_type", healthTypeChoices).Default(api.LB_HEALTH_CHECK_HTTP),

		"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true).Default(""),
		"health_check_path":      validators.NewURLPathValidator("health_check_path").Default(""),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10).Default(5),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 2, 10).Default(2),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 2, 120).Default(5),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 5, 300).Default(30),
	}

	if err := RunValidators(keyVHealth, data); err != nil {
		return nil, err
	}

	data.Set("acl_status", jsonutils.NewString(api.LB_BOOL_OFF))
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	data.Set("scheduler", jsonutils.NewString(api.LB_SCHEDULER_RR)) // aws 不支持指定调度算法
	return data, nil
}

func (self *SAwsRegionDriver) validateCreateNetworkListenerData(ctx context.Context, ownerId mcclient.IIdentityProvider, lb *models.SLoadbalancer, backendGroup *models.SLoadbalancerBackendGroup, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	listenerTypeV := validators.NewStringChoicesValidator("listener_type", api.AWS_NETWORK_LB_LISTENER_TYPES)
	listenerPortV := validators.NewPortValidator("listener_port")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),

		"listener_type": listenerTypeV,
		"listener_port": listenerPortV,

		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Default(api.LB_SENDPROXY_OFF),
	}

	if err := RunValidators(keyV, data); err != nil {
		return nil, err
	}

	//  listener uniqueness
	listenerType := listenerTypeV.Value
	err := models.LoadbalancerListenerManager.CheckAwsListenerUniqueness(ctx, lb, nil, listenerType, listenerPortV.Value)
	if err != nil {
		return nil, err
	}

	// backendgroup check
	if backendGroup != nil {
		if backendGroup.LoadbalancerId == "" {
			_, err := db.Update(backendGroup, func() error {
				backendGroup.LoadbalancerId = lb.Id
				return nil
			})
			if err != nil {
				return nil, err
			}
		}

		if backendGroup.LoadbalancerId != lb.Id {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, lb.Id)
		}
	}

	// health check default depends on input parameters
	// 不支持指定http_code
	healthTypeChoices := choices.NewChoices(api.LB_HEALTH_CHECK_TCP, api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS)
	keyVHealth := map[string]validators.IValidator{
		"health_check":      validators.NewStringChoicesValidator("health_check", choices.NewChoices(api.LB_BOOL_ON)).Default(api.LB_BOOL_ON),
		"health_check_type": validators.NewStringChoicesValidator("health_check_type", healthTypeChoices).Default(api.LB_HEALTH_CHECK_TCP),

		"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true).Default(""),
		"health_check_path":      validators.NewURLPathValidator("health_check_path").Default(""),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10).Default(3),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 10, 10).Default(10),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 10, 30).Default(30),
	}

	if err := RunValidators(keyVHealth, data); err != nil {
		return nil, err
	}

	healthCheckRise, _ := data.Int("health_check_rise")
	data.Set("health_check_fall", jsonutils.NewInt(healthCheckRise))
	data.Set("sticky_session", jsonutils.NewString(api.LB_BOOL_OFF))
	data.Set("acl_status", jsonutils.NewString(api.LB_BOOL_OFF))
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	data.Set("scheduler", jsonutils.NewString(api.LB_SCHEDULER_RR)) // aws 不支持指定调度算法
	return data, nil
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup)
	if !ok {
		return nil, httperrors.NewInputParameterError("invalid parameter backendgroup %s", backendGroup.GetId())
	}

	if lb.LoadbalancerSpec == api.LB_AWS_SPEC_APPLICATION {
		if _, err := self.validateCreateApplicationListenerData(ctx, ownerId, lb, lbbg, data); err != nil {
			return nil, err
		}
	} else if lb.LoadbalancerSpec == api.LB_AWS_SPEC_NETWORK {
		if _, err := self.validateCreateNetworkListenerData(ctx, ownerId, lb, lbbg, data); err != nil {
			return nil, err
		}
	} else {
		return nil, httperrors.NewInputParameterError("invalid loadbalancer_spec %s", lb.LoadbalancerSpec)
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerListenerData(ctx, userCred, ownerId, data, lb, backendGroup)
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer acl", self.GetProvider())
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer certificate", self.GetProvider())
}

func (self *SAwsRegionDriver) validateUpdateApplicationListenerData(ctx context.Context, ownerId mcclient.IIdentityProvider, lb *models.SLoadbalancer, lblis *models.SLoadbalancerListener, backendGroup db.IModel, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	listenerTypeV := validators.NewStringChoicesValidator("listener_type", api.AWS_APPLICATION_LB_LISTENER_TYPES)
	listenerPortV := validators.NewPortValidator("listener_port")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),

		"listener_type": listenerTypeV,
		"listener_port": listenerPortV,
		"send_proxy":    validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Default(api.LB_SENDPROXY_OFF),

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES).Default(api.LB_BOOL_OFF),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", choices.NewChoices(api.LB_STICKY_SESSION_TYPE_INSERT)).Default(api.LB_STICKY_SESSION_TYPE_INSERT),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)).Optional(true),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout").Optional(true),
	}

	if err := RunValidators(keyV, data); err != nil {
		return nil, err
	}

	//  listener uniqueness
	listenerType := listenerTypeV.Value
	err := models.LoadbalancerListenerManager.CheckAwsListenerUniqueness(ctx, lb, lblis, listenerType, listenerPortV.Value)
	if err != nil {
		return nil, err
	}

	// backendgroup check & vpc
	if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && (lbbg.LoadbalancerId != "" && lbbg.LoadbalancerId != lb.Id) {
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

		if err := RunValidators(httpsV, data); err != nil {
			return nil, err
		}
	}

	// health check default depends on input parameters
	healthTypeChoices := choices.NewChoices(api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS)
	keyVHealth := map[string]validators.IValidator{
		"health_check":      validators.NewStringChoicesValidator("health_check", choices.NewChoices(api.LB_BOOL_ON)).Default(api.LB_BOOL_ON),
		"health_check_type": validators.NewStringChoicesValidator("health_check_type", healthTypeChoices).Default(api.LB_HEALTH_CHECK_HTTP),

		"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true).Default(""),
		"health_check_path":      validators.NewURLPathValidator("health_check_path").Default(""),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10).Default(5),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 2, 10).Default(2),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 2, 120).Default(5),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 5, 300).Default(30),
	}

	if err := RunValidators(keyVHealth, data); err != nil {
		return nil, err
	}

	data.Set("acl_status", jsonutils.NewString(api.LB_BOOL_OFF))
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	data.Set("scheduler", jsonutils.NewString(api.LB_SCHEDULER_RR)) // aws 不支持指定调度算法
	return data, nil
}

func (self *SAwsRegionDriver) validateUpdateNetworkListenerData(ctx context.Context, ownerId mcclient.IIdentityProvider, lb *models.SLoadbalancer, lblis *models.SLoadbalancerListener, backendGroup db.IModel, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	listenerTypeV := validators.NewStringChoicesValidator("listener_type", api.AWS_APPLICATION_LB_LISTENER_TYPES)
	listenerPortV := validators.NewPortValidator("listener_port")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),

		"listener_type": listenerTypeV,
		"listener_port": listenerPortV,

		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Default(api.LB_SENDPROXY_OFF),
	}

	if err := RunValidators(keyV, data); err != nil {
		return nil, err
	}

	//  listener uniqueness
	listenerType := listenerTypeV.Value
	err := models.LoadbalancerListenerManager.CheckAwsListenerUniqueness(ctx, lb, lblis, listenerType, listenerPortV.Value)
	if err != nil {
		return nil, err
	}

	// backendgroup check
	if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && (lbbg.LoadbalancerId != "" && lbbg.LoadbalancerId != lb.Id) {
		return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
			lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, lb.Id)
	}

	// health check default depends on input parameters
	// 不支持修改协议及指定http_code
	healthTypeChoices := choices.NewChoices(api.LB_HEALTH_CHECK_TCP, api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS)
	keyVHealth := map[string]validators.IValidator{
		"health_check":      validators.NewStringChoicesValidator("health_check", choices.NewChoices(api.LB_BOOL_ON)).Default(api.LB_BOOL_ON),
		"health_check_type": validators.NewStringChoicesValidator("health_check_type", healthTypeChoices).Default(api.LB_HEALTH_CHECK_TCP),

		"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true).Default(""),
		"health_check_path":      validators.NewURLPathValidator("health_check_path").Default(""),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(api.LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 2, 10).Default(3),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 2, 10).Default(3),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 10, 10).Default(10),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 10, 30).Default(30),
	}

	if err := RunValidators(keyVHealth, data); err != nil {
		return nil, err
	}

	data.Set("sticky_session", jsonutils.NewString(api.LB_BOOL_OFF))
	data.Set("acl_status", jsonutils.NewString(api.LB_BOOL_OFF))
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	data.Set("scheduler", jsonutils.NewString(api.LB_SCHEDULER_RR)) // aws 不支持指定调度算法
	return data, nil
}

func (self *SAwsRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lblis *models.SLoadbalancerListener, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	ownerId := lblis.GetOwnerId()
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", ownerId)
	if err := lbV.Validate(data); err != nil {
		return nil, err
	}

	lb := lbV.Model.(*models.SLoadbalancer)
	if lb.LoadbalancerSpec == api.LB_AWS_SPEC_APPLICATION {
		if _, err := self.validateUpdateApplicationListenerData(ctx, ownerId, lb, lblis, backendGroup, data); err != nil {
			return nil, err
		}
	} else if lb.LoadbalancerSpec == api.LB_AWS_SPEC_NETWORK {
		if _, err := self.validateUpdateNetworkListenerData(ctx, ownerId, lb, lblis, backendGroup, data); err != nil {
			return nil, err
		}
	} else {
		return nil, httperrors.NewInputParameterError("invalid parameter loadbalancer_spec %s", lb.LoadbalancerSpec)
	}
	return self.SManagedVirtualizationRegionDriver.ValidateUpdateLoadbalancerListenerData(ctx, userCred, data, lblis, backendGroup)
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	domainV := validators.NewDomainNameValidator("domain")
	pathV := validators.NewURLPathValidator("path")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),
		"domain": domainV.AllowEmpty(true).Default("").Optional(true),
		"path":   pathV.Default("").Optional(true),
	}

	if err := RunValidators(keyV, data); err != nil {
		return nil, err
	}

	condition, err := data.GetString("condition")
	if err != nil || len(condition) == 0 {
		if len(pathV.Value) == 0 && len(domainV.Value) == 0 {
			return nil, httperrors.NewMissingParameterError("condition")
		} else {
			segs := []string{}
			if len(pathV.Value) > 0 {
				segs = append(segs, fmt.Sprintf(`{"field":"path-pattern","pathPatternConfig":{"values":["%s"]},"values":["%s"]}`, pathV.Value, pathV.Value))
			}

			if len(domainV.Value) > 0 {
				segs = append(segs, fmt.Sprintf(`{"field":"host-header","hostHeaderConfig":{"values":["%s"]},"values":["%s"]}`, domainV.Value, domainV.Value))
			}
			condition = fmt.Sprintf(`[%s]`, strings.Join(segs, ","))
		}
	}

	if err := models.ValidateListenerRuleConditions(condition); err != nil {
		return nil, httperrors.NewInputParameterError("%s", err)
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

	// check backend group protocol http & https
	// data.Remove("domain")
	// data.Remove("path")
	data.Set("condition", jsonutils.NewString(condition))
	data.Set("cloudregion_id", jsonutils.NewString(listener.CloudregionId))
	data.Set("manager_id", jsonutils.NewString(listener.ManagerId))
	return data, nil
}

func (self *SAwsRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	lbr := ctx.Value("lbr").(*models.SLoadbalancerListenerRule)

	condition, err := data.GetString("condition")
	if err == nil {
		if err := models.ValidateListenerRuleConditions(condition); err != nil {
			return nil, httperrors.NewInputParameterError("%s", err)
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

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendType string, lb *models.SLoadbalancer, backendGroup *models.SLoadbalancerBackendGroup, backend db.IModel) (*jsonutils.JSONDict, error) {
	ownerId := ctx.Value("ownerId").(mcclient.IIdentityProvider)
	man := models.LoadbalancerBackendManager
	portV := validators.NewPortValidator("port")
	keyV := map[string]validators.IValidator{
		"weight":     validators.NewRangeValidator("weight", 0, 100).Default(10),
		"port":       portV,
		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Default(api.LB_SENDPROXY_OFF),
	}

	if err := RunValidators(keyV, data); err != nil {
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

		count, err := man.Query().IsFalse("pending_deleted").Equals("backend_group_id", backendGroup.GetId()).Equals("backend_id", backendV.Model.GetId()).Equals("port", portV.Value).CountWithError()
		if err != nil {
			return nil, err
		}

		if count > 0 {
			return nil, httperrors.NewInputParameterError("The backend %s is already registered on port %d", backendV.Model.GetId(), portV.Value)
		}

		basename = guest.Name
		backend = backendV.Model

		address, err := models.LoadbalancerBackendManager.GetGuestAddress(guest)
		if err != nil {
			return nil, err
		}
		data.Set("address", jsonutils.NewString(address))
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

func (self *SAwsRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lbbg *models.SLoadbalancerBackendGroup) (*jsonutils.JSONDict, error) {
	keyV := map[string]validators.IValidator{
		"weight":     validators.NewRangeValidator("weight", 0, 100).Optional(true),
		"port":       validators.NewPortValidator("port").Optional(true),
		"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES).Optional(true),
	}

	if err := RunValidators(keyV, data); err != nil {
		return nil, err
	}

	// 不能更新端口和权重
	port, err := data.Int("port")
	if err == nil && port != 0 {
		return data, fmt.Errorf("can not update backend port.")
	}

	weight, err := data.Int("weight")
	if err == nil && weight != 0 {
		return data, fmt.Errorf("can not update backend weight.")
	}

	return data, nil
}

func (self *SAwsRegionDriver) createLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, lblis *models.SLoadbalancerListener, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend) (jsonutils.JSONObject, error) {
	// create loadbalancer backendgroup cache
	cachedLbbg := &models.SAwsCachedLbbg{}
	cachedLbbg.ManagerId = lb.ManagerId
	cachedLbbg.CloudregionId = lb.CloudregionId
	cachedLbbg.LoadbalancerId = lb.GetId()
	cachedLbbg.BackendGroupId = lbbg.GetId()
	cachedLbbg.ProtocolType = lblis.ListenerType
	cachedLbbg.Port = lblis.ListenerPort
	cachedLbbg.TargetType = "instance"
	cachedLbbg.Status = api.LB_STATUS_ENABLED
	cachedLbbg.HealthCheckProtocol = lblis.HealthCheckType
	cachedLbbg.HealthCheckInterval = lblis.HealthCheckInterval

	err := models.AwsCachedLbbgManager.TableSpec().Insert(cachedLbbg)
	if err != nil {
		return nil, err
	}

	group, err := lbbg.GetAwsBackendGroupParams(lblis, nil)
	if err != nil {
		return nil, err
	}

	iRegion, err := lbbg.GetIRegion()
	if err != nil {
		return nil, err
	}

	iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
	if err != nil {
		return nil, err
	}

	iLoadbalancerBackendGroup, err := iLoadbalancer.CreateILoadBalancerBackendGroup(group)
	if err != nil {
		return nil, err
	}

	cachedLbbg.SetModelManager(models.AwsCachedLbbgManager, cachedLbbg)
	if err := db.SetExternalId(cachedLbbg, userCred, iLoadbalancerBackendGroup.GetGlobalId()); err != nil {
		return nil, err
	}

	for _, backend := range backends {
		cachedlbb := &models.SAwsCachedLb{}
		cachedlbb.ManagerId = lb.ManagerId
		cachedlbb.CloudregionId = lb.CloudregionId
		cachedlbb.CachedBackendGroupId = cachedLbbg.GetId()
		cachedlbb.BackendId = backend.ID
		err = models.AwsCachedLbManager.TableSpec().Insert(cachedlbb)
		if err != nil {
			return nil, err
		}

		ibackend, err := iLoadbalancerBackendGroup.AddBackendServer(backend.ExternalID, backend.Weight, backend.Port)
		if err != nil {
			return nil, err
		}

		cachedlbb.SetModelManager(models.AwsCachedLbManager, cachedlbb)
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
		models.AwsCachedLbManager.SyncLoadbalancerBackends(ctx, userCred, provider, cachedLbbg, iBackends, &models.SSyncRange{})
	}
	return nil, nil

}

func (self *SAwsRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	// 必须AwsLoadbalancerLoadbalancerBackendGroupCreateTask的调用，才实际执行创建
	if _, ok := task.(*tasks.AwsLoadbalancerLoadbalancerBackendGroupCreateTask); !ok {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return nil, nil
		})
		return nil
	}

	lbId, err := task.GetParams().GetString("loadbalancer_id")
	if err != nil {
		return fmt.Errorf("missing loadbalancer id")
	}

	lb, err := db.FetchByExternalId(models.LoadbalancerManager, lbId)
	if err != nil {
		return err
	}

	listenerId, err := task.GetParams().GetString("listener_id")
	if err != nil {
		return fmt.Errorf("missing listenerId")
	}

	lblis, err := db.FetchByExternalId(models.LoadbalancerListenerManager, listenerId)
	if err != nil {
		return err
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.createLoadbalancerBackendGroup(ctx, userCred, lb.(*models.SLoadbalancer), lblis.(*models.SLoadbalancerListener), lbbg, backends)
	})
	return nil
}

func (self *SAwsRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg := lbb.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			return nil, fmt.Errorf("failed to find lbbg for backend %s", lbb.Name)
		}
		lb := lbbg.GetLoadbalancer()
		if lb == nil {
			return nil, fmt.Errorf("failed to find lb for backendgroup %s", lbbg.Name)
		}

		cachedlbbgs, err := models.AwsCachedLbbgManager.GetCachedBackendGroups(lbbg.GetId())
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

			_, err = models.AwsCachedLbManager.CreateAwsCachedLb(ctx, userCred, lbb, &cachedLbbg, ibackend, cachedLbbg.GetOwnerId())
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

func (self *SAwsRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}

		cachedlbbs, err := models.AwsCachedLbManager.GetBackendsByLocalBackendId(lbb.GetId())
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

func (self *SAwsRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}

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

		{
			lbbg := lblis.GetLoadbalancerBackendGroup()
			if lbbg == nil {
				return nil, fmt.Errorf("aws loadbalancer listener releated backend group not found")
			}

			params, err := lbbg.GetAwsBackendGroupParams(lblis, nil)
			if err != nil {
				return nil, err
			}

			group, _ := models.AwsCachedLbbgManager.GetUsableCachedBackendGroup(lblis.LoadbalancerId, lblis.BackendGroupId, lblis.ListenerType, lblis.HealthCheckType, lblis.HealthCheckInterval)
			if group != nil {
				// 服务器组存在
				ilbbg, err := group.GetICloudLoadbalancerBackendGroup()
				if err != nil {
					return nil, err
				}
				// 服务器组已经存在，直接同步即可
				if err := ilbbg.Sync(params); err != nil {
					return nil, err
				}
			} else {
				backends, err := lbbg.GetBackendsParams()
				if err != nil {
					return nil, err
				}
				// 服务器组不存在
				_, err = self.createLoadbalancerBackendGroup(ctx, userCred, loadbalancer, lblis, lbbg, backends)
				if err != nil {
					return nil, err
				}
			}
		}

		params, err := lblis.GetAwsLoadbalancerListenerParams()
		if err != nil {
			return nil, err
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

		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, loadbalancer.GetOwnerId())
	})
	return nil
}

func (self *SAwsRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
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
			Name:      lbr.Name,
			Condition: lbr.Condition,
		}

		group, err := models.AwsCachedLbbgManager.GetUsableCachedBackendGroup(listener.LoadbalancerId, listener.BackendGroupId, listener.ListenerType, listener.HealthCheckType, listener.HealthCheckInterval)
		if err != nil {
			return nil, err
		}

		rule.BackendGroupID = group.ExternalId
		rule.BackendGroupType = group.TargetType

		iListenerRule, err := iListener.CreateILoadBalancerListenerRule(rule)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lbr, userCred, iListenerRule.GetGlobalId()); err != nil {
			return nil, err
		}
		return nil, lbr.SyncWithCloudLoadbalancerListenerRule(ctx, userCred, iListenerRule, nil)
	})
	return nil
}

func (self *SAwsRegionDriver) RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}

		if len(lb.ExternalId) == 0 {
			return nil, nil
		}

		{
			lbbgs, err := lb.GetLoadbalancerBackendgroups()
			if err != nil {
				return nil, err
			}

			for i := range lbbgs {
				lbbg := &lbbgs[i]
				err := self.RequestDeleteLoadbalancerBackendGroup(ctx, userCred, lbbg, task)
				if err != nil {
					lbbg.SetStatus(userCred, api.LB_STATUS_DELETE_FAILED, err.Error())
					return nil, err
				} else {
					lbbg.LBPendingDelete(ctx, userCred)
				}
			}
		}

		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iLoadbalancer.Delete()
	})
	return nil
}

func (self *SAwsRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
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

				// lblis.ExternalId = lbcert.ExternalId
			}
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
					_, err := self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, err
					}
				}

				// lblis.AclId = lbacl.ExternalId
			}
		}

		params, err := lblis.GetAwsLoadbalancerListenerParams()
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
		if err := iListener.Refresh(); err != nil {
			return nil, err
		}
		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, nil)
	})
	return nil
}
