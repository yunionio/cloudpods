package qcloud

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SLBBackendGroup struct {
	lb       *SLoadbalancer   // 必须不能为nil
	listener *SLBListener     // 必须不能为nil
	rule     *SLBListenerRule // tcp、udp、tcp_ssl监听rule 为nil
}

// https://cloud.tencent.com/document/product/214/30676
// https://cloud.tencent.com/document/product/214/31789
func (self *SLBBackendGroup) AddBackendServer(serverId string, weight int, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	panic("implement me")
}

// https://cloud.tencent.com/document/product/214/30687
// https://cloud.tencent.com/document/product/214/31794
func (self *SLBBackendGroup) RemoveBackendServer(serverId string, weight int, port int) error {
	panic("implement me")
}

// 腾讯云无后端服务器组
func (self *SLBBackendGroup) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLBBackendGroup) Sync(name string) error {
	panic("implement me")
}

func backendGroupIdGen(lbid string, secondId string) string {
	if len(secondId) > 0 {
		return fmt.Sprintf("%s/%s", lbid, secondId)
	} else {
		return lbid
	}
}

// http https 后端服务器只与规则绑定
func (self *SLBBackendGroup) GetId() string {
	t := self.listener.GetListenerType()
	if t == models.LB_LISTENER_TYPE_HTTP || t == models.LB_LISTENER_TYPE_HTTPS {
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
	return ""
}

func (self *SLBBackendGroup) Refresh() error {
	return nil
}

// todo: model没有更新这个字段？
func (self *SLBBackendGroup) IsEmulated() bool {
	return true
}

func (self *SLBBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBBackendGroup) IsDefault() bool {
	return false
}

// todo: ??
func (self *SLBBackendGroup) GetType() string {
	return models.LB_BACKENDGROUP_TYPE_NORMAL
}

func (self *SLBBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
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

	ibackends := make([]cloudprovider.ICloudLoadbalancerBackend, len(backends))
	for i := range backends {
		backends[i].group = self
		ibackends[i] = &backends[i]
	}

	return ibackends, nil
}
