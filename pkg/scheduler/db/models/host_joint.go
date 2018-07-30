package models

type HostJointModel struct {
	JointBaseModel
	HostID string `json:"host_id" gorm:"not null"`
}
