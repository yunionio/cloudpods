package billing

import "time"

type BillingDetailsInfo struct {
}

type BillingResourceListInput struct {
	// 计费类型，按需计费和预付费
	// pattern:prepaid|postpaid
	BillingType string `json:"billing_type"`

	// 计费过期时间的查询起始时间
	BillingExpireSince time.Time `json:"billing_expire_since"`
	// 计费过期时间的查询终止时间
	BillingExpireBefore time.Time `json:"billing_expire_before"`

	// 计费周期
	// example:7d
	BillingCycle string `json:"billing_cycle"`
}
