package models

import (
	"encoding/json"

	"github.com/jinzhu/gorm"
)

type Aggregate struct {
	StandaloneModel
	DefaultStrategy string `json:"default_strategy" gorm:"not null"`
}

func (c Aggregate) TableName() string {
	return aggregatesTable
}

func (c Aggregate) String() string {
	s, _ := json.Marshal(c)
	return string(s)
}

func NewAggregateResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Aggregate{}
	}
	models := func() interface{} {
		aggregates := []Aggregate{}
		return &aggregates
	}

	return newResource(db, aggregatesTable, model, models)
}
