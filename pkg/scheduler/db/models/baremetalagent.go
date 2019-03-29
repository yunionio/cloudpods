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

type BaremetalAgent struct {
	StandaloneModel
	AccessIP   string `json:"access_ip" gorm:"not null"`
	ManagerURI string `json:"manager_uri,omitempty"`
	Status     string `json:"status" gorm:"not null"`
	ZoneID     string `json:"zone_id,omitempty"`
	Version    string `json:"version,omitempty"`
}

func (b BaremetalAgent) TableName() string {
	return baremetalAgentsTable
}

func (b BaremetalAgent) String() string {
	s, _ := JsonString(b)
	return s
}

func NewBaremetalAgentResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &BaremetalAgent{}
	}
	models := func() interface{} {
		agents := []BaremetalAgent{}
		return &agents
	}

	return newResource(db, baremetalAgentsTable, model, models)
}
