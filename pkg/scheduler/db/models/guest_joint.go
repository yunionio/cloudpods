package models

type GuestJointModel struct {
	JointBaseModel
	GuestID string `json:"guest_id" gorm:"column:guest_id;not null;index"`
}
