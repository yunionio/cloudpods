package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

const (
	GuestNetworksResourceName = "guestnetworks"
)

type GuestNetwork struct {
	StandaloneModel
	GuestID       string `json:"guest_id,omitempty" gorm:"column:guest_id;not null"`
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

func (n GuestNetwork) TableName() string {
	return guestNetworksTable
}

func (n GuestNetwork) String() string {
	s, _ := JsonString(n)
	return string(s)
}

func NewGuestNetworksResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &GuestNetwork{}
	}
	models := func() interface{} {
		guestNetworks := []GuestNetwork{}
		return &guestNetworks
	}

	return newResource(db, guestNetworksTable, model, models)
}

type GuestNicCount struct {
	NetworkID string `json:"network_id,omitempty" gorm:"column:network_id;not null"`
	Count     int    `json:"count" gorm:"column:count;not null"`
}

func (c GuestNicCount) First() string {
	return c.NetworkID
}

func (c GuestNicCount) Second() int {
	return c.Count
}

func GuestNicCounts() ([]GuestNicCount, error) {
	counts := []GuestNicCount{}
	err := GuestNetworks.DB().Table(guestNetworksTable).
		Select("network_id,count(*) as count").
		Where("deleted=0").
		Group("network_id").
		Scan(&counts).Error
	return counts, err
}

type GuestNicCounti struct {
	Count int `json:"count" gorm:"column:count;not null"`
}

func (c GuestNicCounti) First() int {
	return c.Count
}
func GuestNicCountsWithNetworkID(networkID string) (GuestNicCounti, error) {
	counts := GuestNicCounti{0}
	err := GuestNetworks.DB().Table(guestNetworksTable).
		Select("count(*) as count").
		Where(fmt.Sprintf("network_id = '%s' and deleted=0", networkID)).
		Scan(&counts).Error
	return counts, err
}
