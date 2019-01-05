package models

import (
	"github.com/jinzhu/gorm"
)

type Zone struct {
	StandaloneModel
	Status        string `json:"status" gorm:"column:status;not null"`
	Location      string `gorm:"column:location"`
	ManagerUri    string `gorm:"column:manager_uri"`
	CloudregionId string `gorm:"column:cloudregion_id"`
}

func (z Zone) TableName() string {
	return zonesTable
}

func (z Zone) String() string {
	s, _ := JsonString(z)
	return s
}

func NewZoneResource(db *gorm.DB) (Resourcer, error) {
	return newResource(db, zonesTable,
		func() interface{} {
			return &Zone{}
		},
		func() interface{} {
			zones := []Zone{}
			return &zones
		},
	)
}

func FetchZoneByID(id string) (*Zone, error) {
	zone, err := FetchByID(Zones, id)
	if err != nil {
		return nil, err
	}
	return zone.(*Zone), nil
}
