package models

import (
	"time"

	"yunion.io/x/jsonutils"
)

const (
	BILLING_TYPE_POSTPAID = "postpaid"
	BILLING_TYPE_PREPAID  = "prepaid"
)

type SBillingResourceBase struct {
	BillingType  string    `width:"36" charset:"ascii" nullable:"true" default:"postpaid" list:"user" create:"admin_optional"`
	ExpiredAt    time.Time `nullable:"true" list:"user" create:"admin_optional"`
	BillingCycle string    `width:"10" charset:"ascii" nullable:"true" list:"user" create:"admin_optional"`
}

func (self *SBillingResourceBase) GetChargeType() string {
	if len(self.BillingType) > 0 {
		return self.BillingType
	} else {
		return BILLING_TYPE_POSTPAID
	}
}

func (self *SBillingResourceBase) GetBillingShortDesc() jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	if self.GetChargeType() == BILLING_TYPE_PREPAID {
		ret.Add(jsonutils.NewTimeString(self.ExpiredAt), "expired_at")
		ret.Add(jsonutils.NewString(self.BillingCycle), "billing_cycle")
	}
	return ret
}

func (self *SBillingResourceBase) IsValidPrePaid() bool {
	if self.BillingType == BILLING_TYPE_PREPAID {
		now := time.Now().UTC()
		if self.ExpiredAt.After(now) {
			return true
		}
	}
	return false
}
