package qcloud

import (
	"fmt"
	"strconv"
	"time"

	"strings"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SLBBackendGroup struct {
	lb       *SLoadbalancer   // 必须不能为nil
	listener *SLBListener     // 必须不能为nil
	rule     *SLBListenerRule // tcp、udp、tcp_ssl监听rule 为nil
}

// 返回requestid
func (self *SLBBackendGroup) appLBBackendServer(action string, serverId string, weight int, port int) (string, error) {
	if len(serverId) == 0 {
		return "", fmt.Errorf("loadbalancer backend instance id should not be empty.")
	}

	params := map[string]string{
		"LoadBalancerId":       self.lb.GetId(),
		"ListenerId":           self.listener.GetId(),
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

// 返回requestid
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
	var requestId string
	var err error
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

// 腾讯云无后端服务器组。
func (self *SLBBackendGroup) Delete() error {
	return fmt.Errorf("Please remove related listener/rule frist")
}

// 腾讯云无后端服务器组
func (self *SLBBackendGroup) Sync(name string) error {
	return nil
}

func backendGroupIdGen(lbid string, secondId string) string {
	if len(secondId) > 0 {
		return fmt.Sprintf("%s/%s", lbid, secondId)
	} else {
		return lbid
	}
}

func (self *SLBBackendGroup) GetId() string {
	t := self.listener.GetListenerType()
	if t == models.LB_LISTENER_TYPE_HTTP || t == models.LB_LISTENER_TYPE_HTTPS {
		// http https 后端服务器只与规则绑定
		return backendGroupIdGen(self.lb.GetId(), self.rule.GetId())
	} else if self.lb.Forward == LB_TYPE_APPLICATION {
		return backendGroupIdGen(self.lb.GetId(), self.listener.GetId())
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
	return models.LB_STATUS_ENABLED
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
	return models.LB_BACKENDGROUP_TYPE_NORMAL
}

func (self *SLBBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := self.GetBackends()
	if err != nil {
		return nil, err
	}

	ibackends := make([]cloudprovider.ICloudLoadbalancerBackend, len(backends))
	for i := range backends {
		ibackends[i] = &backends[i]
	}

	return ibackends, nil
}

func (self *SLBBackendGroup) GetBackends() ([]SLBBackend, error) {
	backends := []SLBBackend{}
	var err error
	if self.rule != nil {
		// http、https监听
		backends, err = self.lb.region.GetLBBackends(self.lb.Forward, self.lb.GetId(), self.listener.GetId(), self.rule.GetId())
		if err != nil {
			return nil, err
		}
	} else {
		// tcp,udp,tcp_ssl监听
		backends, err = self.lb.region.GetLBBackends(self.lb.Forward, self.lb.GetId(), self.listener.GetId(), "")
		if err != nil {
			return nil, err
		}
	}

	for i := range backends {
		backends[i].group = self
	}

	return backends, nil
}
