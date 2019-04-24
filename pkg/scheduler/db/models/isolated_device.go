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

type IsolatedDevice struct {
	StandaloneModel
	HostID         string `json:"host_id,omitempty" gorm:"column:host_id;not null"`
	DevType        string `json:"dev_type" gorm:"column:dev_type;not null"`
	Model          string `json:"model" gorm:"column:model;not null"`
	GuestID        string `json:"guest_id" gorm:"column:guest_id"`
	Addr           string `json:"addr" gorm:"column:addr"`
	VendorDeviceID string `json:"vendor_device_id" gorm:"column:vendor_device_id"`
}

func (d IsolatedDevice) TableName() string {
	return isolatedDeviceTable
}

func (d IsolatedDevice) String() string {
	str, _ := JsonString(d)
	return str
}

func NewIsolatedDeviceResource(db *gorm.DB) (Resourcer, error) {
	return newResource(db, isolatedDeviceTable,
		func() interface{} { return &IsolatedDevice{} },
		func() interface{} { return &([]IsolatedDevice{}) })
}
