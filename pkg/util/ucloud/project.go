package ucloud

import "yunion.io/x/jsonutils"

// https://docs.ucloud.cn/api/summary/get_project_list
type SProject struct {
	ProjectID     string `json:"ProjectId"`
	ProjectName   string `json:"ProjectName"`
	ParentID      string `json:"ParentId"`
	ParentName    string `json:"ParentName"`
	CreateTime    int64  `json:"CreateTime"`
	IsDefault     bool   `json:"IsDefault"`
	MemberCount   int64  `json:"MemberCount"`
	ResourceCount int64  `json:"ResourceCount"`
}

func (self *SProject) GetId() string {
	return self.ProjectID
}

func (self *SProject) GetName() string {
	return self.ProjectName
}

func (self *SProject) GetGlobalId() string {
	return self.GetId()
}

func (self *SProject) GetStatus() string {
	return ""
}

func (self *SProject) Refresh() error {
	return nil
}

func (self *SProject) IsEmulated() bool {
	return false
}

func (self *SProject) GetMetadata() *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

func (self *SUcloudClient) FetchProjects() ([]SProject, error) {
	params := NewUcloudParams()
	projects := make([]SProject, 0)
	err := self.DoListAll("GetProjectList", params, &projects)
	return projects, err
}
