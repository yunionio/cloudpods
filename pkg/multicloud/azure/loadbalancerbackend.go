package azure

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SLoadbalancerBackend struct {
	multicloud.SResourceBase

	lbbg *SLoadbalancerBackendGroup
	// networkInterfaces 通过接口地址反查虚拟机地址
	Name        string `json:"name"`
	ID          string `json:"id"`
	Type        string `json:"type"`
	BackendPort int
}

func (self *SLoadbalancerBackend) GetId() string {
	return self.ID + "::" + strconv.Itoa(self.BackendPort)
}

func (self *SLoadbalancerBackend) GetName() string {
	return self.Name
}

func (self *SLoadbalancerBackend) GetGlobalId() string {
	return self.GetId()
}

func (self *SLoadbalancerBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerBackend) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerBackend) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SLoadbalancerBackend) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

func (self *SLoadbalancerBackend) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SLoadbalancerBackend) GetWeight() int {
	return 0
}

func (self *SLoadbalancerBackend) GetPort() int {
	return self.BackendPort
}

func (self *SLoadbalancerBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (self *SLoadbalancerBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self *SLoadbalancerBackend) GetBackendId() string {
	return self.ID
}

func (self *SLoadbalancerBackend) SyncConf(ctx context.Context, port, weight int) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SyncConf")
}
