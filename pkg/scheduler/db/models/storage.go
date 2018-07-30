package models

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"

	o "github.com/yunionio/onecloud/cmd/scheduler/options"
)

type Storage struct {
	StandaloneModel
	Capacity    int64    `json:"capacity" gorm:"not null"`
	StorageType string   `json:"storage_type" gorm:"not null"`
	MediumType  string   `json:"medium_type" gorm:"not null"`
	Cmtbound    *float64 `json:"cmtbound"`
	Status      string   `json:"status" gorm:"not null"`
	StorageConf string   `json:"storage_conf" gorm:"type:text"`
	ZoneID      string   `json:"zone_id"`
}

func (s Storage) TableName() string {
	return storageTable
}

func (s Storage) String() string {
	str, _ := JsonString(s)
	return str
}

func NewStorageResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Storage{}
	}
	models := func() interface{} {
		storages := []Storage{}
		return &storages
	}

	return newResource(db, storageTable, model, models)
}

type StorageCapacity struct {
	StorageID string `json:"storage_id" gorm:"column:storage_id;not null"`
	Status    string `json:"status" gorm:"not null"`
	TotalSize int64  `json:"total_size" gorm:"column:total_size"`
}

func (s StorageCapacity) First() string {
	return s.StorageID
}

func (s StorageCapacity) Second() string {
	return s.Status
}

func (s StorageCapacity) Third() interface{} {
	return s.TotalSize
}

func GetStorageCapacities(storageIDs []string) ([]StorageCapacity, error) {
	results := make([]StorageCapacity, 0)
	err := Disks.DB().Table(disksTable).
		Select("storage_id, status, sum(disk_size) as total_size").
		Where(fmt.Sprintf("storage_id in ('%s') and deleted=0", strings.Join(storageIDs, "','"))).
		Group("storage_id, status").Scan(&results).Error
	return results, err
}

func (s Storage) OverCommitBound() float64 {
	if s.Cmtbound != nil {
		return *s.Cmtbound
	}
	return float64(o.GetOptions().DefaultStorageOvercommitBound)
}
