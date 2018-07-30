package models

import (
	"github.com/jinzhu/gorm"
)

type Group struct {
	VirtualResourceModel
	ServiceType   string `json:"service_type" gorm:"column:service_type"`
	ParentID      string `json:"parent_id" gorm:"column:parent_id"`
	ZoneID        string `json:"zone_id" gorm:"column:zone_id"`
	SchedStrategy string `json:"sched_strategy" gorm:"column:sched_strategy"`
}

func (g Group) TableName() string {
	return groupTable
}

func (g Group) String() string {
	str, _ := JsonString(g)
	return str
}

func NewGroupResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Group{}
	}
	models := func() interface{} {
		groups := []Group{}
		return &groups
	}
	return newResource(db, groupTable, model, models)
}
