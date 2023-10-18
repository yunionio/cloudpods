// Copyright 2023 Yunion
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

package volcengine

import (
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/billing"
)

const (
	PrePaidInstanceChargeType  TChargeType = "PrePaid"
	PostPaidInstanceChargeType TChargeType = "PostPaid"
	DefaultInstanceChargeType              = PostPaidInstanceChargeType
)

func convertChargeType(ct TChargeType) string {
	switch ct {
	case PrePaidInstanceChargeType:
		return api.BILLING_TYPE_PREPAID
	case PostPaidInstanceChargeType:
		return api.BILLING_TYPE_POSTPAID
	default:
		return ""
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
