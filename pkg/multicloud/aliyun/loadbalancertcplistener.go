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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	PersistenceTimeout       int    //是否开启了会话保持。取值为0时，表示没有开启。

	AclType string //	访问控制类型

	AclId string //	监听绑定的访问策略组ID。当AclStatus参数的值为on时，该参数必选。

	HealthCheck               string //	是否开启健康检查。
	HealthCheckType           string //TCP协议监听的健康检查方式。取值：tcp | http
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
		return api.LB_STATUS_ENABLED
	case "configuring", "stopping", "stopped":
		return api.LB_STATUS_DISABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (listerner *SLoadbalancerTCPListener) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (listerner *SLoadbalancerTCPListener) GetEgressMbps() int {
	if listerner.Bandwidth < 1 {
		return 0
	}
	return listerner.Bandwidth
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
	return listerner.HealthCheckType
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

func (listerner *SLoadbalancerTCPListener) GetBackendServerPort() int {
	return listerner.BackendServerPort
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
	if listener.EgressMbps < 1 {
		listener.EgressMbps = -1
	}
	params["Bandwidth"] = fmt.Sprintf("%d", listener.EgressMbps)
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
	switch listener.BackendGroupType {
	case api.LB_BACKENDGROUP_TYPE_NORMAL:
		params["VServerGroupId"] = listener.BackendGroupID
		params["VServerGroup"] = "on"
	case api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
		params["MasterSlaveServerGroupId"] = listener.BackendGroupID
		params["MasterSlaveServerGroup"] = "on"
	case api.LB_BACKENDGROUP_TYPE_DEFAULT:
		params["BackendServerPort"] = fmt.Sprintf("%d", listener.BackendServerPort)
	}
	if len(listener.Name) > 0 {
		params["Description"] = listener.Name
	}

	if utils.IsInStringArray(listener.ListenerType, []string{api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP}) {
		if listener.HealthCheckTimeout >= 1 && listener.HealthCheckTimeout <= 300 {
			params["HealthCheckConnectTimeout"] = fmt.Sprintf("%d", listener.HealthCheckTimeout)
		}
	}
	switch listener.ListenerType {
	case api.LB_LISTENER_TYPE_UDP:
		if len(listener.HealthCheckReq) > 0 {
			params["healthCheckReq"] = listener.HealthCheckReq
		}
		if len(listener.HealthCheckExp) > 0 {
			params["healthCheckExp"] = listener.HealthCheckExp
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

func (listerner *SLoadbalancerTCPListener) GetProjectId() string {
	return ""
}
