package regiondrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SQcloudRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SQcloudRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SQcloudRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lbbg.GetIRegion()
		if err != nil {
			return nil, err
		}

		// 腾讯云本身没有后端服务器组，因此不需要在qcloud端执行创建操作
		if iRegion.GetProvider() == models.CLOUD_PROVIDER_QCLOUD {
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
		if err := lbbg.SetExternalId(iLoadbalancerBackendGroup.GetGlobalId()); err != nil {
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
		if lbbg.GetProviderName() == models.CLOUD_PROVIDER_QCLOUD && lbbg.RefCount() == 0 {
			return nil, nil
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
		if err := lbb.SetExternalId(iLoadbalancerBackend.GetGlobalId()); err != nil {
			return nil, err
		}
		return nil, lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iLoadbalancerBackend, "", false)
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
		if iRegion.GetProvider() == models.CLOUD_PROVIDER_QCLOUD {
			count := lbbg.RefCount()
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
		if err := lblis.SetExternalId(iListener.GetGlobalId()); err != nil {
			return nil, err
		}

		// ====腾讯云添加后端服务器=====
		if iRegion.GetProvider() == models.CLOUD_PROVIDER_QCLOUD {
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

		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, "", false)
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
		if err := lbr.SetExternalId(iListenerRule.GetGlobalId()); err != nil {
			return nil, err
		}
		// ====腾讯云添加后端服务器=====
		if listener.GetProviderName() == models.CLOUD_PROVIDER_QCLOUD && len(rule.BackendGroupID) > 0 {
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
		return nil, lbr.SyncWithCloudLoadbalancerListenerRule(ctx, userCred, iListenerRule, "", false)
	})
	return nil
}
