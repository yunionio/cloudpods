package models

import (
	"github.com/jinzhu/gorm"
)

type GroupGuest struct {
	GroupJointModel
	Tag     *string `json:"tag" gorm:"column:tag"`
	GuestID *string `json:"guest_id" gorm:"column:guest_id"`
}

func (g GroupGuest) TableName() string {
	return groupGuestTable
}

func (g GroupGuest) String() string {
	str, _ := JsonString(g)
	return str
}

func NewGroupGuestResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &GroupGuest{}
	}
	models := func() interface{} {
		groupGuests := []GroupGuest{}
		return &groupGuests
	}

	return newResource(db, groupGuestTable, model, models)
}
