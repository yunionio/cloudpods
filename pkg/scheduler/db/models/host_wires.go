package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

const (
	HostWireResourceName = "hostwires"
)

type HostWire struct {
	StandaloneModel
	Bridge    string `json:"bridge,omitempty" gorm:"not null"`
	Interface string `json:"interface,omitempty" gorm:"not null"`
	HostID    string `json:"host_id,omitempty" gorm:"not null"`
	WireID    string `json:"wire_id,omitempty" gorm:"not null"`
}

func (w HostWire) TableName() string {
	return hostWiresTable
}

func (w HostWire) String() string {
	str, _ := JsonString(w)
	return str
}

func NewHostWiresResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &HostWire{}
	}
	models := func() interface{} {
		hostWires := []HostWire{}
		return &hostWires
	}

	return newResource(db, hostWiresTable, model, models)
}

type Host2Wire struct {
	HostID string `json:"host_id" gorm:"column:host_id;not null"`
	WireID string `json:"wire_id" gorm:"column:wire_id;not null"`
}

func (c Host2Wire) First() string {
	return c.WireID
}

func SelectWiresWithHostID(hostID string) ([]Host2Wire, error) {
	wires := []Host2Wire{}
	err := HostWires.DB().Table(hostWiresTable).
		Select("distinct wire_id").
		Where(fmt.Sprintf("host_id = '%s' and deleted=0", hostID)).
		Scan(&wires).Error

	return wires, err
}
func SelectHostHasWires() ([]Host2Wire, error) {
	wires := []Host2Wire{}
	err := HostWires.DB().Table(hostWiresTable).
		Select("host_id,wire_id").
		Where("deleted=0").
		Scan(&wires).Error

	return wires, err
}
