package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/compute/consts"
)

type SLoadbalancerDefaultBackend struct {
	lbbg *SLoadbalancerDefaultBackendGroup

	ServerId string
	Weight   int
}

func (backend *SLoadbalancerDefaultBackend) GetName() string {
	return backend.ServerId
}

func (backend *SLoadbalancerDefaultBackend) GetId() string {
	return fmt.Sprintf("%s/%s", backend.lbbg.lb.LoadBalancerId, backend.ServerId)
}

func (backend *SLoadbalancerDefaultBackend) GetGlobalId() string {
	return backend.GetId()
}

func (backend *SLoadbalancerDefaultBackend) GetStatus() string {
	return ""
}

func (backend *SLoadbalancerDefaultBackend) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backend *SLoadbalancerDefaultBackend) IsEmulated() bool {
	return false
}

func (backend *SLoadbalancerDefaultBackend) Refresh() error {
	return nil
}

func (backend *SLoadbalancerDefaultBackend) GetWeight() int {
	return backend.Weight
}

func (backend *SLoadbalancerDefaultBackend) GetPort() int {
	return 0
}

func (backend *SLoadbalancerDefaultBackend) GetBackendType() string {
	return consts.LB_BACKEND_GUEST
}

func (backend *SLoadbalancerDefaultBackend) GetBackendRole() string {
	return consts.LB_BACKEND_ROLE_DEFAULT
}

func (backend *SLoadbalancerDefaultBackend) GetBackendId() string {
	return backend.ServerId
}

func (backend *SLoadbalancerDefaultBackend) GetProjectId() string {
	return ""
}
