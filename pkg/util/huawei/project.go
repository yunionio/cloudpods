package huawei

import (
	"strings"

	"fmt"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/huawei/client"
)

// https://support.huaweicloud.com/api-iam/zh-cn_topic_0057845625.html
type SProject struct {
	client *SHuaweiClient

	IsDomain    bool   `json:"is_domain"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	ID          string `json:"id"`
	ParentID    string `json:"parent_id"`
	DomainID    string `json:"domain_id"`
	Name        string `json:"name"`
}

func (self *SProject) GetRegionID() string {
	return strings.Split(self.Name, "_")[0]
}

func (self *SProject) GetHealthStatus() string {
	if self.Enabled {
		return models.CLOUD_PROVIDER_HEALTH_NORMAL
	}

	return models.CLOUD_PROVIDER_HEALTH_SUSPENDED
}

func (self *SHuaweiClient) fetchProjects() ([]SProject, error) {
	huawei, _ := clients.NewClientWithAccessKey("", "", self.accessKey, self.secret)
	projects := make([]SProject, 0)
	err := DoList(huawei.Projects.List, nil, &projects)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (self *SHuaweiClient) GetProjectById(projectId string) (SProject, error) {
	projects, err := self.fetchProjects()
	if err != nil {
		return SProject{}, err
	}

	for _, project := range projects {
		if project.ID == projectId {
			return project, nil
		}
	}
	return SProject{}, fmt.Errorf("project %s not found", projectId)
}
