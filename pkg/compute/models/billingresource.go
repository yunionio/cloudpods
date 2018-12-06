package models

import (
	"time"
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

type SCloudBillingInfo struct {
	Provider           string    `json:",omitempty"`
	Account            string    `json:",omitempty"`
	AccountId          string    `json:",omitempty"`
	Manager            string    `json:",omitempty"`
	ManagerId          string    `json:",omitempty"`
	ManagerProject     string    `json:",omitempty"`
	ManagerProjectId   string    `json:",omitempty"`
	Region             string    `json:",omitempty"`
	RegionId           string    `json:",omitempty"`
	RegionExtId        string    `json:",omitempty"`
	Zone               string    `json:",omitempty"`
	ZoneId             string    `json:",omitempty"`
	ZoneExtId          string    `json:",omitempty"`
	PriceKey           string    `json:",omitempty"`
	ChargeType         string    `json:",omitempty"`
	InternetChargeType string    `json:",omitempty"`
	ExpiredAt          time.Time `json:",omitempty"`
	BillingCycle       string    `json:",omitempty"`
}

type SCloudBillingInfo struct {
	SCloudProviderInfo

	SBillingBaseInfo

	PriceKey           string `json:",omitempty"`
	InternetChargeType string `json:",omitempty"`
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
