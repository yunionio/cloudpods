package qcloud

import (
	"strings"
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
	if strings.Index(p.ProjectId, ".") != -1 {
		return strings.Split(p.ProjectId, ".")[0]
	}
	return ""
}

func (p *SProject) GetGlobalId() string {
	return p.GetId()
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
