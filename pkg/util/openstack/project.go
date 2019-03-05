package openstack

import "yunion.io/x/jsonutils"

type SProject struct {
	Description string
	Enabled     bool
	ID          string
	Name        string
}

func (p *SProject) GetId() string {
	return p.ID
}

func (p *SProject) GetGlobalId() string {
	return p.GetId()
}

func (p *SProject) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (p *SProject) GetName() string {
	return p.Name
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
