package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

const (
	BaremetalNetworksResourceName = "baremetalnetworks"
)

type BaremetalNetwork struct {
	StandaloneModel
	BaremetalID   string `json:"baremetal_id,omitempty" gorm:"column:Baremetal_id;not null"`
	NetworkID     string `json:"network_id,omitempty" gorm:"column:network_id;not null"`
	MacAddr       string `json:"mac_addr" gorm:"column:mac_addr;not null"`
	IpAddr        string `json:"ip_addr,omitempty" gorm:"column:ip_addr"`
	Ip6Addr       string `json:"ip6_addr" gorm:"column:ip6_addr"`
	Driver        string `json:"driver" gorm:"column:driver"`
	BwLimit       int64  `json:"bw_limit" gorm:"column:bw_limit;not null"`
	Index         int    `json:"index" gorm:"column:index;not null"`
	Virtual       int    `json:"virtual" gorm:"column:virtual"`
	IfName        string `json:"if_name,omitempty" gorm:"column:if_name"`
	MappingIpAddr string `json:"mapping_ip_addr" gorm:"column:mapping_ip_addr"`
}

func (n BaremetalNetwork) TableName() string {
	return baremetalNetworksTable
}

func (n BaremetalNetwork) String() string {
	s, _ := JsonString(n)
	return string(s)
}

func NewBaremetalNetworksResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &BaremetalNetwork{}
	}
	models := func() interface{} {
		baremetalNetworks := []BaremetalNetwork{}
		return &baremetalNetworks
	}

	return newResource(db, baremetalNetworksTable, model, models)
}

type BaremetalNicCount struct {
	NetworkID string `json:"network_id,omitempty" gorm:"column:network_id;not null"`
	Count     int    `json:"count" gorm:"column:count;not null"`
}

func (c BaremetalNicCount) First() string {
	return c.NetworkID
}

func (c BaremetalNicCount) Second() int {
	return c.Count
}
func BaremetalNicCounts() ([]BaremetalNicCount, error) {
	counts := []BaremetalNicCount{}
	err := BaremetalNetworks.DB().Table(baremetalNetworksTable).
		Select("network_id,count(*) as count").
		Where("deleted=0").
		Group("network_id").
		Scan(&counts).Error
	return counts, err
}

type BaremetalNicCounti struct {
	Count int `json:"count" gorm:"column:count;not null"`
}

func (c BaremetalNicCounti) First() int {
	return c.Count
}
func BaremetalNicCountsWithNetworkID(networkID string) (BaremetalNicCounti, error) {
	counts := BaremetalNicCounti{0}
	err := BaremetalNetworks.DB().Table(baremetalNetworksTable).
		Select("count(*) as count").
		Where(fmt.Sprintf("network_id = '%s' and deleted=0", networkID)).
		Scan(&counts).Error
	return counts, err
}
