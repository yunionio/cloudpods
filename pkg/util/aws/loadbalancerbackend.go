package aws

import (
	"fmt"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SElbBackend struct {
	region *SRegion
	group  *SElbBackendGroup

	Target       Target       `json:"Target"`
	TargetHealth TargetHealth `json:"TargetHealth"`
}

type Target struct {
	ID   string `json:"Id"`
	Port int    `json:"Port"`
}

type TargetHealth struct {
	State       string `json:"State"`
	Reason      string `json:"Reason"`
	Description string `json:"Description"`
}

func (self *SElbBackend) GetId() string {
	return fmt.Sprintf("%s::%s::%d", self.group.GetId(), self.Target.ID, self.Target.Port)
}

func (self *SElbBackend) GetName() string {
	return self.GetId()
}

func (self *SElbBackend) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbBackend) Refresh() error {
	return nil
}

func (self *SElbBackend) IsEmulated() bool {
	return false
}

func (self *SElbBackend) GetMetadata() *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

func (self *SElbBackend) GetProjectId() string {
	return ""
}

func (self *SElbBackend) GetWeight() int {
	return 0
}

func (self *SElbBackend) GetPort() int {
	return self.Target.Port
}

func (self *SElbBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (self *SElbBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self *SElbBackend) GetBackendId() string {
	return self.Target.ID
}

func (self *SElbBackend) SyncConf(port, weight int) error {
	return self.region.SyncElbBackend(self.GetId(), self.GetBackendId(), self.Target.Port, port)
}

func (self *SRegion) SyncElbBackend(backendId, serverId string, oldPort, newPort int) error {
	err := self.RemoveElbBackend(backendId, serverId, 0, oldPort)
	if err != nil {
		return err
	}

	_, err = self.AddElbBackend(backendId, serverId, 0, newPort)
	if err != nil {
		return err
	}

	return nil
}
