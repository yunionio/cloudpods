package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SLoadbalancerDefaultBackendGroup struct {
	lb *SLoadbalancer
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetName() string {
	return fmt.Sprintf("%s(%s)-default", backendgroup.lb.LoadBalancerName, backendgroup.lb.Address)
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetId() string {
	return fmt.Sprintf("%s/default", backendgroup.lb.LoadBalancerId)
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetGlobalId() string {
	return backendgroup.GetId()
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetStatus() string {
	return models.LB_STATUS_ENABLED
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) IsDefault() bool {
	return true
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetType() string {
	return models.LB_BACKENDGROUP_TYPE_DEFAULT
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) IsEmulated() bool {
	return false
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) Refresh() error {
	return nil
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	loadbalancer, err := backendgroup.lb.region.GetLoadbalancerDetail(backendgroup.lb.LoadBalancerId)
	if err != nil {
		return nil, err
	}
	ibackends := []cloudprovider.ICloudLoadbalancerBackend{}
	for i := 0; i < len(loadbalancer.BackendServers.BackendServer); i++ {
		loadbalancer.BackendServers.BackendServer[i].lbbg = backendgroup
		ibackends = append(ibackends, &loadbalancer.BackendServers.BackendServer[i])
	}
	return ibackends, nil
}
