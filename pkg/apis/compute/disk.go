package compute

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type DiskCreateInput struct {
	apis.Meta

	*DiskConfig

	// prefer options
	PreferRegion string `json:"prefer_region_id"`
	PreferZone   string `json:"prefer_zone_id"`
	PreferWire   string `json:"prefer_wire_id"`
	PreferHost   string `json:"prefer_host_id"`

	Name        string `json:"name"`
	Description string `json:"description"`
	Hypervisor  string `json:"hypervisor"`
	Project     string `json:"project"`
}

// ToServerCreateInput used by disk schedule
func (req *DiskCreateInput) ToServerCreateInput() *ServerCreateInput {
	return &ServerCreateInput{
		ServerConfigs: &ServerConfigs{
			PreferRegion: req.PreferRegion,
			PreferZone:   req.PreferZone,
			PreferWire:   req.PreferWire,
			PreferHost:   req.PreferHost,
			Hypervisor:   req.Hypervisor,
			Disks:        []*DiskConfig{req.DiskConfig},
			Project:      req.Project,
		},
		Name: req.Name,
	}
}

func (req *ServerCreateInput) ToDiskCreateInput() *DiskCreateInput {
	return &DiskCreateInput{
		DiskConfig:   req.Disks[0],
		PreferRegion: req.PreferRegion,
		PreferHost:   req.PreferHost,
		PreferZone:   req.PreferZone,
		PreferWire:   req.PreferWire,
		Name:         req.Name,
		Project:      req.Project,
		Hypervisor:   req.Hypervisor,
	}
}
