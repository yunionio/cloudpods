// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
