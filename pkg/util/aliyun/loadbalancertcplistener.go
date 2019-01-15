package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/utils"
)

type SLoadbalancerTCPListener struct {
	lb *SLoadbalancer

	ListenerPort      int    //	负载均衡实例前端使用的端口。
	BackendServerPort int    //	负载均衡实例后端使用的端口。
	Bandwidth         int    //	监听的带宽峰值。
	Status            string //	当前监听的状态，取值：starting | running | configuring | stopping | stopped
	Description       string

	Scheduler                string //	调度算法。
	VServerGroupId           string //	绑定的服务器组ID。
	MasterSlaveServerGroupId string //	绑定的主备服务器组ID。
	AclStatus                string //	是否开启访问控制功能。取值：on | off（默认值）

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
	if len(listener.Description) == 0 {
		listener.Refresh()
	}
	if len(listener.Description) > 0 {
		return listener.Description
	}
	return fmt.Sprintf("TCP:%d", listener.ListenerPort)
}

func (listerner *SLoadbalancerTCPListener) GetId() string {
	return fmt.Sprintf("%s/%d", listerner.lb.LoadBalancerId, listerner.ListenerPort)
}

func (listerner *SLoadbalancerTCPListener) GetGlobalId() string {
	return listerner.GetId()
}

func (listerner *SLoadbalancerTCPListener) GetStatus() string {
	switch listerner.Status {
	case "starting", "running":
		return models.LB_STATUS_ENABLED
	case "configuring", "stopping", "stopped":
		return models.LB_STATUS_DISABLED
	default:
		return models.LB_STATUS_UNKNOWN
	}
}

func (listerner *SLoadbalancerTCPListener) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (listerner *SLoadbalancerTCPListener) IsEmulated() bool {
	return false
}

func (listerner *SLoadbalancerTCPListener) Refresh() error {
	lis, err := listerner.lb.region.GetLoadbalancerTCPListener(listerner.lb.LoadBalancerId, listerner.ListenerPort)
	if err != nil {
		return err
	}
	return jsonutils.Update(listerner, lis)
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
	return listerner.MasterSlaveServerGroupId
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

func (region *SRegion) constructBaseCreateListenerParams(lb *SLoadbalancer, listener *cloudprovider.SLoadbalancerListener) map[string]string {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	switch lb.InternetChargeType {
	case "paybytraffic":
		params["Bandwidth"] = "-1"
	case "paybybandwidth":
		if lb.Bandwidth > 5000 {
			lb.Bandwidth = 5000
		}
		params["Bandwidth"] = fmt.Sprintf("%d", lb.Bandwidth)
	default:
		params["Bandwidth"] = fmt.Sprintf("%d", listener.Bandwidth)
	}
	params["ListenerPort"] = fmt.Sprintf("%d", listener.ListenerPort)
	params["LoadBalancerId"] = lb.LoadBalancerId
	if len(listener.AccessControlListID) > 0 {
		params["AclId"] = listener.AccessControlListID
	}
	if utils.IsInStringArray(listener.AccessControlListStatus, []string{"on", "off"}) {
		params["AclStatus"] = listener.AccessControlListStatus
	}
	if utils.IsInStringArray(listener.AccessControlListType, []string{"white", "black"}) {
		params["AclType"] = listener.AccessControlListType
	}
	if len(listener.BackendGroupID) > 0 {
		switch listener.BackendGroupType {
		case models.LB_BACKENDGROUP_TYPE_NORMAL:
			params["VServerGroupId"] = listener.BackendGroupID
		case models.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
			params["MasterSlaveServerGroupId"] = listener.BackendGroupID
		}
	} else {
		//TODO
		params["BackendServerPort"] = ""
	}
	if len(listener.Name) > 0 {
		params["Description"] = listener.Name
	}
	if listener.EstablishedTimeout >= 10 && listener.EstablishedTimeout <= 900 {
		params["EstablishedTimeout"] = fmt.Sprintf("%d", listener.EstablishedTimeout)
	}

	if utils.IsInStringArray(listener.ListenerType, []string{models.LB_LISTENER_TYPE_TCP, models.LB_LISTENER_TYPE_UDP}) {
		if listener.HealthCheckTimeout >= 1 && listener.HealthCheckTimeout <= 300 {
			params["HealthCheckConnectTimeout"] = fmt.Sprintf("%d", listener.HealthCheckTimeout)
		}
	}

	if len(listener.HealthCheckDomain) > 0 {
		params["HealthCheckDomain"] = listener.HealthCheckDomain
	}

	if len(listener.HealthCheckHttpCode) > 0 {
		params["HealthCheckHttpCode"] = listener.HealthCheckHttpCode
	}

	if len(listener.HealthCheckURI) > 0 {
		params["HealthCheckURI"] = listener.HealthCheckURI
	}

	if listener.HealthCheckRise >= 2 && listener.HealthCheckRise <= 10 {
		params["HealthyThreshold"] = fmt.Sprintf("%d", listener.HealthCheckRise)
	}

	if listener.HealthCheckFail >= 2 && listener.HealthCheckFail <= 10 {
		params["UnhealthyThreshold"] = fmt.Sprintf("%d", listener.HealthCheckFail)
	}

	if listener.HealthCheckInterval >= 1 && listener.HealthCheckInterval <= 50 {
		params["healthCheckInterval"] = fmt.Sprintf("%d", listener.HealthCheckInterval)
	}

	params["Scheduler"] = listener.Scheduler
	return params
}

func (region *SRegion) CreateLoadbalancerTCPListener(lb *SLoadbalancer, listener *cloudprovider.SLoadbalancerListener) (cloudprovider.ICloudLoadbalancerListener, error) {
	params := region.constructBaseCreateListenerParams(lb, listener)
	_, err := region.lbRequest("CreateLoadBalancerTCPListener", params)
	if err != nil {
		return nil, err
	}
	iListener, err := region.GetLoadbalancerTCPListener(lb.LoadBalancerId, listener.ListenerPort)
	if err != nil {
		return nil, err
	}
	iListener.lb = lb
	return iListener, nil
}

func (listerner *SLoadbalancerTCPListener) Delete() error {
	return listerner.lb.region.DeleteLoadbalancerListener(listerner.lb.LoadBalancerId, listerner.ListenerPort)
}

func (listerner *SLoadbalancerTCPListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (listerner *SLoadbalancerTCPListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (listerner *SLoadbalancerTCPListener) Start() error {
	return listerner.lb.region.startListener(listerner.ListenerPort, listerner.lb.LoadBalancerId)
}

func (listerner *SLoadbalancerTCPListener) Stop() error {
	return listerner.lb.region.stopListener(listerner.ListenerPort, listerner.lb.LoadBalancerId)
}

func (region *SRegion) SyncLoadbalancerTCPListener(lb *SLoadbalancer, listener *cloudprovider.SLoadbalancerListener) error {
	params := region.constructBaseCreateListenerParams(lb, listener)
	_, err := region.lbRequest("SetLoadBalancerTCPListenerAttribute", params)
	return err
}

func (listerner *SLoadbalancerTCPListener) Sync(lblis *cloudprovider.SLoadbalancerListener) error {
	return listerner.lb.region.SyncLoadbalancerTCPListener(listerner.lb, lblis)
}
