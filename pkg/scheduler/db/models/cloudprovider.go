package models

import (
	"github.com/jinzhu/gorm"
)

type Cloudprovider struct {
	StandaloneModel
	Status         string `json:"status" gorm:"column:status;not null"`
	Enabled        bool   `json:"enabled" gorm:"column:enabled;not null"`
	AccessUrl      string `json:"access_url" gorm:"column:access_url"`
	Provider       string `json:"provider" gorm:"column:provider"`
	CloudaccountId string `json:"cloudaccount_id" gorm:"column:cloudaccount_id"`
	ProjectId      string `json:"tenant_id" gorm:"column:tenant_id"`
}

func (c Cloudprovider) TableName() string {
	return cloudproviderTable
}

func (c Cloudprovider) String() string {
	s, _ := JsonString(c)
	return s
}

func NewCloudproviderResource(db *gorm.DB) (Resourcer, error) {
	return newResource(db, cloudproviderTable,
		func() interface{} {
			return &Cloudprovider{}
		},
		func() interface{} {
			cs := []Cloudprovider{}
			return &cs
		})
}

func FetchCloudproviderById(id string) (*Cloudprovider, error) {
	obj, err := FetchByID(CloudProviders, id)
	if err != nil {
		return nil, err
	}
	return obj.(*Cloudprovider), nil
}
