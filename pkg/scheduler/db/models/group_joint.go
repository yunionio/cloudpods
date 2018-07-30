package models

type GroupJointModel struct {
	JointBaseModel
	GroupID string `json:"group_id" gorm:"not null"`
}
