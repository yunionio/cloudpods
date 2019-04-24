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
	"fmt"

	"github.com/jinzhu/gorm"
)

const (
	ReserveDipsResourceName = "reservedips"
)

type ReserveDipNetwork struct {
	StandaloneModel
	NetworkID string `json:"network_id,omitempty" gorm:"column:network_id;not null"`
	IpAddr    string `json:"ip_addr,omitempty" gorm:"column:ip_addr"`
	Notes     string `json:"notes" gorm:"column:notes"`
}

func (n ReserveDipNetwork) TableName() string {
	return reserveDipsTable
}

func (n ReserveDipNetwork) String() string {
	s, _ := JsonString(n)
	return string(s)
}

func NewReserveDipsNetworksResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &ReserveDipNetwork{}
	}
	models := func() interface{} {
		reserveDips := []ReserveDipNetwork{}
		return &reserveDips
	}

	return newResource(db, reserveDipsTable, model, models)
}

type ReserveNicCount struct {
	NetworkID string `json:"network_id,omitempty" gorm:"column:network_id;not null"`
	Count     int    `json:"count" gorm:"column:count;not null"`
}

func (c ReserveNicCount) First() string {
	return c.NetworkID
}

func (c ReserveNicCount) Second() int {
	return c.Count
}
func ReserveNicCounts() ([]ReserveNicCount, error) {
	counts := []ReserveNicCount{}
	err := ReserveDipsNerworks.DB().Table(reserveDipsTable).
		Select("network_id,count(*) as count").
		Where("deleted=0").
		Group("network_id").
		Scan(&counts).Error
	return counts, err
}

type ReserveNicCounti struct {
	Count int `json:"count" gorm:"column:count;not null"`
}

func (c ReserveNicCounti) First() int {
	return c.Count
}
func ReserveNicCountsWithNetworkID(networkID string) (ReserveNicCounti, error) {
	counts := ReserveNicCounti{0}
	err := ReserveDipsNerworks.DB().Table(reserveDipsTable).
		Select("count(*) as count").
		Where(fmt.Sprintf("network_id = '%s' and deleted=0", networkID)).
		Scan(&counts).Error
	return counts, err
}
