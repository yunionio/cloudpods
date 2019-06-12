package multicloud

import "time"

type SBillingBase struct{}

func (self *SBillingBase) GetBillingType() string {
	return ""
}

func (self *SBillingBase) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SBillingBase) GetExpiredAt() time.Time {
	return time.Time{}
}
