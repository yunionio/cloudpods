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

type BillingExchangeRateListOptions struct {
	options.BaseListOptions
}

func (opt *BillingExchangeRateListOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type BillingExchangeRateUpdateOptions struct {
	ID string `help:"ID of billing exchange rate" json:"-"`

	Rate float64 `help:"exchange rate" json:"rate"`
}

func (opt *BillingExchangeRateUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *BillingExchangeRateUpdateOptions) GetId() string {
	return opt.ID
}
