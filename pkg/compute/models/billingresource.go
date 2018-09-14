package models

import "time"

const (
	BILLING_TYPE_POSTPAID = "postpaid"
	BILLING_TYPE_PREPAID = "prepaid"
)

type SBillingResourceBase struct {
	BillingType string `width:"36" charset:"ascii" nullable:"true" default:"postpaid" list:"user" create:"required"`
	ExpiredAt time.Time `nullable:"true" list:"user" create:"optional"`
}

func (self *SBillingResourceBase) GetChargeType() string {
	if len(self.BillingType) > 0 {
		return self.BillingType
	} else {
		return BILLING_TYPE_POSTPAID
	}
}
