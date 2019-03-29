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

type Cluster struct {
	StandaloneModel
	HostIPStart  string `json:"host_ip_start" gorm:"not null"`
	HostIPEnd    string `json:"host_ip_end" gorm:"not null"`
	HostNetmask  int    `json:"host_netmask,omitempty"`
	HostGateway  string `json:"host_gateway,omitempty"`
	HostDNS      string `json:"host_dns,omitempty"`
	ScheduleRank int    `json:"schedule_rank,omitempty"`
	ZoneID       string `json:"zone_id" gorm:"not null"`
}

func (c Cluster) TableName() string {
	return clustersTable
}

func (c Cluster) String() string {
	s, _ := json.Marshal(c)
	return string(s)
}

func NewClusterResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Cluster{}
	}
	models := func() interface{} {
		clusters := []Cluster{}
		return &clusters
	}

	return newResource(db, clustersTable, model, models)
}
