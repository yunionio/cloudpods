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
	"sort"
	"strconv"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	randutil "yunion.io/x/pkg/util/rand"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SKVMRegionDriver struct {
	SBaseRegionDriver
}

func init() {
	driver := SKVMRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func RunValidators(ctx context.Context, validators map[string]validators.IValidator, data *jsonutils.JSONDict, optional bool) error {
	for _, v := range validators {
		if optional {
			v.Optional(true)
		}
		if err := v.Validate(ctx, data); err != nil {
			return err
		}
	}

	return nil
}

func (self *SKVMRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SKVMRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		err := input.Rules[i].Check()
		if err != nil {
			return input, httperrors.NewInputParameterError("rule %d is invalid: %s", i, err)
		}
	}
	return input, nil
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error) {
	// find available networks
	var network *models.SNetwork = nil
	if len(input.NetworkId) > 0 {
		netObj, err := validators.ValidateModel(ctx, userCred, models.NetworkManager, &input.NetworkId)
		if err != nil {
			return nil, err
		}
		network = netObj.(*models.SNetwork)
	} else if len(input.VpcId) > 0 {
		vpcObj, err := validators.ValidateModel(ctx, userCred, models.VpcManager, &input.VpcId)
		if err != nil {
			return nil, err
		}
		vpc := vpcObj.(*models.SVpc)
		networks, err := vpc.GetNetworks()
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		networksLen := len(networks)
		if networksLen > 0 {
			i := randutil.Intn(networksLen)
			j := (i + 1) % networksLen
			for {
				net := &networks[j]
				addrCount, err := net.GetFreeAddressCount()
				if err != nil {
					continue
				}
				if addrCount > 0 {
					network = net
					break
				}

				j = (j + 1) % networksLen
				if j == i {
					break
				}
			}
		}
		if network == nil {
			return nil, httperrors.NewBadRequestError("no usable network in vpc %s(%s)", vpc.Name, vpc.Id)
		}
	} else {
		return nil, httperrors.NewMissingParameterError("network_id")
	}

	if network.ServerType != api.NETWORK_TYPE_GUEST {
		return nil, httperrors.NewBadRequestError("only network type %q is allowed", api.NETWORK_TYPE_GUEST)
	}

	if len(input.ClusterId) > 0 {
		clusterObj, err := validators.ValidateModel(ctx, userCred, models.LoadbalancerClusterManager, &input.ClusterId)
		if err != nil {
			return nil, err
		}
		cluster := clusterObj.(*models.SLoadbalancerCluster)
		input.ZoneId = cluster.ZoneId
		if cluster.WireId != "" && cluster.WireId != network.WireId {
			return nil, httperrors.NewInputParameterError("cluster wire affiliation does not match network's: %s != %s",
				cluster.WireId, network.WireId)
		}
	} else {
		if len(input.ZoneId) == 0 {
			return nil, httperrors.NewMissingParameterError("zone_id")
		}
		clusters := models.LoadbalancerClusterManager.FindByZoneId(input.ZoneId)
		if len(clusters) == 0 {
			return nil, httperrors.NewInputParameterError("zone %s has no lbcluster", input.ZoneId)
		}
		var (
			wireMatched []*models.SLoadbalancerCluster
			wireNeutral []*models.SLoadbalancerCluster
		)
		for i := range clusters {
			c := &clusters[i]
			if c.WireId != "" {
				if c.WireId == network.WireId {
					wireMatched = append(wireMatched, c)
				}
			} else {
				wireNeutral = append(wireNeutral, c)
			}
		}
		var choices []*models.SLoadbalancerCluster
		if len(wireMatched) > 0 {
			choices = wireMatched
		} else if len(wireNeutral) > 0 {
			choices = wireNeutral
		} else {
			return nil, httperrors.NewInputParameterError("no viable lbcluster")
		}
		i := randutil.Intn(len(choices))
		input.ClusterId = choices[i].Id
	}

	input.NetworkType = api.LB_NETWORK_TYPE_VPC
	if input.VpcId == api.DEFAULT_VPC_ID {
		input.NetworkType = api.LB_NETWORK_TYPE_CLASSIC
	}
	return input, nil
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, input *api.LoadbalancerBackendGroupCreateInput) (*api.LoadbalancerBackendGroupCreateInput, error) {
	return input, nil
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup, input *api.LoadbalancerBackendCreateInput) (*api.LoadbalancerBackendCreateInput, error) {
	return input, nil
}

func (self *SKVMRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, input *api.LoadbalancerBackendUpdateInput) (*api.LoadbalancerBackendUpdateInput, error) {
	return input, nil
}

func (self *SKVMRegionDriver) IsSupportLoadbalancerListenerRuleRedirect() bool {
	return true
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerRuleCreateInput) (*api.LoadbalancerListenerRuleCreateInput, error) {
	return input, nil
}

func (self *SKVMRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LoadbalancerListenerRuleUpdateInput) (*api.LoadbalancerListenerRuleUpdateInput, error) {
	return input, nil
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerCreateInput,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup) (*api.LoadbalancerListenerCreateInput, error) {
	return input, nil
}

func (self *SKVMRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput) (*api.LoadbalancerListenerUpdateInput, error) {
	/*

		certV := validators.NewModelIdOrNameValidator("certificate", "loadbalancercertificate", ownerId)
		tlsCipherPolicyV := validators.NewStringChoicesValidator("tls_cipher_policy", api.LB_TLS_CIPHER_POLICIES).Default(api.LB_TLS_CIPHER_POLICY_1_2)
		keyV := map[string]validators.IValidator{
			"send_proxy": validators.NewStringChoicesValidator("send_proxy", api.LB_SENDPROXY_CHOICES),

			"acl_status": aclStatusV,
			"acl_type":   aclTypeV,
			"acl":        aclV,

			"scheduler":   validators.NewStringChoicesValidator("scheduler", api.LB_SCHEDULER_TYPES),
			"egress_mbps": validators.NewRangeValidator("egress_mbps", api.LB_MbpsMin, api.LB_MbpsMax),

			"client_request_timeout":  validators.NewRangeValidator("client_request_timeout", 0, 600),
			"client_idle_timeout":     validators.NewRangeValidator("client_idle_timeout", 0, 600),
			"backend_connect_timeout": validators.NewRangeValidator("backend_connect_timeout", 0, 180),
			"backend_idle_timeout":    validators.NewRangeValidator("backend_idle_timeout", 0, 600),

			"sticky_session":                validators.NewStringChoicesValidator("sticky_session", api.LB_BOOL_VALUES),
			"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", api.LB_STICKY_SESSION_TYPES),
			"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)),
			"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout"),

			"health_check":      validators.NewStringChoicesValidator("health_check", api.LB_BOOL_VALUES),
			"health_check_type": models.LoadbalancerListenerManager.CheckTypeV(lblis.ListenerType),

			"health_check_domain":    validators.NewDomainNameValidator("health_check_domain").AllowEmpty(true),
			"health_check_path":      validators.NewURLPathValidator("health_check_path"),
			"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", api.LB_HEALTH_CHECK_HTTP_CODES).Sep(","),

			"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 1000),
			"health_check_fall":     validators.NewRangeValidator("health_check_fall", 1, 1000),
			"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 300),
			"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 1000),

			"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for"),
			"gzip":            validators.NewBoolValidator("gzip"),

			"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate"),
			"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src"),

			"certificate":       certV,
			"tls_cipher_policy": tlsCipherPolicyV,
			"enable_http2":      validators.NewBoolValidator("enable_http2"),

			"redirect":        redirectV,
			"redirect_code":   redirectCodeV,
			"redirect_scheme": redirectSchemeV,
			"redirect_host":   redirectHostV.AllowEmpty(true),
			"redirect_path":   redirectPathV.AllowEmpty(true),
		}

		if err := RunValidators(keyV, data, true); err != nil {
			return nil, err
		}

		var (
			redirectType = redirectV.Value
			listenerType = lblis.ListenerType
		)
		if redirectType != api.LB_REDIRECT_OFF {
			if redirectType == api.LB_REDIRECT_RAW {
				scheme, host, path := redirectSchemeV.Value, redirectHostV.Value, redirectPathV.Value
				if (scheme == "" || scheme == listenerType) && host == "" && path == "" {
					return nil, httperrors.NewInputParameterError("redirect must have at least one of scheme, host, path changed")
				}
			}
		}
		// NOTE: it's okay we turn off redirect
		//
		//  - scheduler have default value on creation
		//  - backend_group_id is allowed to have unset value for http, https listener

		if err := models.LoadbalancerListenerManager.ValidateAcl(aclStatusV, aclTypeV, aclV, data, lblis.GetProviderName()); err != nil {
			return nil, err
		}

		{
			if backendGroup == nil {
				if lblis.ListenerType != api.LB_LISTENER_TYPE_HTTP &&
					lblis.ListenerType != api.LB_LISTENER_TYPE_HTTPS {
					return nil, httperrors.NewInputParameterError("non http listener must have backend group set")
				}
			} else if lbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != lblis.LoadbalancerId {
				return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
					lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, lblis.LoadbalancerId)
			}
		}
	*/

	return input, nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerInstance(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, input *api.LoadbalancerCreateInput, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_, err := db.Update(lb, func() error {
			// TODO support use reserved ip address
			req := &models.SLoadbalancerNetworkRequestData{
				Loadbalancer: lb,
				NetworkId:    lb.NetworkId,
				Address:      lb.Address,
			}
			// NOTE the small window when agents can see the ephemeral address
			ln, err := models.LoadbalancernetworkManager.NewLoadbalancerNetwork(ctx, userCred, req)
			if err != nil {
				log.Errorf("allocating loadbalancer network failed: %v, req: %#v", err, req)
				lb.Address = ""
			} else {
				lb.Address = ln.IpAddr
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "db.Update")
		}
		// bind eip
		eipAddr := ""
		if input.EipBw > 0 && len(input.EipId) == 0 {
			// create eip first
			eipPendingUsage := &models.SRegionQuota{Eip: 1}
			eipPendingUsage.SetKeys(lb.GetQuotaKeys())
			eip, err := models.ElasticipManager.NewEipForVMOnHost(ctx, userCred, &models.NewEipForVMOnHostArgs{
				Bandwidth:     input.EipBw,
				BgpType:       input.EipBgpType,
				ChargeType:    input.EipChargeType,
				AutoDellocate: input.EipAutoDellocate,

				Loadbalancer: lb,
				PendingUsage: eipPendingUsage,
			})
			if err != nil {
				log.Errorf("NewEipForVMOnHost fail %s", err)
				quotas.CancelPendingUsage(ctx, userCred, eipPendingUsage, eipPendingUsage, false)
			} else {
				eipAddr = eip.IpAddr
				opts := api.ElasticipAssociateInput{
					InstanceId:         lb.Id,
					InstanceExternalId: lb.ExternalId,
					InstanceType:       api.EIP_ASSOCIATE_TYPE_LOADBALANCER,
				}

				err = eip.AllocateAndAssociateInstance(ctx, userCred, lb, opts, "")
				if err != nil {
					return nil, errors.Wrap(err, "AllocateAndAssociateInstance")
				}
			}
		} else if len(input.EipId) > 0 {
			_eip, err := models.ElasticipManager.FetchById(input.EipId)
			if err != nil {
				return nil, errors.Wrapf(err, "ElasticipManager.FetchById(%s)", input.EipId)
			}
			eip := _eip.(*models.SElasticip)
			err = eip.AssociateLoadbalancer(ctx, userCred, lb)
			if err != nil {
				return nil, errors.Wrapf(err, "eip.AssociateLoadbalancer")
			}
			eipAddr = eip.IpAddr
		}

		if len(eipAddr) > 0 {
			_, err = db.Update(lb, func() error {
				lb.Address = eipAddr
				lb.AddressType = api.LB_ADDR_TYPE_INTERNET
				return nil
			})
			if err != nil {
				return nil, errors.Wrap(err, "set loadbalancer address")
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SKVMRegionDriver) RequestStartLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestStopLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestSyncstatusLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	originStatus, _ := task.GetParams().GetString("origin_status")
	if utils.IsInStringArray(originStatus, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_DISABLED}) {
		lb.SetStatus(ctx, userCred, originStatus, "")
	} else {
		lb.SetStatus(ctx, userCred, api.LB_STATUS_ENABLED, "")
	}
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestUpdateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestLoadbalancerAclSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	lbacl.SetStatus(ctx, userCred, apis.STATUS_AVAILABLE, "")
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestLoadbalancerCertificateSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	lbcert.SetStatus(ctx, userCred, apis.STATUS_AVAILABLE, "")
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) GetBackendStatusForAdd() []string {
	return []string{api.VM_RUNNING, api.VM_READY}
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestStartLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestStopLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestSyncstatusLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	originStatus, _ := task.GetParams().GetString("origin_status")
	if utils.IsInStringArray(originStatus, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_DISABLED}) {
		lblis.SetStatus(ctx, userCred, originStatus, "")
	} else {
		lblis.SetStatus(ctx, userCred, api.LB_STATUS_ENABLED, "")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	if !utils.IsInStringArray(input.CidrBlock, []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"}) {
		return input, httperrors.NewInputParameterError("Invalid cidr_block, want 192.168.0.0/16|10.0.0.0/8|172.16.0.0/12, got %s", input.CidrBlock)
	}
	return input, nil
}

func (self *SKVMRegionDriver) RequestDeleteVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) GetEipDefaultChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
}

func (self *SKVMRegionDriver) ValidateEipChargeType(chargeType string) error {
	if chargeType != api.EIP_CHARGE_TYPE_BY_BANDWIDTH {
		return httperrors.NewInputParameterError("%s only supports eip charge type %q",
			self.GetProvider(), api.EIP_CHARGE_TYPE_BY_BANDWIDTH)
	}
	return nil
}

func (self *SKVMRegionDriver) ValidateCreateEipData(ctx context.Context, userCred mcclient.TokenCredential, input *api.SElasticipCreateInput) error {
	if err := self.ValidateEipChargeType(input.ChargeType); err != nil {
		return err
	}
	var network *models.SNetwork
	if input.NetworkId != "" {
		_network, err := models.NetworkManager.FetchByIdOrName(ctx, userCred, input.NetworkId)
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError2("network", input.NetworkId)
			}
			return httperrors.NewGeneralError(err)
		}
		network = _network.(*models.SNetwork)
		input.BgpType = network.BgpType
	} else {
		q := models.NetworkManager.Query().
			Equals("server_type", api.NETWORK_TYPE_EIP).
			Equals("bgp_type", input.BgpType)
		var nets []models.SNetwork
		if err := db.FetchModelObjects(models.NetworkManager, q, &nets); err != nil {
			return err
		}
		eipNets := make([]models.SEipNetwork, 0)
		for i := range nets {
			net := &nets[i]
			cnt, _ := net.GetFreeAddressCount()
			if cnt > 0 {
				eipNets = append(eipNets, models.NewEipNetwork(net, userCred, userCred, cnt))
			}
		}
		if len(eipNets) == 0 {
			return httperrors.NewNotFoundError("no available eip network")
		}
		// prefer networks with identical project, domain, more free address, Id
		sort.Sort(models.SEipNetworks(eipNets))
		log.Debugf("eipnets: %s", jsonutils.Marshal(eipNets))
		network = eipNets[0].GetNetwork()
		input.NetworkId = network.Id
	}
	if network.ServerType != api.NETWORK_TYPE_EIP {
		return httperrors.NewInputParameterError("bad network type %q, want %q", network.ServerType, api.NETWORK_TYPE_EIP)
	}
	input.NetworkId = network.Id

	if len(input.IpAddr) > 0 {
		if !network.Contains(input.IpAddr) {
			return httperrors.NewInputParameterError("candidate %s out of range", input.IpAddr)
		}
		addrTable := network.GetUsedAddresses(ctx)
		if _, ok := addrTable[input.IpAddr]; ok {
			return httperrors.NewInputParameterError("requested ip %s is occupied!", input.IpAddr)
		}
	}

	vpc, _ := network.GetVpc()
	if vpc == nil {
		return httperrors.NewInputParameterError("failed to found vpc for network %s(%s)", network.Name, network.Id)
	}
	region, err := vpc.GetRegion()
	if err != nil {
		return err
	}
	if region.Id != input.CloudregionId {
		return httperrors.NewUnsupportOperationError("network %s(%s) does not belong to %s", network.Name, network.Id, self.GetProvider())
	}
	return nil
}

func (self *SKVMRegionDriver) ValidateSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot) error {
	storage := snapshot.GetStorage()
	if storage == nil {
		return httperrors.NewInternalServerError("Kvm snapshot missing storage ??")
	}
	return models.GetStorageDriver(storage.StorageType).ValidateSnapshotDelete(ctx, snapshot)
}

func (self *SKVMRegionDriver) RequestDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	storage := snapshot.GetStorage()
	if storage == nil {
		return httperrors.NewInternalServerError("Kvm snapshot missing storage ??")
	}
	return models.GetStorageDriver(storage.StorageType).RequestDeleteSnapshot(ctx, snapshot, task)
}

func (self *SKVMRegionDriver) RequestDeleteInstanceSnapshot(ctx context.Context, isp *models.SInstanceSnapshot, task taskman.ITask) error {
	snapshots, err := isp.GetSnapshots()
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		task.SetStage("OnInstanceSnapshotDelete", nil)
		if isp.WithMemory && isp.MemoryFileHostId != "" && isp.MemoryFilePath != "" {
			// request delete memory snapshot
			host := models.HostManager.FetchHostById(isp.MemoryFileHostId)
			if host == nil {
				return errors.Errorf("Not found host by %q", isp.MemoryFileHostId)
			}
			header := task.GetTaskRequestHeader()
			url := fmt.Sprintf("%s/servers/memory-snapshot", host.ManagerUri)
			if _, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "DELETE", url, header, jsonutils.Marshal(&hostapi.GuestMemorySnapshotDeleteRequest{
				InstanceSnapshotId: isp.GetId(),
				Path:               isp.MemoryFilePath,
			}), false); err != nil {
				return err
			}
		} else {
			taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
				return nil, nil
			})
		}
		return nil
	}

	params := jsonutils.NewDict()
	taskParams := task.GetParams()
	var deleteSnapshotTotalCnt int64 = 1
	if taskParams.Contains("snapshot_total_count") {
		deleteSnapshotTotalCnt, _ = taskParams.Int("snapshot_total_count")
	}
	deletedSnapshotCnt := deleteSnapshotTotalCnt - int64(len(snapshots))
	params.Set("del_snapshot_id", jsonutils.NewString(snapshots[0].Id))
	task.SetStage("OnKvmSnapshotDelete", params)
	err = snapshots[0].StartSnapshotDeleteTask(ctx, task.GetUserCred(), false, task.GetTaskId(), int(deleteSnapshotTotalCnt), int(deletedSnapshotCnt))
	if err != nil {
		return err
	}
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteInstanceBackup(ctx context.Context, ib *models.SInstanceBackup, task taskman.ITask) error {
	backups, err := ib.GetBackups()
	if err != nil {
		return err
	}
	if len(backups) == 0 {
		task.SetStage("OnInstanceBackupDelete", nil)
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return nil, nil
		})
		return nil
	}
	params := jsonutils.NewDict()
	params.Set("del_backup_id", jsonutils.NewString(backups[0].Id))
	task.SetStage("OnKvmDiskBackupDelete", params)
	forceDelete := jsonutils.QueryBoolean(task.GetParams(), "force_delete", false)
	err = backups[0].StartBackupDeleteTask(ctx, task.GetUserCred(), task.GetTaskId(), forceDelete)
	if err != nil {
		return err
	}
	return nil
}

func (self *SKVMRegionDriver) RequestResetToInstanceSnapshot(ctx context.Context, guest *models.SGuest, isp *models.SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error {
	jIsps, err := isp.GetInstanceSnapshotJointsByOrder(guest)
	if err != nil {
		return errors.Wrap(err, "GetInstanceSnapshotJointsByOrder")
	}
	diskIndexI64, err := params.Int("disk_index")
	if err != nil {
		return errors.Wrap(err, "get 'disk_index' from params")
	}
	diskIndex := int(diskIndexI64)
	if diskIndex >= len(jIsps) {
		task.SetStage("OnInstanceSnapshotReset", nil)
		withMem := jsonutils.QueryBoolean(params, "with_memory", false)
		if isp.WithMemory && withMem {
			// reset do memory snapshot
			host, err := guest.GetHost()
			if err != nil {
				return err
			}
			header := task.GetTaskRequestHeader()
			url := fmt.Sprintf("%s/servers/%s/memory-snapshot-reset", host.ManagerUri, guest.GetId())
			if _, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, jsonutils.Marshal(&hostapi.GuestMemorySnapshotResetRequest{
				InstanceSnapshotId: isp.GetId(),
				Path:               isp.MemoryFilePath,
				Checksum:           isp.MemoryFileChecksum,
			}), false); err != nil {
				return err
			}
		} else {
			taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
				return nil, nil
			})
		}
		return nil
	}

	isj := jIsps[diskIndex]

	params = jsonutils.NewDict()
	params.Set("disk_index", jsonutils.NewInt(int64(diskIndex)))
	task.SetStage("OnKvmDiskReset", params)

	disk, err := isj.GetSnapshotDisk()
	if err != nil {
		return errors.Wrapf(err, "Get %d snapshot disk", diskIndex)
	}
	err = disk.StartResetDisk(ctx, task.GetUserCred(), isj.SnapshotId, false, guest, task.GetTaskId())
	if err != nil {
		return errors.Wrap(err, "StartResetDisk")
	}
	return nil
}

func (self *SKVMRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	_, err := storage.GetMasterHost()
	if err != nil {
		return errors.Wrapf(err, "storage.GetMasterHost")
	}
	return models.GetStorageDriver(storage.StorageType).ValidateCreateSnapshotData(ctx, userCred, disk, input)
}

func (self *SKVMRegionDriver) RequestCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	storage := snapshot.GetStorage()
	if storage == nil {
		return httperrors.NewInternalServerError("Kvm snapshot missing storage ??")
	}
	return models.GetStorageDriver(storage.StorageType).RequestCreateSnapshot(ctx, snapshot, task)
}

func (self *SKVMRegionDriver) RequestCreateInstanceSnapshot(ctx context.Context, guest *models.SGuest, isp *models.SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error {
	disks, _ := guest.GetGuestDisks()
	diskIndexI64, err := params.Int("disk_index")
	if err != nil {
		return errors.Wrap(err, "get 'disk_index' from params")
	}
	diskIndex := int(diskIndexI64)
	if diskIndex >= len(disks) {
		task.SetStage("OnInstanceSnapshot", nil)
		if isp.WithMemory {
			// request do memory snapshot
			host, err := guest.GetHost()
			if err != nil {
				return err
			}
			header := task.GetTaskRequestHeader()
			url := fmt.Sprintf("%s/servers/%s/memory-snapshot", host.ManagerUri, guest.GetId())
			if _, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, jsonutils.Marshal(&hostapi.GuestMemorySnapshotRequest{
				InstanceSnapshotId: isp.GetId(),
			}), false); err != nil {
				return err
			}
		} else {
			taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
				return nil, nil
			})
		}
		return nil
	}

	snapshot, err := func() (*models.SSnapshot, error) {
		lockman.LockClass(ctx, models.SnapshotManager, "name")
		defer lockman.ReleaseClass(ctx, models.SnapshotManager, "name")

		snapshotName, err := db.GenerateName(ctx, models.SnapshotManager, task.GetUserCred(),
			fmt.Sprintf("%s-%s", isp.Name, randutil.String(8)))
		if err != nil {
			return nil, errors.Wrap(err, "Generate snapshot name")
		}

		return models.SnapshotManager.CreateSnapshot(
			ctx, task.GetUserCred(), api.SNAPSHOT_MANUAL, disks[diskIndex].DiskId,
			guest.Id, "", snapshotName, -1, false, "")
	}()
	if err != nil {
		return err
	}

	err = isp.InheritTo(ctx, task.GetUserCred(), snapshot)
	if err != nil {
		return errors.Wrapf(err, "unable to inherit from instance snapshot %s to snapshot %s", isp.GetId(), snapshot.GetId())
	}

	err = models.InstanceSnapshotJointManager.CreateJoint(ctx, isp.Id, snapshot.Id, int8(diskIndex))
	if err != nil {
		return err
	}

	params = jsonutils.NewDict()
	params.Set("disk_index", jsonutils.NewInt(int64(diskIndex)))
	params.Set(strconv.Itoa(diskIndex), jsonutils.NewString(snapshot.Id))
	task.SetStage("OnKvmDiskSnapshot", params)

	if err := snapshot.StartSnapshotCreateTask(ctx, task.GetUserCred(), nil, task.GetTaskId()); err != nil {
		return err
	}
	return nil
}

func (self *SKVMRegionDriver) SnapshotIsOutOfChain(disk *models.SDisk) bool {
	storage, _ := disk.GetStorage()
	return models.GetStorageDriver(storage.StorageType).SnapshotIsOutOfChain(disk)
}

func (self *SKVMRegionDriver) GetDiskResetParams(snapshot *models.SSnapshot) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshot.Id))
	params.Set("out_of_chain", jsonutils.NewBool(snapshot.OutOfChain))
	params.Set("location", jsonutils.NewString(snapshot.Location))
	if len(snapshot.BackingDiskId) > 0 {
		params.Set("backing_disk_id", jsonutils.NewString(snapshot.BackingDiskId))
	}
	return params
}

func (self *SKVMRegionDriver) OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, data jsonutils.JSONObject) error {
	if disk.DiskSize != snapshot.Size {
		_, err := db.Update(disk, func() error {
			disk.DiskSize = snapshot.Size
			return nil
		})
		if err != nil {
			return err
		}
	}
	storage, _ := disk.GetStorage()
	return models.GetStorageDriver(storage.StorageType).OnDiskReset(ctx, userCred, disk, snapshot, data)
}

func (self *SKVMRegionDriver) OnSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask, data jsonutils.JSONObject) error {
	task.SetStage("OnKvmSnapshotDelete", nil)
	task.ScheduleRun(data)
	return nil
}

func (self *SKVMRegionDriver) RequestSyncDiskStatus(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		storage, err := disk.GetStorage()
		if err != nil {
			return nil, errors.Wrapf(err, "disk.GetStorage")
		}
		host, err := storage.GetMasterHost()
		if err != nil {
			return nil, errors.Wrapf(err, "storage.GetMasterHost")
		}
		header := task.GetTaskRequestHeader()
		url := fmt.Sprintf("%s/disks/%s/%s/status", host.ManagerUri, storage.Id, disk.Id)
		_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", url, header, nil, false)
		if err != nil {
			return nil, err
		}
		var diskStatus string
		originStatus, _ := task.GetParams().GetString("origin_status")
		status, _ := res.GetString("status")
		if status == api.DISK_EXIST {
			if sets.NewString(api.DISK_UNKNOWN, api.DISK_REBUILD_FAILED).Has(originStatus) {
				diskStatus = api.DISK_READY
			} else {
				diskStatus = originStatus
			}
		} else {
			diskStatus = api.DISK_UNKNOWN
		}
		return nil, disk.SetStatus(ctx, userCred, diskStatus, "sync status")
	})
	return nil
}

func (self *SKVMRegionDriver) RequestCreateInstanceBackup(ctx context.Context, guest *models.SGuest, ib *models.SInstanceBackup, task taskman.ITask, params *jsonutils.JSONDict) error {
	disks, _ := guest.GetGuestDisks()
	task.SetStage("OnKvmDisksSnapshot", params)
	for i := range disks {
		disk := disks[i]
		backup, err := func() (*models.SDiskBackup, error) {
			lockman.LockClass(ctx, models.DiskBackupManager, "name")
			defer lockman.ReleaseClass(ctx, models.DiskBackupManager, "name")

			diskBackupName, err := db.GenerateName(ctx, models.DiskBackupManager, task.GetUserCred(),
				fmt.Sprintf("%s-%s", ib.Name, randutil.String(8)))
			if err != nil {
				return nil, errors.Wrap(err, "Generate diskbackup name")
			}

			return models.DiskBackupManager.CreateBackup(ctx, task.GetUserCred(), disk.DiskId, ib.BackupStorageId, diskBackupName)
		}()
		if err != nil {
			return err
		}
		err = ib.InheritTo(ctx, task.GetUserCred(), backup)
		if err != nil {
			return errors.Wrapf(err, "unable to inherit from instance backup %s to backup %s", ib.GetId(), backup.GetId())
		}
		err = models.InstanceBackupJointManager.CreateJoint(ctx, ib.Id, backup.Id, int8(i))
		if err != nil {
			return err
		}
		taskParams := jsonutils.NewDict()
		taskParams.Set("only_snapshot", jsonutils.JSONTrue)
		if err := backup.StartBackupCreateTask(ctx, task.GetUserCred(), taskParams, task.GetTaskId()); err != nil {
			return err
		}
	}
	return nil
}

func (self *SKVMRegionDriver) RequestPackInstanceBackup(ctx context.Context, ib *models.SInstanceBackup, task taskman.ITask, packageName string) error {
	backupStorage, err := ib.GetBackupStorage()
	if err != nil {
		return errors.Wrap(err, "unable to get backupStorage")
	}
	backups, err := ib.GetBackups()
	if err != nil {
		return errors.Wrap(err, "unable to get backups")
	}
	host, err := models.HostManager.GetEnabledKvmHostForDiskBackup(&backups[0])
	if err != nil {
		return errors.Wrap(err, "GetEnabledKvmHostForDiskBackup")
	}

	backupIds := make([]string, len(backups))
	for i := range backupIds {
		backupIds[i] = backups[i].GetId()
	}
	metadata, err := ib.PackMetadata(ctx, task.GetUserCred())
	if err != nil {
		return errors.Wrap(err, "unable to PackMetadata")
	}
	url := fmt.Sprintf("%s/storages/pack-instance-backup", host.ManagerUri)
	body := jsonutils.NewDict()
	body.Set("package_name", jsonutils.NewString(packageName))
	body.Set("backup_storage_id", jsonutils.NewString(backupStorage.GetId()))
	accessInfo, err := backupStorage.GetAccessInfo()
	if err != nil {
		return errors.Wrap(err, "GetAccessInfo")
	}
	body.Set("backup_storage_access_info", jsonutils.Marshal(accessInfo))
	body.Set("backup_ids", jsonutils.Marshal(backupIds))
	body.Set("metadata", jsonutils.Marshal(metadata))
	header := task.GetTaskRequestHeader()
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "unable to pack instancebackup")
	}
	return nil
}

func (self *SKVMRegionDriver) RequestUnpackInstanceBackup(ctx context.Context, ib *models.SInstanceBackup, task taskman.ITask, packageName string, metadataOnly bool) error {
	log.Infof("RequestUnpackInstanceBackup")
	backupStorage, err := ib.GetBackupStorage()
	if err != nil {
		return errors.Wrap(err, "unable to get backupStorage")
	}
	host, err := models.HostManager.GetEnabledKvmHostForBackupStorage(backupStorage)
	if err != nil {
		return errors.Wrap(err, "unable to GetEnabledKvmHost")
	}
	url := fmt.Sprintf("%s/storages/unpack-instance-backup", host.ManagerUri)
	log.Infof("url: %s", url)
	body := jsonutils.NewDict()
	body.Set("package_name", jsonutils.NewString(packageName))
	body.Set("backup_storage_id", jsonutils.NewString(backupStorage.GetId()))
	accessInfo, err := backupStorage.GetAccessInfo()
	if err != nil {
		return errors.Wrap(err, "GetAccessInfo")
	}
	body.Set("backup_storage_access_info", jsonutils.Marshal(accessInfo))
	if metadataOnly {
		body.Set("metadata_only", jsonutils.JSONTrue)
	}
	header := task.GetTaskRequestHeader()
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "unable to pack instancebackup")
	}
	return nil
}

func (self *SKVMRegionDriver) RequestSyncBackupStorageStatus(ctx context.Context, userCred mcclient.TokenCredential, bs *models.SBackupStorage, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		host, err := models.HostManager.GetEnabledKvmHostForBackupStorage(bs)
		if err != nil {
			return nil, errors.Wrap(err, "GetEnabledKvmHostForBackupStorage")
		}
		url := fmt.Sprintf("%s/storages/sync-backup-storage", host.ManagerUri)
		body := jsonutils.NewDict()
		body.Set("backup_storage_id", jsonutils.NewString(bs.GetId()))
		accessInfo, err := bs.GetAccessInfo()
		if err != nil {
			return nil, errors.Wrap(err, "GetAccessInfo")
		}
		body.Set("backup_storage_access_info", jsonutils.Marshal(accessInfo))
		header := task.GetTaskRequestHeader()
		_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
		if err != nil {
			return nil, err
		}
		status, _ := res.GetString("status")
		reason, _ := res.GetString("reason")
		return nil, bs.SetStatus(ctx, userCred, status, reason)
	})
	return nil
}

func (self *SKVMRegionDriver) RequestSyncInstanceBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, ib *models.SInstanceBackup, task taskman.ITask) error {
	originStatus, _ := task.GetParams().GetString("origin_status")
	if utils.IsInStringArray(originStatus, []string{
		api.INSTANCE_BACKUP_STATUS_CREATING,
		api.INSTANCE_BACKUP_STATUS_DELETING,
		// api.INSTANCE_BACKUP_STATUS_RECOVERY,
		api.INSTANCE_BACKUP_STATUS_PACK,
		api.INSTANCE_BACKUP_STATUS_CREATING_FROM_PACKAGE,
		api.INSTANCE_BACKUP_STATUS_SAVING,
		api.INSTANCE_BACKUP_STATUS_SNAPSHOT,
	}) {
		err := ib.SetStatus(ctx, userCred, originStatus, "sync status")
		if err != nil {
			return err
		}
		task.SetStageComplete(ctx, nil)
		return nil
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		task.SetStage("OnKvmBackupSyncstatus", nil)
		backups, err := ib.GetBackups()
		if err != nil {
			return nil, errors.Wrap(err, "unable to get backups")
		}
		for i := range backups {
			params := jsonutils.NewDict()
			params.Add(jsonutils.NewString(backups[i].GetStatus()), "origin_status")
			task, err := taskman.TaskManager.NewTask(ctx, "DiskBackupSyncstatusTask", &backups[i], userCred, params, task.GetTaskId(), "", nil)
			if err != nil {
				return nil, err
			}
			task.ScheduleRun(nil)
		}
		return nil, nil
	})
	return nil
}

func (self *SKVMRegionDriver) RequestSyncDiskBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, backup *models.SDiskBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		originStatus, _ := task.GetParams().GetString("origin_status")
		if utils.IsInStringArray(originStatus, []string{api.BACKUP_STATUS_CREATING, api.BACKUP_STATUS_SNAPSHOT, api.BACKUP_STATUS_SAVING, api.BACKUP_STATUS_CLEANUP_SNAPSHOT, api.BACKUP_STATUS_DELETING}) {
			return nil, backup.SetStatus(ctx, userCred, originStatus, "sync status")
		}
		backupStorage, err := backup.GetBackupStorage()
		if err != nil {
			return nil, errors.Wrap(err, "unable to get backupStorage")
		}
		/*storage, _ := backup.GetStorage()
		var host *models.SHost
		if storage != nil {
			host, _ = storage.GetMasterHost()
		}
		if host == nil {
			host, err = models.HostManager.GetEnabledKvmHost()
			if err != nil {
				return nil, errors.Wrap(err, "unable to GetEnabledKvmHost")
			}
		}*/
		host, err := models.HostManager.GetEnabledKvmHostForDiskBackup(backup)
		if err != nil {
			return nil, errors.Wrap(err, "GetEnabledKvmHostForDiskBackup")
		}
		log.Infof("host: %s, ManagerUri: %s", host.GetId(), host.ManagerUri)
		url := fmt.Sprintf("%s/storages/sync-backup", host.ManagerUri)
		body := jsonutils.NewDict()
		body.Set("backup_id", jsonutils.NewString(backup.GetId()))
		body.Set("backup_storage_id", jsonutils.NewString(backupStorage.GetId()))
		accessInfo, err := backupStorage.GetAccessInfo()
		if err != nil {
			return nil, errors.Wrap(err, "GetAccessInfo")
		}
		body.Set("backup_storage_access_info", jsonutils.Marshal(accessInfo))
		header := task.GetTaskRequestHeader()
		_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
		if err != nil {
			return nil, err
		}
		var backupStatus string
		status, _ := res.GetString("status")
		if status == api.BACKUP_EXIST {
			backupStatus = api.BACKUP_STATUS_READY
		} else {
			backupStatus = api.BACKUP_STATUS_UNKNOWN
		}
		return nil, backup.SetStatus(ctx, userCred, backupStatus, "sync status")
	})
	return nil
}

func (self *SKVMRegionDriver) RequestSyncSnapshotStatus(ctx context.Context, userCred mcclient.TokenCredential, snapshot *models.SSnapshot, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		storage := snapshot.GetStorage()
		host, err := storage.GetMasterHost()
		if err != nil {
			return nil, errors.Wrapf(err, "storage.GetMasterHost")
		}
		header := task.GetTaskRequestHeader()
		url := fmt.Sprintf("%s/snapshots/%s/%s/%s/status", host.ManagerUri, storage.Id, snapshot.DiskId, snapshot.Id)
		_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", url, header, nil, false)
		if err != nil {
			return nil, err
		}
		var snapshotStatus string
		originStatus, _ := task.GetParams().GetString("origin_status")
		status, _ := res.GetString("status")
		if status == api.SNAPSHOT_EXIST {
			if originStatus == api.SNAPSHOT_UNKNOWN {
				snapshotStatus = api.SNAPSHOT_READY
			} else {
				snapshotStatus = originStatus
			}
		} else {
			snapshotStatus = api.SNAPSHOT_UNKNOWN
		}
		return nil, snapshot.SetStatus(ctx, userCred, snapshotStatus, "sync status")
	})
	return nil
}

func (self *SKVMRegionDriver) RequestAssociateEipForNAT(ctx context.Context, userCred mcclient.TokenCredential, nat *models.SNatGateway, eip *models.SElasticip, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotSupported, "RequestAssociateEipForNAT")
}

func (self *SKVMRegionDriver) ValidateCacheSecgroup(ctx context.Context, userCred mcclient.TokenCredential, secgroup *models.SSecurityGroup, vpc *models.SVpc, classic bool) error {
	return errors.Wrap(httperrors.ErrNotSupported, "No need to cache secgroup for onecloud region")
}

func (self *SKVMRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ElasticcacheCreateInput) (*api.ElasticcacheCreateInput, error) {
	return input, httperrors.NewNotSupportedError("Not support create elasticcache")
}

func (self *SKVMRegionDriver) RequestRestartElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestSyncElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestChangeElasticcacheSpec(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestSetElasticcacheMaintainTime(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheChangeSpec(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestUpdateElasticcacheSecgroups(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheSetMaintainTime(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheAllocatePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheReleasePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheFlushInstance(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheUpdateInstanceParameters(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheUpdateBackupPolicy(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) ValidateCreateElasticcacheAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, nil
}

func (self *SKVMRegionDriver) ValidateCreateElasticcacheAclData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, nil
}

func (self *SKVMRegionDriver) AllowCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *models.SElasticcache) error {
	return fmt.Errorf("not support create kvm elastic cache backup")
}

func (self *SKVMRegionDriver) ValidateCreateElasticcacheBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, nil
}

func (self *SKVMRegionDriver) RequestCreateElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheAccount *models.SElasticcacheAccount, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestCreateElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheAcl *models.SElasticcacheAcl, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheBackup *models.SElasticcacheBackup, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAcl, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, eb *models.SElasticcacheBackup, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheAccountResetPassword(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheAclUpdate(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAcl, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) RequestElasticcacheBackupRestoreInstance(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheBackup, task taskman.ITask) error {
	return nil
}

func (self *SKVMRegionDriver) AllowUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *models.SElasticcache) error {
	return fmt.Errorf("not support update kvm elastic cache auth_mode")
}

func (self *SKVMRegionDriver) RequestSyncBucketStatus(ctx context.Context, userCred mcclient.TokenCredential, bucket *models.SBucket, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iBucket, err := bucket.GetIBucket(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "bucket.GetIBucket")
		}

		return nil, bucket.SetStatus(ctx, userCred, iBucket.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SKVMRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return false
}

func (self *SKVMRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 0
}

func (self *SKVMRegionDriver) RequestDeleteBackup(ctx context.Context, backup *models.SDiskBackup, task taskman.ITask) error {
	backupStorage, err := backup.GetBackupStorage()
	if err != nil {
		return errors.Wrap(err, "unable to get backupStorage")
	}
	host, err := models.HostManager.GetEnabledKvmHostForDiskBackup(backup)
	if err != nil {
		return errors.Wrap(err, "GetEnabledKvmHostForDiskBackup")
	}

	url := fmt.Sprintf("%s/storages/delete-backup", host.ManagerUri)
	body := jsonutils.NewDict()
	body.Set("backup_id", jsonutils.NewString(backup.GetId()))
	body.Set("backup_storage_id", jsonutils.NewString(backupStorage.GetId()))
	accessInfo, err := backupStorage.GetAccessInfo()
	if err != nil {
		return errors.Wrap(err, "GetAccessInfo")
	}
	body.Set("backup_storage_access_info", jsonutils.Marshal(accessInfo))
	header := task.GetTaskRequestHeader()
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "unable to backup")
	}
	return nil
}

func (self *SKVMRegionDriver) RequestCreateBackup(ctx context.Context, backup *models.SDiskBackup, snapshotId string, task taskman.ITask) error {
	backupStorage, err := backup.GetBackupStorage()
	if err != nil {
		return errors.Wrap(err, "unable to get backupStorage")
	}
	disk, err := backup.GetDisk()
	if err != nil {
		return errors.Wrap(err, "unable to get disk")
	}
	guest := disk.GetGuest()
	if guest == nil {
		return errors.Wrap(err, "unable to get guest")
	}
	storage, err := disk.GetStorage()
	if err != nil {
		return errors.Wrap(err, "unable to get storage")
	}
	snapshotObj, err := models.SnapshotManager.FetchById(snapshotId)
	if err != nil {
		return errors.Wrap(err, "fetch snapshot")
	}
	snapshot := snapshotObj.(*models.SSnapshot)
	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/disks/%s/backup/%s", host.ManagerUri, storage.Id, disk.Id)
	body := jsonutils.NewDict()
	body.Set("snapshot_id", jsonutils.NewString(snapshotId))
	if snapshot.Location != "" {
		body.Set("snapshot_location", jsonutils.NewString(snapshot.Location))
	}
	body.Set("backup_id", jsonutils.NewString(backup.GetId()))
	body.Set("backup_storage_id", jsonutils.NewString(backupStorage.GetId()))
	accessInfo, err := backupStorage.GetAccessInfo()
	if err != nil {
		return errors.Wrap(err, "GetAccessInfo")
	}
	body.Set("backup_storage_access_info", jsonutils.Marshal(accessInfo))
	if len(backup.EncryptKeyId) > 0 {
		body.Set("encrypt_key_id", jsonutils.NewString(backup.EncryptKeyId))
	}
	header := task.GetTaskRequestHeader()
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "unable to backup")
	}
	return nil
}

func (self *SKVMRegionDriver) RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip, input api.ElasticipAssociateInput, obj db.IStatusStandaloneModel, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if err := eip.AssociateInstance(ctx, userCred, input.InstanceType, obj); err != nil {
			return nil, errors.Wrapf(err, "associate eip %s(%s) to %s %s(%s)", eip.Name, eip.Id, obj.Keyword(), obj.GetName(), obj.GetId())
		}
		switch input.InstanceType {
		case api.EIP_ASSOCIATE_TYPE_SERVER:
			err := self.requestAssociateEipWithServer(ctx, userCred, eip, input, obj, task)
			if err != nil {
				return nil, err
			}
		case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
			err := self.requestAssociateEipWithInstanceGroup(ctx, userCred, eip, input, obj, task)
			if err != nil {
				return nil, err
			}
		case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
			err := self.requestAssociateEipWithLoadbalancer(ctx, userCred, eip, input, obj, task)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "instance type %s", input.InstanceType)
		}
		if err := eip.SetStatus(ctx, userCred, api.EIP_STATUS_READY, api.EIP_STATUS_ASSOCIATE); err != nil {
			return nil, errors.Wrapf(err, "set eip status to %s", api.EIP_STATUS_READY)
		}
		return nil, nil
	})
	return nil
}

func (self *SKVMRegionDriver) requestAssociateEipWithServer(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip, input api.ElasticipAssociateInput, obj db.IStatusStandaloneModel, task taskman.ITask) error {
	guest := obj.(*models.SGuest)

	if guest.GetHypervisor() != api.HYPERVISOR_KVM {
		return errors.Wrapf(cloudprovider.ErrNotSupported, "not support associate eip for hypervisor %s", guest.GetHypervisor())
	}

	lockman.LockObject(ctx, guest)
	defer lockman.ReleaseObject(ctx, guest)

	var guestnics []models.SGuestnetwork
	{
		netq := models.NetworkManager.Query().SubQuery()
		wirq := models.WireManager.Query().SubQuery()
		vpcq := models.VpcManager.Query().SubQuery()
		gneq := models.GuestnetworkManager.Query()
		q := gneq.Equals("guest_id", guest.Id).
			IsNullOrEmpty("eip_id")
		if len(input.IpAddr) > 0 {
			q = q.Equals("ip_addr", input.IpAddr)
		}
		q = q.Join(netq, sqlchemy.Equals(netq.Field("id"), gneq.Field("network_id")))
		q = q.Join(wirq, sqlchemy.Equals(wirq.Field("id"), netq.Field("wire_id")))
		q = q.Join(vpcq, sqlchemy.Equals(vpcq.Field("id"), wirq.Field("vpc_id")))
		q = q.Filter(sqlchemy.NotEquals(vpcq.Field("id"), api.DEFAULT_VPC_ID))
		if err := db.FetchModelObjects(models.GuestnetworkManager, q, &guestnics); err != nil {
			return errors.Wrapf(err, "db.FetchModelObjects")
		}
		if len(guestnics) == 0 {
			return errors.Errorf("guest has no nics to associate eip")
		}
	}

	guestnic := &guestnics[0]
	lockman.LockObject(ctx, guestnic)
	defer lockman.ReleaseObject(ctx, guestnic)
	if _, err := db.Update(guestnic, func() error {
		guestnic.EipId = eip.Id
		return nil
	}); err != nil {
		return errors.Wrapf(err, "set associated eip for guestnic %s (guest:%s, network:%s)",
			guestnic.Ifname, guestnic.GuestId, guestnic.NetworkId)
	}
	return nil
}

func (self *SKVMRegionDriver) requestAssociateEipWithInstanceGroup(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip, input api.ElasticipAssociateInput, obj db.IStatusStandaloneModel, task taskman.ITask) error {
	group := obj.(*models.SGroup)

	lockman.LockObject(ctx, group)
	defer lockman.ReleaseObject(ctx, group)

	var groupnics []models.SGroupnetwork
	{
		gneq := models.GroupnetworkManager.Query()
		q := gneq.Equals("group_id", group.Id).
			IsNullOrEmpty("eip_id")
		if len(input.IpAddr) > 0 {
			q = q.Equals("ip_addr", input.IpAddr)
		}
		if err := db.FetchModelObjects(models.GroupnetworkManager, q, &groupnics); err != nil {
			return errors.Wrapf(err, "db.FetchModelObjects")
		}
		if len(groupnics) == 0 {
			return errors.Errorf("instance group has no nics to associate eip")
		}
	}

	groupnic := &groupnics[0]
	lockman.LockObject(ctx, groupnic)
	defer lockman.ReleaseObject(ctx, groupnic)
	if _, err := db.Update(groupnic, func() error {
		groupnic.EipId = eip.Id
		return nil
	}); err != nil {
		return errors.Wrapf(err, "set associated eip for groupnic %s/%s (guest:%s, network:%s)",
			groupnic.IpAddr, groupnic.Ip6Addr, groupnic.GroupId, groupnic.NetworkId)
	}
	return nil
}

func (self *SKVMRegionDriver) requestAssociateEipWithLoadbalancer(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	eip *models.SElasticip,
	input api.ElasticipAssociateInput,
	obj db.IStatusStandaloneModel,
	task taskman.ITask,
) error {
	lb := obj.(*models.SLoadbalancer)

	if _, err := db.Update(lb, func() error {
		lb.Address = eip.IpAddr
		lb.AddressType = api.LB_ADDR_TYPE_INTERNET
		return nil
	}); err != nil {
		return errors.Wrap(err, "set loadbalancer address")
	}

	if err := eip.AssociateLoadbalancer(ctx, userCred, lb); err != nil {
		return errors.Wrapf(err, "associate eip %s(%s) to loadbalancer %s(%s)", eip.Name, eip.Id, lb.Name, lb.Id)
	}
	if err := eip.SetStatus(ctx, userCred, api.EIP_STATUS_READY, api.EIP_STATUS_ASSOCIATE); err != nil {
		return errors.Wrapf(err, "set eip status to %s", api.EIP_STATUS_ALLOCATE)
	}
	return nil
}

func (self *SKVMRegionDriver) RequestCreateSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	secgroup *models.SSecurityGroup,
	rules api.SSecgroupRuleResourceSet,
) error {
	_, err := db.Update(secgroup, func() error {
		secgroup.VpcId = ""
		return nil
	})
	if err != nil {
		return err
	}
	for _, r := range rules {
		rule := &models.SSecurityGroupRule{
			Priority:    int(*r.Priority),
			Protocol:    r.Protocol,
			Ports:       r.Ports,
			Direction:   r.Direction,
			CIDR:        r.CIDR,
			Action:      r.Action,
			Description: r.Description,
		}
		rule.SecgroupId = secgroup.Id
		models.SecurityGroupRuleManager.TableSpec().Insert(ctx, rule)
	}
	secgroup.SetStatus(ctx, userCred, api.SECGROUP_STATUS_READY, "")
	return nil
}

func (self *SKVMRegionDriver) RequestPrepareSecurityGroups(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, secgroups []models.SSecurityGroup, vpc *models.SVpc, callback func(ids []string) error, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestDeleteSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, secgroup *models.SSecurityGroup, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("cloudregion_id", api.DEFAULT_REGION_ID)
	}, nil
}

func (self *SKVMRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Priority != nil {
		if *input.Priority < 1 || *input.Priority > 100 {
			return nil, httperrors.NewInputParameterError("invalid priority %d", input.Priority)
		}
	}
	if input.Action != nil {
		if !utils.IsInStringArray(*input.Action, []string{string(secrules.SecurityRuleAllow), string(secrules.SecurityRuleDeny)}) {
			return nil, httperrors.NewInputParameterError("invalid action %s", *input.Action)
		}
	}
	if input.Protocol != nil {
		if !utils.IsInStringArray(*input.Protocol, []string{
			secrules.PROTO_ANY,
			secrules.PROTO_UDP,
			secrules.PROTO_TCP,
			secrules.PROTO_ICMP,
		}) {
			return nil, httperrors.NewInputParameterError("invalid protocol %s", *input.Protocol)
		}
	}

	if input.Ports != nil {
		rule := secrules.SecurityRule{}
		err := rule.ParsePorts(*input.Ports)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid ports %s", *input.Ports)
		}
	}

	if input.CIDR != nil && len(*input.CIDR) > 0 && !regutils.MatchCIDR(*input.CIDR) && !regutils.MatchIP4Addr(*input.CIDR) && !regutils.MatchCIDR6(*input.CIDR) && !regutils.MatchIP6Addr(*input.CIDR) {
		return nil, httperrors.NewInputParameterError("invalid cidr %s", *input.CIDR)
	}

	return input, nil
}

func (self *SKVMRegionDriver) ValidateCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, input *api.SSnapshotPolicyCreateInput) (*api.SSnapshotPolicyCreateInput, error) {
	return input, nil
}

func (self *SKVMRegionDriver) RequestCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, sp *models.SSnapshotPolicy, task taskman.ITask) error {
	sp.SetStatus(ctx, userCred, apis.STATUS_AVAILABLE, "")
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestDeleteSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, sp *models.SSnapshotPolicy, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SKVMRegionDriver) RequestSnapshotPolicyBindDisks(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, diskIds []string, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		disks, err := sp.GetUnbindDisks(diskIds)
		if err != nil {
			return nil, errors.Wrapf(err, "GetUnbindDisks")
		}
		ids := []string{}
		for _, disk := range disks {
			ids = append(ids, disk.Id)
		}
		return nil, sp.BindDisks(ctx, disks)
	})
	return nil
}

func (self *SKVMRegionDriver) RequestSnapshotPolicyUnbindDisks(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, diskIds []string, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return nil, sp.UnbindDisks(diskIds)
	})
	return nil
}
