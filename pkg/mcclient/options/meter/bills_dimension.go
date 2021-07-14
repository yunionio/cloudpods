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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type BillDimensionCreateOptions struct {
	apis.StatusStandaloneResourceCreateInput

	DimensionAnalysisIds []string `json:"dimension_analysis_ids" help:"ids of DimensionAnalysis"`
	DimensionItemName    string   `json:"dimension_item_name" help:"name of dimension item" required:"true"`
	DimensionType        string   `json:"dimension_type" choices:"bill|resource_type"`
}

func (o *BillDimensionCreateOptions) Params() (jsonutils.JSONObject, error) {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(o.Name), "name")
	data.Add(jsonutils.NewString(o.DimensionType), "dimension_type")

	dimensionItem := jsonutils.NewDict()
	dimensionItem.Add(jsonutils.NewString(o.DimensionItemName))
	dimensionItem.Add(jsonutils.NewStringArray(o.DimensionAnalysisIds), "dimension_analysis_ids")

	data.Add(jsonutils.NewArray(dimensionItem), "dimension_items")
	return data, nil

}

type BillDimensionListOptions struct {
	options.BaseListOptions
	apis.EnabledResourceBaseListInput

	DimensionType string `json:"dimension_type"`
}

func (o *BillDimensionListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type BillDimensionShowOptions struct {
	ID string `help:"ID of bill dimension " json:"-"`
}

func (o *BillDimensionShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *BillDimensionShowOptions) GetId() string {
	return o.ID
}

type BillDimensionDeleteOptions struct {
	ID string `json:"-"`
}

func (o *BillDimensionDeleteOptions) GetId() string {
	return o.ID
}

func (o *BillDimensionDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type BillDimensionAnalysisListOptions struct {
	options.BaseListOptions
	apis.EnabledResourceBaseListInput

	UsageType    string `json:"usage_type"`
	ResourceType string `json:"resource_type"`
	Description  string `json:"description"`
}

func (o *BillDimensionAnalysisListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type BillDimensionAnalysisShowOptions struct {
	ID string `help:"ID of bill dimension " json:"-"`
}

func (o *BillDimensionAnalysisShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *BillDimensionAnalysisShowOptions) GetId() string {
	return o.ID
}
