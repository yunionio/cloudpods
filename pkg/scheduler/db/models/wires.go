package models

import (
	"github.com/jinzhu/gorm"
)

const (
	NetWireResourceName = "wire"
)

type Wire struct {
	StandaloneModel
	Bandwidth  int64  `json:"bandwidth,omitempty" gorm:"not null"`
	NetDns     string `json:"net_dns,omitempty"`
	NetDomain  string `json:"net_domain,omitempty"`
	VpcVersion int64  `json:"vpc_version,omitempty" gorm:"not null"`
}

func (w Wire) TableName() string {
	return wiresTable
}

func (w Wire) String() string {
	str, _ := JsonString(w)
	return str
}

func NewWiresResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Wire{}
	}
	models := func() interface{} {
		wires := []Wire{}
		return &wires
	}

	return newResource(db, wiresTable, model, models)
}

type WireInfo struct {
	ID   string `json:"id" gorm:"column:id;not null"`
	Name string `json:"name" gorm:"column:name;not null"`
}

func (i WireInfo) First() string {
	return i.ID
}

func (i WireInfo) Second() string {
	return i.Name
}
func LoadAllWires() ([]WireInfo, error) {
	wires := []WireInfo{}
	err := Wires.DB().Table(wiresTable).
		Select("id,name").
		Where("deleted=0").
		Scan(&wires).Error

	return wires, err
}
