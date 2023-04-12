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

package excelutils // import "yunion.io/x/onecloud/pkg/util/excelutils"

type ExcelChartType string

const (
	CHART_TYPE_LINE    ExcelChartType = "line"    // 折线图
	CHART_TYPE_PIE     ExcelChartType = "pie"     // 饼图
	CHART_TYPE_RADAR   ExcelChartType = "radar"   // 雷达图
	CHART_TYPE_SCATTER ExcelChartType = "scatter" // 散点图
	CHART_TYPE_COL     ExcelChartType = "col"     // 柱状图
)

const (
	DEFAULT_SHEET = "Sheet1"
)

var ChartMap = map[ExcelChartType]string{
	CHART_TYPE_LINE:    "折线图",
	CHART_TYPE_PIE:     "饼图",
	CHART_TYPE_RADAR:   "雷达图",
	CHART_TYPE_SCATTER: "散点图",
	CHART_TYPE_COL:     "柱状图",
}

const (
	// 千位分隔
	CELL_STYLE_THOUSANDS_SEPARATOR string = `#,##0.00_);[Red]\( #,##0.00\)`
)
