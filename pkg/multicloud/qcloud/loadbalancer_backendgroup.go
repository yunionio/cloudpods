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

package qcloud

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLBBackendGroup struct {
	lb       *SLoadbalancer   // 必须不能为nil
	listener *SLBListener     // 可能为nil
	rule     *SLBListenerRule // tcp、udp、tcp_ssl监听rule 为nil
}

func (self *SLBBackendGroup) GetLoadbalancerId() string {
	return self.lb.GetId()
}

func (self *SLBBackendGroup) GetILoadbalancer() cloudprovider.ICloudLoadbalancer {
	return self.lb
}

func (self *SLBBackendGroup) GetProtocolType() string {
	return ""
}

func (self *SLBBackendGroup) GetScheduler() string {
	return ""
}

func (self *SLBBackendGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (self *SLBBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (self *SLBBackendGroup) GetListenerId() string {
	if self.listener != nil {
		return self.listener.GetId()
	}

	return ""
}

// 返回requestid
func (self *SLBBackendGroup) appLBBackendServer(action string, serverId string, weight int, port int) (string, error) {
	if len(serverId) == 0 {
		return "", fmt.Errorf("loadbalancer backend instance id should not be empty.")
	}

	params := map[string]string{
		"LoadBalancerId":       self.lb.GetId(),
		"ListenerId":           self.GetListenerId(),
		"Targets.0.InstanceId": serverId,
		"Targets.0.Port":       strconv.Itoa(port),
		"Targets.0.Weight":     strconv.Itoa(weight),
	}

	if self.rule != nil {
		params["LocationId"] = self.rule.GetId()
	}

	resp, err := self.lb.region.clbRequest(action, params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

// deprecated
func (self *SLBBackendGroup) appLBSeventhBackendServer(action string, serverId string, weight int, port int) (string, error) {
	if len(serverId) == 0 {
		return "", fmt.Errorf("loadbalancer backend instance id should not be empty.")
	}

	params := map[string]string{
		"loadBalancerId":        self.lb.GetId(),
		"listenerId":            self.GetListenerId(),
		"backends.0.InstanceId": serverId,
		"backends.0.Port":       strconv.Itoa(port),
		"backends.0.Weight":     strconv.Itoa(weight),
	}

	if self.rule != nil {
		params["LocationId"] = self.rule.GetId()
	}

	resp, err := self.lb.region.clbRequest(action, params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

// https://cloud.tencent.com/document/product/214/30677
func (self *SLBBackendGroup) updateBackendServerWeight(action string, serverId string, weight int, port int) (string, error) {
	if len(serverId) == 0 {
		return "", fmt.Errorf("loadbalancer backend instance id should not be empty.")
	}

	params := map[string]string{
		"LoadBalancerId":       self.lb.GetId(),
		"ListenerId":           self.GetListenerId(),
		"Targets.0.InstanceId": serverId,
		"Targets.0.Port":       strconv.Itoa(port),
		"Targets.0.Weight":     strconv.Itoa(weight),
	}

	if self.rule != nil {
		params["LocationId"] = self.rule.GetId()
	}

	resp, err := self.lb.region.clbRequest(action, params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

// https://cloud.tencent.com/document/product/214/30678
func (self *SLBBackendGroup) updateBackendServerPort(action string, serverId string, oldPort int, newPort int) (string, error) {
	if len(serverId) == 0 {
		return "", fmt.Errorf("loadbalancer backend instance id should not be empty.")
	}

	params := map[string]string{
		"LoadBalancerId":       self.lb.GetId(),
		"ListenerId":           self.GetListenerId(),
		"Targets.0.InstanceId": serverId,
		"Targets.0.Port":       strconv.Itoa(oldPort),
		"NewPort":              strconv.Itoa(newPort),
	}

	if self.rule != nil {
		params["LocationId"] = self.rule.GetId()
	}

	resp, err := self.lb.region.clbRequest(action, params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

// 返回requestid
// https://cloud.tencent.com/document/api/214/1264
func (self *SLBBackendGroup) classicLBBackendServer(action string, serverId string, weight int, port int) (string, error) {
	// 传统型负载均衡忽略了port参数
	params := map[string]string{
		"LoadBalancerId":       self.lb.GetId(),
		"Targets.0.InstanceId": serverId,
		"Targets.0.Weight":     strconv.Itoa(weight),
	}

	resp, err := self.lb.region.clbRequest(action, params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

// https://cloud.tencent.com/document/product/214/30676
// https://cloud.tencent.com/document/product/214/31789
func (self *SLBBackendGroup) AddBackendServer(serverId string, weight int, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	var requestId string
	var err error
	if self.lb.Forward == LB_TYPE_APPLICATION {
		requestId, err = self.appLBBackendServer("RegisterTargets", serverId, weight, port)
	} else {
		requestId, err = self.classicLBBackendServer("RegisterTargetsWithClassicalLB", serverId, weight, port)
	}

	if err != nil {
		return nil, err
	}

	err = self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
	if err != nil {
		return nil, err
	}

	err = self.Refresh()
	if err != nil {
		return nil, err
	}

	backends, err := self.GetBackends()
	if err != nil {
		return nil, err
	}

	for _, backend := range backends {
		if strings.HasSuffix(backend.GetId(), fmt.Sprintf("%s-%d", serverId, port)) {
			return &backend, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

// https://cloud.tencent.com/document/product/214/30687
// https://cloud.tencent.com/document/product/214/31794
func (self *SLBBackendGroup) RemoveBackendServer(serverId string, weight int, port int) error {
	_, err := self.lb.region.GetInstance(serverId)
	if err == cloudprovider.ErrNotFound {
		return nil
	}

	var requestId string
	if self.lb.Forward == LB_TYPE_APPLICATION {
		requestId, err = self.appLBBackendServer("DeregisterTargets", serverId, weight, port)
	} else {
		requestId, err = self.classicLBBackendServer("DeregisterTargetsFromClassicalLB", serverId, weight, port)
	}

	if err != nil {
		if strings.Contains(err.Error(), "not registered") {
			return nil
		}
		return err
	}

	return self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
}

func (self *SLBBackendGroup) UpdateBackendServer(serverId string, oldWeight, oldPort, weight, newPort int) error {
	var requestId string
	var err error
	// if self.lb.Forward == LB_TYPE_APPLICATION {
	// 	// https://cloud.tencent.com/document/product/214/30678
	// 	// https://cloud.tencent.com/document/product/214/30677
	// 	if self.listener.Protocol == api.LB_LISTENER_TYPE_HTTPS || self.listener.Protocol == api.LB_LISTENER_TYPE_HTTP {
	// 		requestId, err = self.appLBSeventhBackendServer("ModifyForwardSeventhBackends", serverId, weight, port)
	// 	} else {
	// 		requestId, err = self.appLBBackendServer("ModifyForwardFourthBackendsWeight", serverId, weight, port)
	// 	}
	// } else {
	// 	requestId, err = self.classicLBBackendServer("ModifyLoadBalancerBackends", serverId, weight, port)
	// }
	if oldWeight != weight {
		requestId, err = self.updateBackendServerWeight("ModifyTargetWeight", serverId, weight, oldPort)
		if err != nil {
			if strings.Contains(err.Error(), "not registered") {
				return nil
			}
			return errors.Wrap(err, "LBBackendGroup.updateBackendServerWeight")
		}

		err = self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
		if err != nil {
			return errors.Wrap(err, "LBBackendGroup.updateBackendServerWeight.WaitLBTaskSuccess")
		}
	}

	if oldPort != newPort {
		requestId, err = self.updateBackendServerPort("ModifyTargetPort", serverId, oldPort, newPort)
		if err != nil {
			if strings.Contains(err.Error(), "not registered") {
				return nil
			}
			return errors.Wrap(err, "LBBackendGroup.updateBackendServerPort")
		}

		err = self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
		if err != nil {
			return errors.Wrap(err, "LBBackendGroup.updateBackendServerPort.WaitLBTaskSuccess")
		}
	}

	return nil
}

// 腾讯云无后端服务器组。
func (self *SLBBackendGroup) Delete() error {
	return fmt.Errorf("Please remove related listener/rule frist")
}

// 腾讯云无后端服务器组
func (self *SLBBackendGroup) Sync(group *cloudprovider.SLoadbalancerBackendGroup) error {
	return nil
}

func backendGroupIdGen(lbid string, secondId string) string {
	if len(secondId) > 0 {
		return fmt.Sprintf("%s", secondId)
	} else {
		return lbid
	}
}

func (self *SLBBackendGroup) GetId() string {
	t := ""
	if self.listener != nil {
		t = self.listener.GetListenerType()
	}

	if t == api.LB_LISTENER_TYPE_HTTP || t == api.LB_LISTENER_TYPE_HTTPS {
		// http https 后端服务器只与规则绑定
		return backendGroupIdGen(self.lb.GetId(), self.rule.GetId())
	} else if self.lb.Forward == LB_TYPE_APPLICATION {
		return backendGroupIdGen(self.lb.GetId(), self.GetListenerId())
	} else {
		// 传统型lb 所有监听共用一个后端服务器组
		return backendGroupIdGen(self.lb.GetId(), "")
	}
}

func (self *SLBBackendGroup) GetName() string {
	return self.GetId()
}

func (self *SLBBackendGroup) GetGlobalId() string {
	return self.GetId()
}

func (self *SLBBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLBBackendGroup) Refresh() error {
	return nil
}

// note: model没有更新这个字段？
func (self *SLBBackendGroup) IsEmulated() bool {
	return true
}

func (self *SLBBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBBackendGroup) IsDefault() bool {
	return false
}

func (self *SLBBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (self *SLBBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := self.GetBackends()
	if err != nil {
		return nil, err
	}

	ibackends := make([]cloudprovider.ICloudLoadbalancerBackend, len(backends))
	for i := range backends {
		backends[i].group = self
		ibackends[i] = &backends[i]
	}

	return ibackends, nil
}

func (self *SLBBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := self.GetILoadbalancerBackends()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(backends); i++ {
		if backends[i].GetGlobalId() == backendId {
			return backends[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SLBBackendGroup) GetBackends() ([]SLBBackend, error) {
	backends := []SLBBackend{}
	var err error
	if self.rule != nil {
		// http、https监听
		backends, err = self.lb.region.GetLBBackends(self.lb.Forward, self.lb.GetId(), self.GetListenerId(), self.rule.GetId())
		if err != nil {
			return nil, err
		}
	} else {
		// tcp,udp,tcp_ssl监听
		backends, err = self.lb.region.GetLBBackends(self.lb.Forward, self.lb.GetId(), self.GetListenerId(), "")
		if err != nil {
			return nil, err
		}
	}

	for i := range backends {
		backends[i].group = self
	}

	return backends, nil
}

func (self *SLBBackendGroup) GetProjectId() string {
	return ""
}
