package qcloud

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SLBListenerRule struct {
	listener *SLBListener

	Domain            string      `json:"Domain"`
	Certificate       certificate `json:"Certificate"`
	URL               string      `json:"Url"`
	HealthCheck       healthCheck `json:"HealthCheck"`
	LocationID        string      `json:"LocationId"`
	Scheduler         string      `json:"Scheduler"`
	SessionExpireTime int64       `json:"SessionExpireTime"`
}

func (self *SLBListenerRule) Delete() error {
	panic("implement me")
}

func (self *SLBListenerRule) GetId() string {
	return self.LocationID
}

func (self *SLBListenerRule) GetName() string {
	return self.LocationID
}

func (self *SLBListenerRule) GetGlobalId() string {
	return self.LocationID
}

// todo： rule status？？
func (self *SLBListenerRule) GetStatus() string {
	return ""
}

func (self *SLBListenerRule) Refresh() error {
	return nil
}

func (self *SLBListenerRule) IsEmulated() bool {
	return false
}

func (self *SLBListenerRule) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBListenerRule) GetDomain() string {
	return self.Domain
}

func (self *SLBListenerRule) GetPath() string {
	return self.URL
}

func (self *SLBListenerRule) GetBackendGroup() *SLBBackendGroup {
	t := self.listener.GetListenerType()
	if t == models.LB_LISTENER_TYPE_HTTP || t == models.LB_LISTENER_TYPE_HTTPS {
		return &SLBBackendGroup{
			lb:       self.listener.lb,
			listener: self.listener,
			rule:     self,
		}
	}

	return nil
}

// 只有http、https协议监听规则有backendgroupid
func (self *SLBListenerRule) GetBackendGroupId() string {
	bg := self.GetBackendGroup()
	if bg == nil {
		return ""
	}

	return bg.GetId()
}
