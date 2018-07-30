package models

import (
	"github.com/jinzhu/gorm"
)

type BaremetalAgent struct {
	StandaloneModel
	AccessIP   string `json:"access_ip" gorm:"not null"`
	ManagerURI string `json:"manager_uri,omitempty"`
	Status     string `json:"status" gorm:"not null"`
	ZoneID     string `json:"zone_id,omitempty"`
	Version    string `json:"version,omitempty"`
}

func (b BaremetalAgent) TableName() string {
	return baremetalAgentsTable
}

func (b BaremetalAgent) String() string {
	s, _ := JsonString(b)
	return s
}

func NewBaremetalAgentResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &BaremetalAgent{}
	}
	models := func() interface{} {
		agents := []BaremetalAgent{}
		return &agents
	}

	return newResource(db, baremetalAgentsTable, model, models)
}
