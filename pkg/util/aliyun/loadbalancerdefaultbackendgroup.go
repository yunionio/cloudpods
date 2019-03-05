package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/consts"
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
	return consts.LB_STATUS_ENABLED
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) IsDefault() bool {
	return true
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetType() string {
	return consts.LB_BACKENDGROUP_TYPE_DEFAULT
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

func (backendgroup *SLoadbalancerDefaultBackendGroup) Sync(name string) error {
	return cloudprovider.ErrNotSupported
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) AddBackendServer(loadbalancerId, serverId string, weight, port int) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	servers := jsonutils.NewArray()
	servers.Add(jsonutils.Marshal(map[string]string{"ServerId": serverId, "Weight": fmt.Sprintf("%d", weight)}))
	params["BackendServers"] = servers.String()
	_, err := region.lbRequest("AddBackendServers", params)
	return err
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) AddBackendServer(serverId string, weight, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	if err := backendgroup.lb.region.AddBackendServer(backendgroup.lb.LoadBalancerId, serverId, weight, port); err != nil {
		return nil, err
	}
	return &SLoadbalancerDefaultBackend{lbbg: backendgroup, ServerId: serverId, Weight: weight}, nil
}

func (region *SRegion) RemoveBackendServer(loadbalancerId, serverId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	servers := jsonutils.NewArray()
	servers.Add(jsonutils.NewString(serverId))
	params["BackendServers"] = servers.String()
	_, err := region.lbRequest("RemoveBackendServers", params)
	return err
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) RemoveBackendServer(serverId string, weight, port int) error {
	return backendgroup.lb.region.RemoveBackendServer(backendgroup.lb.LoadBalancerId, serverId)
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetProjectId() string {
	return ""
}
