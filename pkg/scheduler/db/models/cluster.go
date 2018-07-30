package models

import (
	"encoding/json"

	"github.com/jinzhu/gorm"
)

type Cluster struct {
	StandaloneModel
	HostIPStart  string `json:"host_ip_start" gorm:"not null"`
	HostIPEnd    string `json:"host_ip_end" gorm:"not null"`
	HostNetmask  int    `json:"host_netmask,omitempty"`
	HostGateway  string `json:"host_gateway,omitempty"`
	HostDNS      string `json:"host_dns,omitempty"`
	ScheduleRank int    `json:"schedule_rank,omitempty"`
	ZoneID       string `json:"zone_id" gorm:"not null"`
}

func (c Cluster) TableName() string {
	return clustersTable
}

func (c Cluster) String() string {
	s, _ := json.Marshal(c)
	return string(s)
}

func NewClusterResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Cluster{}
	}
	models := func() interface{} {
		clusters := []Cluster{}
		return &clusters
	}

	return newResource(db, clustersTable, model, models)
}
