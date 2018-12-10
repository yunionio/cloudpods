package aliyun

import "yunion.io/x/jsonutils"

type SLoadbalancerUDPListener struct {
	lb *SLoadbalancer

	ListenerPort      int    //	负载均衡实例前端使用的端口。
	BackendServerPort int    //	负载均衡实例后端使用的端口。
	Bandwidth         int    //	监听的带宽峰值。
	Status            string //	当前监听的状态，取值：starting | running | configuring | stopping | stopped

	Scheduler               string //	调度算法
	VServerGroupId          string //	绑定的服务器组ID。
	MaterSlaveServerGroupId string //	绑定的主备服务器组ID。
	AclStatus               string //	是否开启访问控制功能。取值：on | off（默认值）

	AclType string //	访问控制类型：

	AclId string //	监听绑定的访问策略组ID。当AclStatus参数的值为on时，该参数必选。

	HealthCheck               string //	是否开启健康检查。
	HealthyThreshold          int    //	健康检查阈值。
	UnhealthyThreshold        int    //	不健康检查阈值。
	HealthCheckConnectTimeout int    //	每次健康检查响应的最大超时间，单位为秒。
	HealthCheckInterval       int    //	健康检查的时间间隔，单位为秒。
	HealthCheckConnectPort    int    //	健康检查的端口。
}

func (listener *SLoadbalancerUDPListener) GetName() string {
	return ""
}

func (listerner *SLoadbalancerUDPListener) GetId() string {
	return ""
}

func (listerner *SLoadbalancerUDPListener) GetGlobalId() string {
	return listerner.GetId()
}

func (listerner *SLoadbalancerUDPListener) GetStatus() string {
	return ""
}

func (listerner *SLoadbalancerUDPListener) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (listerner *SLoadbalancerUDPListener) IsEmulated() bool {
	return false
}

func (listerner *SLoadbalancerUDPListener) Refresh() error {
	return nil
}

func (region *SRegion) GetLoadbalancerUDPListener(loadbalancerId string, listenerPort string) (*SLoadbalancerUDPListener, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = listenerPort
	body, err := region.lbRequest("DescribeLoadBalancerUDPListenerAttribute", params)
	if err != nil {
		return nil, err
	}
	listener := SLoadbalancerUDPListener{}
	return &listener, body.Unmarshal(&listener)
}
