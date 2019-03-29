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
