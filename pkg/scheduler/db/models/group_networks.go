package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

const (
	GroupNetworksResourceName = "groupnetworks"
)

type GroupNetwork struct {
	StandaloneModel
	GroupID       string `json:"group_id,omitempty" gorm:"column:group_id;not null"`
	NetworkID     string `json:"network_id,omitempty" gorm:"column:network_id;not null"`
	IpAddr        string `json:"ip_addr,omitempty" gorm:"column:ip_addr"`
	Index         int    `json:"index" gorm:"column:index;not null"`
	EipID         string `json:"eip_id,omitempty" gorm:"column:if_name"`
	MappingIpAddr string `json:"mapping_ip_addr" gorm:"column:mapping_ip_addr"`
}

func (n GroupNetwork) TableName() string {
	return groupNetworksTable
}

func (n GroupNetwork) String() string {
	s, _ := JsonString(n)
	return string(s)
}

func NewGroupNetworksResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &GroupNetwork{}
	}
	models := func() interface{} {
		groupNetworks := []GroupNetwork{}
		return &groupNetworks
	}

	return newResource(db, groupNetworksTable, model, models)
}

type GroupNicCount struct {
	NetworkID string `json:"network_id,omitempty" gorm:"column:network_id;not null"`
	Count     int    `json:"count" gorm:"column:count;not null"`
}

func (c GroupNicCount) First() string {
	return c.NetworkID
}

func (c GroupNicCount) Second() int {
	return c.Count
}
func GroupNicCounts() ([]GroupNicCount, error) {
	counts := []GroupNicCount{}

	err := Groups.DB().Table(groupNetworksTable).
		Select("network_id,count(*) as count").
		Where("deleted=0").
		Group("network_id").
		Scan(&counts).Error
	return counts, err
}

type GroupNicCounti struct {
	Count int `json:"count" gorm:"column:count;not null"`
}

func (c GroupNicCounti) First() int {
	return c.Count
}
func GroupNicCountsWithNetworkID(networkID string) (GroupNicCounti, error) {
	counts := GroupNicCounti{0}

	err := Groups.DB().Table(groupNetworksTable).
		Select("count(*) as count").
		Where(fmt.Sprintf("network_id = '%s' and deleted=0", networkID)).
		Scan(&counts).Error
	return counts, err
}
