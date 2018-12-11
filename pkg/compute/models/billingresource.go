package models

import (
	"time"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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

func (self *SBillingResourceBase) FetchCloudBillingInfo(info *SCloudBillingInfo) {
	info.ChargeType = self.GetChargeType()
	if self.GetChargeType() == BILLING_TYPE_PREPAID {
		info.ExpiredAt = self.ExpiredAt
		info.BillingCycle = self.BillingCycle
	}
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

func MakeCloudBillingInfo(region *SCloudregion, zone *SZone, provider *SCloudprovider) SCloudBillingInfo {
	info := SCloudBillingInfo{}

	if zone != nil {
		info.Zone = zone.GetName()
		info.ZoneId = zone.GetId()
	}

	if region != nil {
		info.Region = region.GetName()
		info.RegionId = region.GetId()
	}

	if provider != nil {
		info.Manager = provider.GetName()
		info.ManagerId = provider.GetId()

		if len(provider.ProjectId) > 0 {
			info.ManagerProjectId = provider.ProjectId
			tc, err := db.TenantCacheManager.FetchTenantById(appctx.Background, provider.ProjectId)
			if err == nil {
				info.ManagerProject = tc.GetName()
			}
		}

		account := provider.GetCloudaccount()
		info.Account = account.GetName()
		info.AccountId = account.GetId()

		driver, err := provider.GetDriver()

		if err == nil {
			info.Provider = driver.GetId()

			if region != nil {
				iregion, err := driver.GetIRegionById(region.ExternalId)
				if err == nil {
					info.RegionExtId = iregion.GetId()
					if zone != nil {
						izone, err := iregion.GetIZoneById(zone.ExternalId)
						if err == nil {
							info.ZoneExtId = izone.GetId()
						}
					}
				}
			}
		}
	}

	return info
}
