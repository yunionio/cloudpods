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

type HostStorage struct {
	HostJointModel
	MountPoint string `json:"mount_point" gorm:"not null"`
	StorageID  string `json:"storage_id" gorm:"not null"`
}

func (s HostStorage) TableName() string {
	return hostStorageTable
}

func (s HostStorage) String() string {
	str, _ := JsonString(s)
	return str
}

func NewHostStorageResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &HostStorage{}
	}
	models := func() interface{} {
		storages := []HostStorage{}
		return &storages
	}

	return newResource(db, hostStorageTable, model, models)
}
