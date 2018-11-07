package models

import (
	"github.com/jinzhu/gorm"
)

type Dynamicschedtag struct {
	StandaloneModel

	Condition  string `gorm:"column:condition;not null"`
	SchedtagId string `gorm:"column:schedtag_id;not null"`
	Enabled    bool   `gorm:"column:enabled"`
}

func (d Dynamicschedtag) TableName() string {
	return dynamicschedtagTable
}

func (d Dynamicschedtag) String() string {
	s, _ := JsonString(d)
	return s
}

func NewDynaimcschedtagResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Dynamicschedtag{}
	}
	models := func() interface{} {
		tags := []Dynamicschedtag{}
		return &tags
	}

	return newResource(db, dynamicschedtagTable, model, models)
}

func FetchEnabledDynamicschedtags() ([]*Dynamicschedtag, error) {
	cond := map[string]interface{}{
		"deleted": false,
		"enabled": true,
	}
	objs, err := AllWithCond(Dynamicschedtags, cond)
	if err != nil {
		return nil, err
	}
	tags := []*Dynamicschedtag{}
	for _, obj := range objs {
		tags = append(tags, obj.(*Dynamicschedtag))
	}
	return tags, nil
}

func (d Dynamicschedtag) FetchSchedTag() (*Aggregate, error) {
	obj, err := FetchByID(Aggregates, d.SchedtagId)
	if err != nil {
		return nil, err
	}
	return obj.(*Aggregate), err
}
