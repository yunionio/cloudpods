package aliyun

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SLoadbalancerMasterSlaveBackend struct {
	lbbg *SLoadbalancerMasterSlaveBackendGroup

	ServerId   string
	Weight     int
	Port       int
	ServerType string
}

func (backend *SLoadbalancerMasterSlaveBackend) GetName() string {
	return backend.ServerId
}

func (backend *SLoadbalancerMasterSlaveBackend) GetId() string {
	return fmt.Sprintf("%s/%s", backend.lbbg.MasterSlaveServerGroupId, backend.ServerId)
}

func (backend *SLoadbalancerMasterSlaveBackend) GetGlobalId() string {
	return backend.GetId()
}

func (backend *SLoadbalancerMasterSlaveBackend) GetStatus() string {
	return ""
}

func (backend *SLoadbalancerMasterSlaveBackend) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backend *SLoadbalancerMasterSlaveBackend) IsEmulated() bool {
	return false
}

func (backend *SLoadbalancerMasterSlaveBackend) Refresh() error {
	return nil
}

func (backend *SLoadbalancerMasterSlaveBackend) GetWeight() int {
	return backend.Weight
}

func (backend *SLoadbalancerMasterSlaveBackend) GetPort() int {
	return backend.Port
}

func (backend *SLoadbalancerMasterSlaveBackend) GetBackendType() string {
	return models.LB_BACKEND_GUEST
}

func (backend *SLoadbalancerMasterSlaveBackend) GetBackendRole() string {
	return strings.ToLower(backend.ServerType)
}

func (backend *SLoadbalancerMasterSlaveBackend) GetBackendId() string {
	return backend.ServerId
}
