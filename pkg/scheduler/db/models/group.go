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
