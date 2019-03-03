package aliyun

import (
	"time"

	"yunion.io/x/onecloud/pkg/compute/models"
)

func convertChargeType(ct TChargeType) string {
	switch ct {
	case PrePaidInstanceChargeType:
		return models.BILLING_TYPE_PREPAID
	case PostPaidInstanceChargeType:
		return models.BILLING_TYPE_POSTPAID
	default:
		return models.BILLING_TYPE_PREPAID
	}
}

func convertExpiredAt(expired time.Time) time.Time {
	if !expired.IsZero() {
		now := time.Now()
		if expired.Sub(now) < time.Hour*24*365*6 {
			return expired
		}
	}
	return time.Time{}
}
