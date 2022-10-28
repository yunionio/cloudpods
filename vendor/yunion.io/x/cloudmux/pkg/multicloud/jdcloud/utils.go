// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jdcloud

import (
	"time"

	"github.com/jdcloud-api/jdcloud-sdk-go/services/charge/models"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
)

func parseTime(ts string) time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05Z", ts)
	return t
}

func billingType(charge *models.Charge) string {
	switch charge.ChargeMode {
	case "prepaid_by_duration":
		return billing_api.BILLING_TYPE_PREPAID
	case "postpaid_by_usage", "postpaid_by_duration":
		return billing_api.BILLING_TYPE_POSTPAID
	default:
		return ""
	}
}

func expireAt(charge *models.Charge) time.Time {
	if billingType(charge) == billing_api.BILLING_TYPE_POSTPAID {
		return time.Time{}
	}
	return parseTime(charge.ChargeExpiredTime)
}
