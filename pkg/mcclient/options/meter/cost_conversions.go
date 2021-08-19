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

type CostConversionListOptions struct {
	options.BaseListOptions
}

func (opt *CostConversionListOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type CostConversionCreateOptions struct {
	Name string

	IsPublicCloud   string `help:"public cloud filter of cost conversion" json:"is_public_cloud"`
	Brand           string `help:"brand filter of cost conversion" json:"brand"`
	CloudaccountId  string `help:"cloudaccount filter of cost conversion" json:"cloudaccount_id"`
	CloudproviderId string `help:"cloudprovider filter of cost conversion" json:"cloudprovider_id"`
	DomainIdFilter  string `help:"domain filter of cost conversion" json:"domain_id_filter"`

	EnableDate string  `help:"enable date of conversion ratio, example:202107" json:"enable_date"`
	Ratio      float64 `help:"cost conversion ratio" json:"ratio"`
}

func (opt *CostConversionCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type CostConversionUpdateOptions struct {
	ID string `help:"ID of cost conversion" json:"-"`

	Ratio float64 `help:"cost conversion ratio" json:"ratio"`
}

func (opt *CostConversionUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *CostConversionUpdateOptions) GetId() string {
	return opt.ID
}

type CostConversionDeleteOptions struct {
	ID string `help:"ID of cost conversion" json:"-"`
}

func (opt *CostConversionDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *CostConversionDeleteOptions) GetId() string {
	return opt.ID
}
