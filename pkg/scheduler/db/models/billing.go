package models

import (
	"time"
)

type BillingResourceBase struct {
	BillingType  string    `json:"billing_type" gorm:"column:billing_type"`
	ExpiredAt    time.Time `json:"expired_at" gorm:"column:expired_at;type:datetime"`
	BillingCycle string    `json:"billing_cycle" gorm:"column:billing_cycle"`
}
