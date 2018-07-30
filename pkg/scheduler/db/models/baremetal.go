package models

import (
	"github.com/jinzhu/gorm"
)

type Baremetal struct {
	StandaloneModel
	Status    string `json:"status" gorm:"not null"`
	Enabled   bool   `json:"enabled" gorm:"not null"`
	CliGUID   string `json:"cli_guid,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	CPUCount  int    `json:"cpu_count,omitempty"`
	NodeCount int    `json:"node_count,omitempty"`
	CPUDesc   string `json:"cpu_desc,omitempty"`
	CPUMHZ    int    `json:"cpu_mhz,omitempty"`
	MemSize   int    `json:"mem_size,omitempty"`

	StorageSize   int    `json:"storage_size,omitempty"`
	StorageType   string `json:"storage_type,omitempty"`
	StorageDriver string `json:"storage_driver,omitempty"`
	StorageInfo   string `json:"storage_info,omitempty"`

	IpmiInfo string `json:"ipmi_info,omitempty" gorm:"type:text"`

	Rack  string `json:"rack,omitempty"`
	Slots string `json:"slots,omitempty"`

	ServerID string `json:"server_id,omitempty"`
	UseCount int    `json:"use_count,omitempty"`

	PoolID string `json:"pool_id,omitempty"`
}

func (b Baremetal) TableName() string {
	return baremetalsTable
}

func (b Baremetal) String() string {
	str, _ := JsonString(b)
	return str
}

func NewBaremetalResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Baremetal{}
	}
	models := func() interface{} {
		baremetals := []Baremetal{}
		return &baremetals
	}

	return newResource(db, baremetalsTable, model, models)
}
