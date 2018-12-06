package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerTCPListener struct {
	lb *SLoadbalancer

	ListenerPort      int    //	负载均衡实例前端使用的端口。
	BackendServerPort int    //	负载均衡实例后端使用的端口。
	Bandwidth         int    //	监听的带宽峰值。
	Status            string //	当前监听的状态，取值：starting | running | configuring | stopping | stopped

	Scheduler               string //	调度算法。
	VServerGroupId          string //	绑定的服务器组ID。
	MaterSlaveServerGroupId string //	绑定的主备服务器组ID。
	AclStatus               string //	是否开启访问控制功能。取值：on | off（默认值）

	AclType string //	访问控制类型

	AclId string //	监听绑定的访问策略组ID。当AclStatus参数的值为on时，该参数必选。

	HealthCheck               string //	是否开启健康检查。
	HealthyThreshold          int    //	健康检查阈值。
	UnhealthyThreshold        int    //	不健康检查阈值。
	HealthCheckConnectTimeout int    //	每次健康检查响应的最大超时间，单位为秒。
	HealthCheckInterval       int    //	健康检查的时间间隔，单位为秒。
	HealthCheckConnectPort    int    //	健康检查的端口。
}

func (listener *SLoadbalancerTCPListener) GetName() string {
	return fmt.Sprintf("TCP:%d", listener.ListenerPort)
}

func (listerner *SLoadbalancerTCPListener) GetId() string {
	return fmt.Sprintf("%s/%d", listerner.lb.LoadBalancerId, listerner.ListenerPort)
}

func (listerner *SLoadbalancerTCPListener) GetGlobalId() string {
	return listerner.GetId()
}

func (listerner *SLoadbalancerTCPListener) GetStatus() string {
	return listerner.Status
}

func (listerner *SLoadbalancerTCPListener) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (listerner *SLoadbalancerTCPListener) IsEmulated() bool {
	return false
}

func (listerner *SLoadbalancerTCPListener) Refresh() error {
	return nil
}

func (listerner *SLoadbalancerTCPListener) GetListenerType() string {
	return "tcp"
}
func (listerner *SLoadbalancerTCPListener) GetListenerPort() int {
	return listerner.ListenerPort
}

func (listerner *SLoadbalancerTCPListener) GetBackendGroupId() string {
	if len(listerner.VServerGroupId) > 0 {
		return listerner.VServerGroupId
	}
	return listerner.MaterSlaveServerGroupId
}

func (listerner *SLoadbalancerTCPListener) GetScheduler() string {
	return listerner.Scheduler
}

func (listerner *SLoadbalancerTCPListener) GetAclStatus() string {
	return listerner.AclStatus
}

func (listerner *SLoadbalancerTCPListener) GetAclType() string {
	return listerner.AclType
}

func (listerner *SLoadbalancerTCPListener) GetAclId() string {
	return listerner.AclId
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheck() string {
	return listerner.HealthCheck
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckType() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckDomain() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckURI() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckCode() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckRise() int {
	return listerner.HealthyThreshold
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckFail() int {
	return listerner.UnhealthyThreshold
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckTimeout() int {
	return listerner.HealthCheckConnectTimeout
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckInterval() int {
	return listerner.HealthCheckInterval
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckReq() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetHealthCheckExp() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetStickySession() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetStickySessionType() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetStickySessionCookie() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetStickySessionCookieTimeout() int {
	return 0
}

func (listerner *SLoadbalancerTCPListener) XForwardedForEnabled() bool {
	return false
}

func (listerner *SLoadbalancerTCPListener) GzipEnabled() bool {
	return false
}

func (listerner *SLoadbalancerTCPListener) GetCertificateId() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) GetTLSCipherPolicy() string {
	return ""
}

func (listerner *SLoadbalancerTCPListener) HTTP2Enabled() bool {
	return false
}

func (listerner *SLoadbalancerTCPListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return []cloudprovider.ICloudLoadbalancerListenerRule{}, nil
}

func (region *SRegion) GetLoadbalancerTCPListener(loadbalancerId string, listenerPort int) (*SLoadbalancerTCPListener, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	body, err := region.lbRequest("DescribeLoadBalancerTCPListenerAttribute", params)
	if err != nil {
		return nil, err
	}
	listener := SLoadbalancerTCPListener{}
	return &listener, body.Unmarshal(&listener)
}
