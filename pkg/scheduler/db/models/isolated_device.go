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
