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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

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
		if utils.IsInStringArray(listenerType, []string{api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP}) {
			if lbbg == nil {
				return nil, httperrors.NewMissingParameterError("backend_group_id")
			}

			// listener check
			q := models.LoadbalancerListenerManager.Query()
			q = q.Equals("loadbalancer_id", lb.GetId())
			q = q.Equals("listener_type", listenerType)
			q = q.Equals("backend_group_id", lbbg.GetId())
			q = q.IsFalse("pending_deleted")
			count, err := q.CountWithError()
			if err != nil {
				return nil, httperrors.NewGeneralError(err)
			}

			if count > 0 {
				return nil, httperrors.NewConflictError("loadbalancer backendgroup aready associate with other %s listener", listenerType)
			}

			// lbbg backend check
			lbbs, err := lbbg.GetBackends()
			if err != nil {
				return nil, httperrors.NewGeneralError(err)
			}

			for i := range lbbs {
				err = checkQcloudBackendGroupUsable("", listenerType, lbbs[i].BackendId, lbbs[i].Port)
				if err != nil {
					return nil, err
				}
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
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackendGroup.GetIRegion")
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
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackendGroup.GetILoadBalancerById")
		}
		group := &cloudprovider.SLoadbalancerBackendGroup{
			Name:      lbbg.Name,
			GroupType: lbbg.Type,
			Backends:  backends,
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.CreateILoadBalancerBackendGroup(group)
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackendGroup.CreateILoadBalancerBackendGroup")
		}
		if err := db.SetExternalId(lbbg, userCred, iLoadbalancerBackendGroup.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackendGroup.GetGlobalId")
		}
		iBackends, err := iLoadbalancerBackendGroup.GetILoadbalancerBackends()
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackendGroup.GetILoadbalancerBackends")
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
		lb := lbbg.GetLoadbalancer()
		if lb == nil {
			return nil, fmt.Errorf("failed to find lb for backendgroup %s", lbbg.Name)
		}

		cachedlbbgs, err := models.QcloudCachedLbbgManager.GetCachedBackendGroups(lbbg.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackend.GetCachedBackendGroups")
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
					continue
				}

				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackend.GetICloudLoadbalancerBackendGroup")
			}

			ibackend, err = iLoadbalancerBackendGroup.AddBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackend.AddBackendServer")
			}

			_, err = models.QcloudCachedLbManager.CreateQcloudCachedLb(ctx, userCred, lbb, &cachedLbbg, ibackend, cachedLbbg.GetOwnerId())
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackend.CreateQcloudCachedLb")
			}
		}

		if ibackend != nil {
			if err := lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, ibackend, nil); err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerBackend.SyncWithCloudLoadbalancerBackend")
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SQcloudRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}

		cachedlbbs, err := models.QcloudCachedLbManager.GetBackendsByLocalBackendId(lbb.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestDeleteLoadbalancerBackend.GetBackendsByLocalBackendId")
		}

		guest := lbb.GetGuest()
		if guest == nil {
			return nil, fmt.Errorf("loadbalancer backend %s related server not found", lbb.GetName())
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
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestDeleteLoadbalancerBackend.GetIRegion")
			}
			iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestDeleteLoadbalancerBackend.GetILoadBalancerById")
			}
			iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedlbbg.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestDeleteLoadbalancerBackend.GetILoadBalancerBackendGroupById")
			}

			err = iLoadbalancerBackendGroup.RemoveBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestDeleteLoadbalancerBackend.RemoveBackendServer")
			}

			err = db.DeleteModel(ctx, userCred, &cachedlbb)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestDeleteLoadbalancerBackend.DeleteModel")
			}
		}

		return nil, nil
	})
	return nil
}

func (self *SQcloudRegionDriver) createCachedLbbg(lb *models.SLoadbalancer, lblis *models.SLoadbalancerListener, lbr *models.SLoadbalancerListenerRule, lbbg *models.SLoadbalancerBackendGroup) (*models.SQcloudCachedLbbg, error) {
	// create loadbalancer backendgroup cache
	cachedLbbg := &models.SQcloudCachedLbbg{}
	cachedLbbg.ManagerId = lb.ManagerId
	cachedLbbg.CloudregionId = lb.CloudregionId
	cachedLbbg.LoadbalancerId = lb.GetId()
	cachedLbbg.BackendGroupId = lbbg.GetId()
	if lbr != nil {
		cachedLbbg.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
		cachedLbbg.AssociatedId = lbr.GetId()
	} else {
		cachedLbbg.AssociatedType = api.LB_ASSOCIATE_TYPE_LISTENER
		cachedLbbg.AssociatedId = lblis.GetId()
	}

	err := models.QcloudCachedLbbgManager.TableSpec().Insert(cachedLbbg)
	if err != nil {
		return nil, errors.Wrap(err, "SQcloudRegionDriver.createCachedLbbg.Insert")
	}

	cachedLbbg.SetModelManager(models.QcloudCachedLbbgManager, cachedLbbg)
	return cachedLbbg, nil
}

func (self *SQcloudRegionDriver) syncCloudlbbs(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, cachedLbbg *models.SQcloudCachedLbbg, extlbbg cloudprovider.ICloudLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend) error {
	ibackends, err := extlbbg.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "QcloudRegionDriver.syncCloudLoadbalancerBackends.GetILoadbalancerBackends")
	}

	for i := range ibackends {
		ibackend := ibackends[i]
		err = extlbbg.RemoveBackendServer(ibackend.GetId(), ibackend.GetWeight(), ibackend.GetPort())
		if err != nil {
			return errors.Wrap(err, "QcloudRegionDriver.syncCloudLoadbalancerBackends.RemoveBackendServer")
		}
	}

	for _, backend := range backends {
		_, err = extlbbg.AddBackendServer(backend.ExternalID, backend.Weight, backend.Port)
		if err != nil {
			return errors.Wrap(err, "QcloudRegionDriver.syncCloudLoadbalancerBackends.AddBackendServer")
		}
	}

	return nil
}

func (self *SQcloudRegionDriver) syncCachedLbbs(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, lbbg *models.SQcloudCachedLbbg, extlbbg cloudprovider.ICloudLoadbalancerBackendGroup) error {
	iBackends, err := extlbbg.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "QcloudRegionDriver.syncLoadbalancerBackendCaches.GetILoadbalancerBackends")
	}

	if len(iBackends) > 0 {
		provider := lb.GetCloudprovider()
		if provider == nil {
			return fmt.Errorf("failed to find cloudprovider for lb %s", lb.Name)
		}

		result := models.QcloudCachedLbManager.SyncLoadbalancerBackends(ctx, userCred, provider, lbbg, iBackends, &models.SSyncRange{})
		if result.IsError() {
			return errors.Wrap(result.AllError(), "QcloudRegionDriver.syncLoadbalancerBackendCaches.SyncLoadbalancerBackends")
		}
	}

	return nil
}

func (self *SQcloudRegionDriver) createLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, lbr *models.SLoadbalancerListenerRule, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend) (jsonutils.JSONObject, error) {
	iRegion, err := lbbg.GetIRegion()
	if err != nil {
		return nil, errors.Wrap(err, "SQcloudRegionDriver.createLoadbalancerBackendGroup.GetIRegion")
	}
	lb := lbbg.GetLoadbalancer()
	if lb == nil {
		return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
	}
	iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "SQcloudRegionDriver.createLoadbalancerBackendGroup.GetILoadBalancerById")
	}

	var ilbbg cloudprovider.ICloudLoadbalancerBackendGroup
	if lbr != nil {
		l := lbr.GetLoadbalancerListener()
		if l == nil {
			return nil, fmt.Errorf("could not create loadbalancer backendgroup, loadbalancer listener rule %s related listener not found", lbr.GetName())
		}

		ilblis, err := iLoadbalancer.GetILoadBalancerListenerById(l.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "SQcloudRegionDriver.createLoadbalancerBackendGroup.GetILoadBalancerListenerById")
		}

		ilbr, err := ilblis.GetILoadBalancerListenerRuleById(lbr.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "SQcloudRegionDriver.createLoadbalancerBackendGroup.GetILoadBalancerListenerRuleById")
		}

		extLbbgId := ilbr.GetBackendGroupId()
		ilbbg, err = iLoadbalancer.GetILoadBalancerBackendGroupById(extLbbgId)
		if err != nil {
			return nil, errors.Wrap(err, "SQcloudRegionDriver.createLoadbalancerBackendGroup.GetILoadBalancerBackendGroupById")
		}
	} else if lblis != nil {
		ilblis, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "SQcloudRegionDriver.createLoadbalancerBackendGroup.Listener.GetILoadBalancerListenerById")
		}

		extLbbgId := ilblis.GetBackendGroupId()
		ilbbg, err = iLoadbalancer.GetILoadBalancerBackendGroupById(extLbbgId)
		if err != nil {
			return nil, errors.Wrap(err, "SQcloudRegionDriver.createLoadbalancerBackendGroup.Listener.GetILoadBalancerBackendGroupById")
		}
	} else {
		return nil, fmt.Errorf("could not create loadbalancer backendgroup, loadbalancer listener & rule are nil")
	}

	cachedLbbg, err := self.createCachedLbbg(lb, lblis, lbr, lbbg)
	if err != nil {
		return nil, errors.Wrap(err, "QcloudRegionDriver.createLoadbalancerBackendGroupCache")
	}

	if err := db.SetExternalId(cachedLbbg, userCred, ilbbg.GetGlobalId()); err != nil {
		return nil, errors.Wrap(err, "SQcloudRegionDriver.createLoadbalancerBackendGroup.SetExternalId")
	}

	err = self.syncCloudlbbs(ctx, userCred, lb, cachedLbbg, ilbbg, backends)
	if err != nil {
		return nil, errors.Wrap(err, "QcloudRegionDriver.createLoadbalancerBackendGroup.syncCloudLoadbalancerBackends")
	}

	err = self.syncCachedLbbs(ctx, userCred, lb, cachedLbbg, ilbbg)
	if err != nil {
		return nil, errors.Wrap(err, "QcloudRegionDriver.createLoadbalancerBackendGroup.syncLoadbalancerBackendCaches")
	}

	return nil, nil

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
					return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListener.FetchById")
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListener.GetOrCreateCachedCertificate")
				}

				if len(lbcert.ExternalId) == 0 {
					err = self.RequestCreateLoadbalancerCertificate(ctx, userCred, lbcert, task)
					if err != nil {
						return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListener.RequestCreateLoadbalancerCertificate")
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

		params, err := lblis.GetQcloudLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListener.GetQcloudLoadbalancerListenerParams")
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListener.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListener.GetILoadBalancerById")
		}
		iListener, err := iLoadbalancer.CreateILoadBalancerListener(params)
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListener.CreateILoadBalancerListener")
		}
		if err := db.SetExternalId(lblis, userCred, iListener.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListener.SetExternalId")
		}

		// ====腾讯云添加后端服务器=====
		if !utils.IsInStringArray(lblis.ListenerType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) {
			lbbg := lblis.GetLoadbalancerBackendGroup()
			if lbbg == nil {
				err := fmt.Errorf("loadbalancer listener %s related backendgroup not found", lblis.GetName())
				return nil, errors.Wrap(err, "SQcloudRegionDriver.RequestCreateLoadbalancerListener.GetLoadbalancerBackendGroup")
			}

			backends, err := lbbg.GetBackendsParams()
			if err != nil {
				return nil, errors.Wrap(err, "SQcloudRegionDriver.RequestCreateLoadbalancerListener.GetBackendsParams")
			}

			_, err = self.createLoadbalancerBackendGroup(ctx, userCred, lblis, nil, lbbg, backends)
			if err != nil {
				return nil, errors.Wrap(err, "SQcloudRegionDriver.RequestCreateLoadbalancerListener.createLoadbalancerBackendGroup")
			}
		}

		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, nil)
	})
	return nil
}

func (self *SQcloudRegionDriver) GetLoadbalancerListenerRuleInputParams(lblis *models.SLoadbalancerListener, lbr *models.SLoadbalancerListenerRule) *cloudprovider.SLoadbalancerListenerRule {
	scheduler := ""
	switch lblis.Scheduler {
	case api.LB_SCHEDULER_WRR:
		scheduler = "WRR"
	case api.LB_SCHEDULER_WLC:
		scheduler = "LEAST_CONN"
	case api.LB_SCHEDULER_SCH:
		scheduler = "IP_HASH"
	default:
		scheduler = "WRR"
	}

	sessionTimeout := 0
	if lblis.StickySession == api.LB_BOOL_ON {
		sessionTimeout = lblis.StickySessionCookieTimeout
	}

	rule := &cloudprovider.SLoadbalancerListenerRule{
		Name:   lbr.Name,
		Domain: lbr.Domain,
		Path:   lbr.Path,

		Scheduler: scheduler,

		HealthCheck:         lblis.HealthCheck,
		HealthCheckType:     lblis.HealthCheckType,
		HealthCheckTimeout:  lblis.HealthCheckTimeout,
		HealthCheckDomain:   lblis.HealthCheckDomain,
		HealthCheckHttpCode: lblis.HealthCheckHttpCode,
		HealthCheckURI:      lblis.HealthCheckURI,
		HealthCheckInterval: lblis.HealthCheckInterval,

		HealthCheckRise: lblis.HealthCheckRise,
		HealthCheckFail: lblis.HealthCheckFall,

		StickySessionCookieTimeout: sessionTimeout,
	}

	return rule
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
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListenerRule.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListenerRule.GetILoadBalancerById")
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(listener.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListenerRule.GetILoadBalancerListenerById")
		}
		rule := self.GetLoadbalancerListenerRuleInputParams(listener, lbr)
		iListenerRule, err := iListener.CreateILoadBalancerListenerRule(rule)
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListenerRule.CreateILoadBalancerListenerRule")
		}
		if err := db.SetExternalId(lbr, userCred, iListenerRule.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestCreateLoadbalancerListenerRule.UpdateListenerRule")
		}
		// ====腾讯云添加后端服务器=====
		lbbg := lbr.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			err := fmt.Errorf("loadbalancer listener rule %s related backendgroup not found", lbr.GetName())
			return nil, errors.Wrap(err, "SQcloudRegionDriver.RequestCreateLoadbalancerListener.GetLoadbalancerBackendGroup")
		}

		backends, err := lbbg.GetBackendsParams()
		if err != nil {
			return nil, errors.Wrap(err, "SQcloudRegionDriver.RequestCreateLoadbalancerListener.GetBackendsParams")
		}

		_, err = self.createLoadbalancerBackendGroup(ctx, userCred, nil, lbr, lbbg, backends)
		if err != nil {
			return nil, errors.Wrap(err, "SQcloudRegionDriver.RequestCreateLoadbalancerListener.createLoadbalancerBackendGroup")
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
		} else {
			if utils.IsInStringArray(lblis.ListenerType, []string{api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP}) {
				cachedLbbgs, err := lbbg.GetQcloudCachedlbbg()
				if err != nil {
					return nil, err
				}

				if len(cachedLbbgs) > 0 {
					for i := range cachedLbbgs {
						if cachedLbbgs[i].AssociatedType == api.LB_ASSOCIATE_TYPE_LISTENER && cachedLbbgs[i].AssociatedId != lblis.GetId() {
							_lblis, err := db.FetchById(models.LoadbalancerListenerManager, cachedLbbgs[i].AssociatedId)
							if err != nil {
								return nil, err
							}

							if _lblis.(*models.SLoadbalancerListener).ListenerType == lblis.ListenerType {
								return nil, httperrors.NewConflictError("loadbalancer aready associated with fourth layer listener %s", cachedLbbgs[i].AssociatedId)
							}
						}
					}
				}

				lbbs, err := lbbg.GetBackends()
				if err != nil {
					return nil, err
				}

				for i := range lbbs {
					err = checkQcloudBackendGroupUsable(lblis.BackendGroupId, lblis.ListenerType, lbbs[i].BackendId, lbbs[i].Port)
					if err != nil {
						return nil, err
					}
				}
			}
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

	port, _ := data.Int("port")
	backendId, _ := data.GetString("backend_id")
	err := CheckQcloudBackendPortUnique(lbbg.GetId(), backendId, int(port))
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (self *SQcloudRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	domainV := validators.NewDomainNameValidator("domain")
	pathV := validators.NewURLPathValidator("path")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),
		"domain": domainV.AllowEmpty(false),
		"path":   pathV.AllowEmpty(false),

		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate").Default(0),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src").Default(0),
	}

	if err := RunValidators(keyV, data, false); err != nil {
		return nil, err
	}

	if path, _ := data.GetString("path"); len(path) == 0 {
		return nil, httperrors.NewInputParameterError("path can not be emtpy")
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
	} else {
		if lbbg == nil {
			return nil, httperrors.NewMissingParameterError("backend_group_id")
		}

		data.Set("backend_group_id", jsonutils.NewString(lbbg.GetId()))
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

// validate backend port unique
func CheckQcloudBackendPortUnique(backendGroupId string, backendServerId string, port int) error {
	q1 := models.LoadbalancerBackendManager.Query("backend_group_id").Equals("port", port).Equals("backend_id", backendServerId).IsFalse("pending_deleted").Distinct()
	count, err := q1.CountWithError()
	if err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	q2 := models.LoadbalancerBackendManager.Query().Equals("backend_group_id", backendGroupId).Equals("port", port).Equals("backend_id", backendServerId).IsFalse("pending_deleted").Distinct()
	count, err = q2.CountWithError()
	if err != nil {
		return err
	}

	if count > 0 {
		return httperrors.NewConflictError("server %s with port %d already in used", backendServerId, port)
	}

	// 检查 当前服务器组没有backend with port记录，但是其他backendgroup存在backend with port记录的情况
	q3 := models.LoadbalancerBackendGroupManager.Query().IsFalse("pending_deleted")
	subLblis := models.LoadbalancerListenerManager.Query().SubQuery()
	q3 = q3.Join(subLblis, sqlchemy.Equals(subLblis.Field("backend_group_id"), q3.Field("id")))
	q3 = q3.Equals("id", backendGroupId)
	count, err = q3.Filter(sqlchemy.Equals(subLblis.Field("listener_type"), api.LB_LISTENER_TYPE_TCP)).CountWithError()
	if err != nil {
		return err
	}

	if count > 0 {
		err = checkQcloudBackendGroupUsable(backendGroupId, api.LB_LISTENER_TYPE_TCP, backendServerId, port)
		if err != nil {
			return err
		}
	}

	count, err = q3.Filter(sqlchemy.Equals(subLblis.Field("listener_type"), api.LB_LISTENER_TYPE_UDP)).CountWithError()
	if err != nil {
		return err
	}

	if count > 0 {
		err = checkQcloudBackendGroupUsable(backendGroupId, api.LB_LISTENER_TYPE_UDP, backendServerId, port)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkQcloudBackendGroupUsable(fromBackendGroup string, listenerType string, backendServerId string, port int) error {
	q := models.QcloudCachedLbManager.Query()
	subLbb := models.LoadbalancerBackendManager.Query().SubQuery()
	subCachedLbbg := models.QcloudCachedLbbgManager.Query().SubQuery()
	subLbbg := models.LoadbalancerBackendGroupManager.Query().SubQuery()
	subLblis := models.LoadbalancerListenerManager.Query().SubQuery()
	q = q.Join(subLbb, sqlchemy.Equals(q.Field("backend_id"), subLbb.Field("id")))
	q = q.Join(subCachedLbbg, sqlchemy.Equals(q.Field("cached_backend_group_id"), subCachedLbbg.Field("id")))
	q = q.Join(subLbbg, sqlchemy.Equals(subCachedLbbg.Field("backend_group_id"), subLbbg.Field("id")))
	q = q.Join(subLblis, sqlchemy.Equals(subCachedLbbg.Field("associated_id"), subLblis.Field("id")))

	q = q.Filter(sqlchemy.Equals(subCachedLbbg.Field("associated_type"), api.LB_ASSOCIATE_TYPE_LISTENER))
	q = q.Filter(sqlchemy.Equals(subLblis.Field("listener_type"), listenerType))
	q = q.Filter(sqlchemy.IsFalse(subLblis.Field("pending_deleted")))
	if len(fromBackendGroup) > 0 {
		q = q.Filter(sqlchemy.NotEquals(subLbbg.Field("id"), fromBackendGroup))
	}
	q = q.Filter(sqlchemy.Equals(subLbb.Field("backend_id"), backendServerId))
	q = q.Filter(sqlchemy.Equals(subLbb.Field("port"), port))
	q = q.Filter(sqlchemy.IsFalse(subLbb.Field("pending_deleted")))
	count, err := q.CountWithError()
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	if count > 0 {
		return httperrors.NewConflictError("server %s with port %d aready used by other %s listener", backendServerId, port, listenerType)
	}

	return nil
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

		address, err := models.LoadbalancerBackendManager.GetGuestAddress(guest)
		if err != nil {
			return nil, errors.Wrap(err, "huaWeiRegionDriver.ValidateCreateLoadbalancerBackendData.GetGuestAddress")
		}

		port, _ := data.Int("port")
		err = CheckQcloudBackendPortUnique(backendGroup.GetId(), guest.GetId(), int(port))
		if err != nil {
			return nil, err
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

func (self *SQcloudRegionDriver) RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		cachedlbbs, err := models.QcloudCachedLbManager.GetBackendsByLocalBackendId(lbb.GetId())
		if err != nil {
			return nil, errors.Wrap(err, "qcloudRegionDriver.RequestSyncLoadbalancerBackend.GetBackendsByLocalBackendId")
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
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestSyncLoadbalancerBackend.GetIRegion")
			}
			iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestSyncLoadbalancerBackend.GetILoadBalancerById")
			}
			iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(cachedlbbg.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestSyncLoadbalancerBackend.GetILoadBalancerBackendGroupById")
			}

			iBackend, err := iLoadbalancerBackendGroup.GetILoadbalancerBackendById(cachedlbb.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestSyncLoadbalancerBackend.GetILoadbalancerBackendById")
			}

			err = iBackend.SyncConf(lbb.Port, lbb.Weight)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestSyncLoadbalancerBackend.SyncConf")
			}

			err = iBackend.Refresh()
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestSyncLoadbalancerBackend.Refresh")
			}

			err = cachedlbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iBackend, nil)
			if err != nil {
				return nil, errors.Wrap(err, "qcloudRegionDriver.RequestSyncLoadbalancerBackend.SyncWithCloudLoadbalancerBackend")
			}
		}

		return nil, nil
	})
	return nil
}

// 目前只支持应用型负载均衡
func (self *SQcloudRegionDriver) syncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup, ilbbg cloudprovider.ICloudLoadbalancerBackendGroup) error {
	if ilbbg != nil {
		ibackends, err := ilbbg.GetILoadbalancerBackends()
		if err != nil {
			return errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.GetILoadbalancerBackends")
		}

		for _, ibackend := range ibackends {
			err = ilbbg.RemoveBackendServer(ibackend.GetBackendId(), ibackend.GetWeight(), ibackend.GetPort())
			if err != nil {
				return errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.RemoveBackendServer")
			}
		}

		backends, err := lbbg.GetBackendsParams()
		if err != nil {
			return errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.GetBackendsParams")
		}

		for _, backend := range backends {
			_, err = ilbbg.AddBackendServer(backend.ExternalID, backend.Weight, backend.Port)
			if err != nil {
				return errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.AddBackendServer")
			}
		}

		_olbbg, err := db.FetchByExternalId(models.QcloudCachedLbbgManager, ilbbg.GetGlobalId())
		if err != nil {
			return errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.FetchByExternalId")
		}

		olbbg := _olbbg.(*models.SQcloudCachedLbbg)
		_, err = db.Update(olbbg, func() error {
			olbbg.ExternalId = ilbbg.GetGlobalId()
			olbbg.BackendGroupId = lbbg.GetId()
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.UpdateListenerBackendGroup")
		}

		err = self.syncCachedLbbs(ctx, userCred, lb, olbbg, ilbbg)
		if err != nil {
			return errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.syncCachedLbbs")
		}
	}

	return nil
}

func (self *SQcloudRegionDriver) RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if utils.IsInStringArray(lblis.ListenerType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) {
			return nil, nil
		}

		lbbg := lblis.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			err := fmt.Errorf("failed to find lbbg for lblis %s", lblis.Name)
			return nil, errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerbackendGroup.GetLoadbalancerBackendGroup")
		}

		iRegion, err := lbbg.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerbackendGroup.GetIRegion")
		}

		lb := lbbg.GetLoadbalancer()
		ilb, err := iRegion.GetILoadBalancerById(lb.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerbackendGroup.GetILoadBalancerById")
		}

		ilisten, err := ilb.GetILoadBalancerListenerById(lblis.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerbackendGroup.GetILoadBalancerListenerById")
		}

		ilbbg, err := ilb.GetILoadBalancerBackendGroupById(ilisten.GetBackendGroupId())
		if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.GetILoadBalancerBackendGroupById")
		}

		// listener lbbg sync
		err = self.syncLoadbalancerBackendGroup(ctx, userCred, lb, lbbg, ilbbg)
		if err != nil {
			return nil, errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.syncLoadbalancerBackendGroup")
		}

		_, err = db.UpdateWithLock(ctx, lblis, func() error {
			lblis.BackendGroupId = lbbg.GetId()
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "QcloudRegionDriver.RequestSyncLoadbalancerBackendGroup.UpdateListenBackendGroupId")
		}

		return nil, nil
	})

	return nil
}

func (self *SQcloudRegionDriver) RequestPullRegionLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) error {
	return nil
}

func (self *SQcloudRegionDriver) RequestPullLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localLoadbalancer *models.SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *models.SSyncRange) error {
	meta := remoteLoadbalancer.GetMetadata()
	if meta == nil {
		return fmt.Errorf("")

	}

	// 经典型负载均衡只有一个后端服务器组，全局共享
	if forward, _ := meta.Int("Forward"); forward == 1 {
		models.SyncQcloudLoadbalancerBackendgroups(ctx, userCred, syncResults, provider, localLoadbalancer, remoteLoadbalancer, syncRange)
		return nil
	} else {
		return self.SManagedVirtualizationRegionDriver.RequestPullLoadbalancerBackendGroup(ctx, userCred, syncResults, provider, localLoadbalancer, remoteLoadbalancer, syncRange)
	}
}

func (self *SQcloudRegionDriver) RequestPreSnapshotPolicyApply(ctx context.Context, userCred mcclient.
	TokenCredential, task taskman.ITask, disk *models.SDisk, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) error {

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		if sp == nil {
			return data, nil
		}
		spcache, err := models.SnapshotPolicyCacheManager.FetchSnapshotPolicyCache(sp.GetId(),
			disk.GetStorage().GetRegion().GetId(), disk.GetStorage().ManagerId)
		if err != nil {
			return nil, err
		}
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

func (self *SQcloudRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
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
					return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.FetchCert")
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetCert")
				}

				if len(lbcert.ExternalId) == 0 {
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.CreateCert")
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedCertificateId = lbcert.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.UpdateCachedCertificateId")
				}
			}
		}

		params, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetParams")
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetILoadbalancer")
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetIListener")
		}
		if err := iListener.Sync(params); err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.SyncListener")
		}
		if err := iListener.Refresh(); err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.RefreshListener")
		}

		if utils.IsInStringArray(lblis.ListenerType, []string{api.LB_LISTENER_TYPE_UDP, api.LB_LISTENER_TYPE_TCP}) {
			return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, nil)
		} else {
			// http&https listener 变更不会同步到监听规则
			return nil, nil
		}
	})
	return nil
}
