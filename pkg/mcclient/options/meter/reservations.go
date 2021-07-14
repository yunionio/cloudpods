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

package meter

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ReservationListOptions struct {
	options.BaseListOptions

	CloudaccountId   string `help:"cloudaccount id of reservation" json:"cloudaccount_id"`
	ResourceType     string `help:"resource type of reservation" json:"resource_type"`
	ReservationYears string `help:"number of reservation years" json:"reservation_years"`
	LookbackDays     string `help:"number of previous days will be consider" json:"lookback_days"`
	PaymentOption    string `help:"payment option of reservation, example:all_upfront/partial_upfront/no_upfront" json:"payment_option"`
	OfferingClass    string `help:"offering class of reservation, example:standard/convertible" json:"offering_class"`
}

func (opt *ReservationListOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}
