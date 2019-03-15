package qcloud

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
)

type SProject struct {
	client *SQcloudClient

	ProjectName string    `json:"projectName"`
	ProjectId   string    `json:"projectId"`
	CreateTime  time.Time `json:"createTime"`
	CreateorUin int       `json:"creatorUin"`
	ProjectInfo string    `json:"projectInfo"`
}

func (p *SProject) GetId() string {
	var pId string
	pos := strings.Index(p.ProjectId, ".")
	if pos >= 0 {
		pId = p.ProjectId[:pos]
	} else {
		pId = p.ProjectId
	}
	return pId
}

func (p *SProject) GetGlobalId() string {
	return p.client.providerId + "/" + p.GetId()
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
