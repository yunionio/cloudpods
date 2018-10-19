package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SManagedResourceBase struct {
	ManagerId string `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"admin_optional"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=True)
}

func (self *SManagedResourceBase) GetCloudprovider() *SCloudprovider {
	if len(self.ManagerId) > 0 {
		return CloudproviderManager.FetchCloudproviderById(self.ManagerId)
	}
	return nil
}

func (self *SManagedResourceBase) GetDriver() (cloudprovider.ICloudProvider, error) {
	provider := self.GetCloudprovider()
	if provider == nil {
		return nil, fmt.Errorf("Resource is self managed")
	}
	return provider.GetDriver()
}

func (self *SManagedResourceBase) IsManaged() bool {
	return len(self.ManagerId) > 0
}

func (self *SManagedResourceBase) getExtraDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	manager := self.GetCloudprovider()
	if manager != nil {
		extra.Add(jsonutils.NewString(manager.Name), "manager")
		extra.Add(jsonutils.NewString(manager.ProjectId), "manager_tenant_id")
		extra.Add(jsonutils.NewString(manager.ProjectId), "manager_project_id")
		project := manager.getProject(ctx)
		if project != nil {
			extra.Add(jsonutils.NewString(project.Name), "manager_tenant")
			extra.Add(jsonutils.NewString(project.Name), "manager_project")
		}
	}
	return extra
}
