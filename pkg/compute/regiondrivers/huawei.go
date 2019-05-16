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
	"strings"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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
	addressType, _ := data.GetString("address_type")
	if addressType == api.LB_ADDR_TYPE_INTERNET {
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
	// required：backend,backend_group,port,weight
	// be3a5b845e604decb9005e6643f688af/ports?network_id=28bf47f5-5999-45dd-9546-9f964b2fac80&tenant_id=be3a5b845e604decb9005e6643f688af&limit=2000 ,验证binding:vif_details primary_interface: true
	return data, nil
}

// func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
// 	// required: protocol,protocol_port,loadbalancer_id
// 	// others: name, description,connection_limit?,http2_enable,default_pool_id,
//
// 	return nil, nil
// }

func (self *SHuaWeiRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizationRegionDriver.ValidateCreateLoadbalancerListenerRuleData(ctx, userCred, data, backendGroup)
	if err != nil {
		return data, err
	}

	domain, _ := data.GetString("domain")
	path, _ := data.GetString("path")
	if domain == "" && path == "" {
		return data, fmt.Errorf("'domain' or 'path' should not be empty.")
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
	/*
		default_pool_id有如下限制：
		不能更新为其他监听器的default_pool。
		不能更新为其他监听器的关联的转发策略所使用的pool。
		default_pool_id对应的后端云服务器组的protocol和监听器的protocol有如下关系：
		监听器的protocol为TCP时，后端云服务器组的protocol必须为TCP。
		监听器的protocol为UDP时，后端云服务器组的protocol必须为UDP。
		监听器的protocol为HTTP或TERMINATED_HTTPS时，后端云服务器组的protocol必须为HTTP。
	*/
	mlbbg, ok := backendGroup.(*models.SLoadbalancerBackendGroup)
	if ok {
		bgProtocol := mlbbg.ProtocolType
		listenerProtocol := lblis.ListenerType
		if !strings.Contains(listenerProtocol, bgProtocol) {
			return data, fmt.Errorf("listener & backendgroup listen protocol mismatch.")
		}

		count, err := mlbbg.RefCount()
		if err != nil {
			return data, err
		}

		if count > 1 || (count == 1 && lblis.BackendGroupId != mlbbg.Id) {
			return data, fmt.Errorf("backendgroup is binding with loadbalancer/listener/listenerrule.")
		}

		if len(lblis.BackendGroupId) == 0 {
			return data, fmt.Errorf("listener is binding with other backendgroup.")
		}
	}

	return data, nil
}

func (self *SHuaWeiRegionDriver) createLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend) (jsonutils.JSONObject, error) {
	if len(lbbg.ProtocolType) == 0 {
		return nil, fmt.Errorf("loadbalancer backendgroup missing protocol type")
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

	ListenerID := ""
	listener := lbbg.GetListener()
	if listener != nil {
		ListenerID = listener.ExternalId
	}

	var stickySession *cloudprovider.SLoadbalancerStickySession
	if lbbg.StickySession == api.LB_BOOL_ON {
		stickySession = &cloudprovider.SLoadbalancerStickySession{
			StickySession:              lbbg.StickySession,
			StickySessionCookie:        lbbg.StickySessionCookie,
			StickySessionType:          lbbg.StickySessionType,
			StickySessionCookieTimeout: lbbg.StickySessionCookieTimeout,
		}
	}

	var healthCheck *cloudprovider.SLoadbalancerHealthCheck
	if lbbg.HealthCheck == api.LB_BOOL_ON {
		healthCheck = &cloudprovider.SLoadbalancerHealthCheck{
			HealthCheckType:     lbbg.HealthCheckType,
			HealthCheckReq:      lbbg.HealthCheckReq,
			HealthCheckExp:      lbbg.HealthCheckExp,
			HealthCheck:         lbbg.HealthCheck,
			HealthCheckTimeout:  lbbg.HealthCheckTimeout,
			HealthCheckDomain:   lbbg.HealthCheckDomain,
			HealthCheckHttpCode: lbbg.HealthCheckHttpCode,
			HealthCheckURI:      lbbg.HealthCheckURI,
			HealthCheckInterval: lbbg.HealthCheckInterval,
			HealthCheckRise:     lbbg.HealthCheckRise,
			HealthCheckFail:     lbbg.HealthCheckFall,
		}
	}

	lb := lbbg.GetLoadbalancer()
	group := &cloudprovider.SLoadbalancerBackendGroup{
		Name:           lbbg.Name,
		GroupType:      lbbg.Type,
		LoadbalancerID: lb.ExternalId,
		ListenerID:     ListenerID,
		ListenType:     lbbg.ProtocolType,
		Scheduler:      lbbg.Scheduler,
		StickySession:  stickySession,
		HealthCheck:    healthCheck,
		Backends:       backends, // todo: support ?
	}
	iLoadbalancerBackendGroup, err := iLoadbalancer.CreateILoadBalancerBackendGroup(group)
	if err != nil {
		return nil, err
	}

	if err := db.SetExternalId(lbbg, userCred, iLoadbalancerBackendGroup.GetGlobalId()); err != nil {
		return nil, err
	}

	for _, backend := range backends {
		ibackend, err := iLoadbalancerBackendGroup.AddBackendServer(backend.ExternalID, backend.Weight, backend.Port)
		if err != nil {
			return nil, err
		}

		b, err := db.FetchById(models.LoadbalancerBackendManager, backend.ID)
		if err != nil {
			return nil, err
		}

		err = db.SetExternalId(b.(*models.SLoadbalancerBackend), userCred, ibackend.GetGlobalId())
		if err != nil {
			return nil, err
		}
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

}

func (self *SHuaWeiRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	// 未指定后端协议类型的情况下，跳过创建步骤
	if len(lbbg.ProtocolType) == 0 {
		return nil
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.createLoadbalancerBackendGroup(ctx, userCred, lbbg, backends)
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
			p, err := lblis.GetLoadbalancerListenerParams()
			if err != nil {
				return nil, err
			}

			err = ilisten.Sync(p)
			if err != nil {
				return nil, err
			}
		}

		backends, err := lbbg.GetBackendsParams()
		if err != nil {
			return nil, err
		}

		if len(lbbg.GetExternalId()) == 0 {
			if _, err := self.createLoadbalancerBackendGroup(ctx, task.GetUserCred(), lbbg, backends); err != nil {
				return nil, err
			}
		} else {
			ilbbg, err := ilb.GetILoadBalancerBackendGroupById(lbbg.GetExternalId())
			if err != nil {
				return nil, err
			}

			group, err := lbbg.GetBackendGroupParams()
			if err != nil {
				return nil, err
			}

			if err := ilbbg.Sync(&group); err != nil {
				return nil, err
			}
		}
		// continue here
		ilbbg, err := ilb.GetILoadBalancerBackendGroupById(lbbg.GetExternalId())
		if err != nil {
			return nil, err
		}

		if err := lbbg.SyncWithCloudLoadbalancerBackendgroup(ctx, task.GetUserCred(), lb, ilbbg, lb.GetOwnerId()); err != nil {
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

				// lblis.AclId = lbacl.ExternalId
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
			params, err := lblis.GetLoadbalancerListenerParams()
			if err != nil {
				return nil, err
			}

			params.BackendGroupID = ""
			err = iListener.Sync(params)
			if err != nil {
				return nil, err
			}
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

		if len(lbbg.ExternalId) == 0 {
			return nil, nil
		}

		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}

		return nil, iLoadbalancerBackendGroup.Delete()
	})
	return nil
}

func (self *SHuaWeiRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
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
			log.Warningf("failed to find guest for lbb %s", lbb.Name)
			return nil, nil
		}
		_, err = guest.GetIVM()
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}

		_, err = iLoadbalancerBackendGroup.GetILoadbalancerBackendById(lbb.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}

		return nil, iLoadbalancerBackendGroup.RemoveBackendServer(lbb.ExternalId, lbb.Weight, lbb.Port)
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
