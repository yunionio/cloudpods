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

type AggregateHost struct {
	JointBaseModel
	HostID      string `json:"host_id" gorm:"column:host_id;not null"`
	AggregateID string `json:"schedtag_id" gorm:"column:schedtag_id;not null"`
}

func (c AggregateHost) TableName() string {
	return aggregateHostsTable
}

func (c AggregateHost) String() string {
	s, _ := json.Marshal(c)
	return string(s)
}

func (c AggregateHost) Aggregate() (*Aggregate, error) {
	a, err := FetchByID(Aggregates, c.AggregateID)
	if err != nil {
		return nil, err
	}
	return a.(*Aggregate), nil
}

func NewAggregateHostResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &AggregateHost{}
	}
	models := func() interface{} {
		aggregate_hosts := []AggregateHost{}
		return &aggregate_hosts
	}

	return newResource(db, aggregateHostsTable, model, models)
}
