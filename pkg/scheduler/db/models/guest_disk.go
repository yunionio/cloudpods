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

type GuestDisk struct {
	GuestJointModel
	DiskID    string `json:"disk_id" gorm:"column:disk_id;not null;index"`
	ImagePath string `json:"image_path" gorm:"column:image_path;not null"`
	Driver    string `json:"driver" gorm:"column:driver"`
	CacheMode string `json:"cache_mode" gorm:"column:cache_mode"`
	AioMode   string `json:"aio_mode" gorm:"column:aio_mode"`
	Index     int    `json:"index" gorm:"column:index;not null"`
}

func (d GuestDisk) TableName() string {
	return guestDiskTable
}

func (d GuestDisk) String() string {
	str, _ := JsonString(d)
	return str
}

func (d GuestDisk) Disk() (*Disk, error) {
	disk, err := FetchByID(Disks, d.DiskID)
	if err != nil {
		return nil, err
	}
	return disk.(*Disk), nil
}

func NewGuestDiskResource(db *gorm.DB) (Resourcer, error) {
	return newResource(db, guestDiskTable,
		func() interface{} { return &GuestDisk{} },
		func() interface{} { return &([]GuestDisk{}) })
}
