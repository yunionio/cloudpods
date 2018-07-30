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
