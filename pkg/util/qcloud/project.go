package qcloud

import (
	"time"

	"yunion.io/x/jsonutils"
)

type SProject struct {
	ProjectName string    `json:"projectName"`
	ProjectId   string    `json:projectId`
	CreateTime  time.Time `json:createTime`
	CreateorUin int       `json:"creatorUin"`
	ProjectInfo string    `json:"projectInfo"`
}

func (p *SProject) GetId() string {
	return p.ProjectId
}

func (p *SProject) GetGlobalId() string {
	return p.ProjectId
}

func (p *SProject) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (p *SProject) GetName() string {
	return p.ProjectName
}

func (p *SProject) GetStatus() string {
	return ""
}

func (p *SProject) IsEmulated() bool {
	return false
}

func (p *SProject) Refresh() error {
	return nil
}
