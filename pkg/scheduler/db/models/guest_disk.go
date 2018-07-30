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
