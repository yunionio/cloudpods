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

package billing

import (
	"yunion.io/x/cloudmux/pkg/apis/billing"
	"yunion.io/x/cloudmux/pkg/apis/compute"
)

type TBillingType string

const (
	BILLING_TYPE_POSTPAID = TBillingType(billing.BILLING_TYPE_POSTPAID)
	BILLING_TYPE_PREPAID  = TBillingType(billing.BILLING_TYPE_PREPAID)
)

type TNetChargeType string

const (
	NET_CHARGE_TYPE_BY_TRAFFIC   = TNetChargeType(compute.EIP_CHARGE_TYPE_BY_TRAFFIC)
	NET_CHARGE_TYPE_BY_BANDWIDTH = TNetChargeType(compute.EIP_CHARGE_TYPE_BY_BANDWIDTH)
)

func ParseBillingType(s string) TBillingType {
	switch s {
	case string(BILLING_TYPE_POSTPAID):
		return BILLING_TYPE_POSTPAID
	case string(BILLING_TYPE_PREPAID):
		return BILLING_TYPE_PREPAID
	}
	return ""
}

func ParseNetChargeType(s string) TNetChargeType {
	switch s {
	case string(NET_CHARGE_TYPE_BY_TRAFFIC):
		return NET_CHARGE_TYPE_BY_TRAFFIC
	case string(NET_CHARGE_TYPE_BY_BANDWIDTH):
		return NET_CHARGE_TYPE_BY_BANDWIDTH
	}
	return ""
}
