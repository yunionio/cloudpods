package models

type SScalingGroupGuestManager struct {
	SGroupJointsManager
}

var ScalingGroupGuestManager *SScalingGroupGuestManager

type SScalingGroupGuest struct {
	SGroupJointsBase

	ScalingGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	GuestStatus    string `width:"36" charset:"ascii" nullable:"false"`
}
