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

package suggestion

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AnalysisPredictConfigOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
	options.BaseListOptions
	QueryType string `help:"query_type of the analysis" choices:"expense_trend" json:"query_type"`
	StartDate string `help:"start_date of the analysis" json:"start_date"`
	EndDate   string `help:"end_date of the analysis" json:"end_date"`
	DataType  string `help:"data_type of the analysis" choices:"day|month" json:"data_type"`
}

func (o *AnalysisPredictConfigOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

func (o *AnalysisPredictConfigOptions) GetId() string {
	return o.ID
}
