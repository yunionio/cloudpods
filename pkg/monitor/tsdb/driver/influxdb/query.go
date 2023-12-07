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

package influxdb

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

var (
	regexpOperatorPattern    = regexp.MustCompile(`^\/.*\/$`)
	regexpMeasurementPattern = regexp.MustCompile(`^\/.*\/$`)
)

func (query *Query) Build(queryCtx *tsdb.TsdbQuery) (string, error) {
	var res string
	res = query.renderSelectors(queryCtx)
	res += query.renderMeasurement()
	res += query.renderWhereClause()
	res += query.renderTimeFilter(queryCtx)
	res += query.renderGroupBy(queryCtx)
	res += query.renderTz()

	calculator := tsdb.NewIntervalCalculator(&tsdb.IntervalOptions{})
	interval := calculator.Calculate(queryCtx.TimeRange, query.Interval)

	res = strings.Replace(res, "$timeFilter", query.renderTimeFilter(queryCtx), -1)
	res = strings.Replace(res, "$interval", interval.Text, -1)
	res = strings.Replace(res, "$__interval_ms", strconv.FormatInt(interval.Milliseconds(), 10), -1)
	res = strings.Replace(res, "$__interval", interval.Text, -1)
	return res, nil
}

func (query *Query) renderTags() []string {
	var res []string
	for i, tag := range query.Tags {
		str := ""

		if i > 0 {
			if tag.Condition == "" {
				str += "AND"
			} else {
				str += tag.Condition
			}
			str += " "
		}

		// If the operator is missing we fall back to sensible defaults
		if tag.Operator == "" {
			if regexpOperatorPattern.Match([]byte(tag.Value)) {
				tag.Operator = "=~"
			} else {
				tag.Operator = "="
			}
		}

		// quote value unless regex or number
		var textValue string
		if tag.Operator == "=~" || tag.Operator == "!~" {
			textValue = tag.Value
		} else if tag.Operator == "<" || tag.Operator == ">" {
			textValue = tag.Value
		} else {
			if utils.IsInStringArray(tag.Value, []string{"true", "false"}) {
				textValue = tag.Value
			} else {
				textValue = fmt.Sprintf("'%s'", strings.Replace(tag.Value, `\`, `\\`, -1))
			}
		}
		textValue = query.renderTagValue(textValue)
		res = append(res, fmt.Sprintf(`%s"%s" %s %s`, str, tag.Key, tag.Operator, textValue))
	}

	return res
}

func (query *Query) renderTagValue(val string) string {
	return strings.ReplaceAll(val, " ", "+")
}

func (query *Query) renderTimeFilter(queryCtx *tsdb.TsdbQuery) string {
	from := ""
	if strings.Contains(queryCtx.TimeRange.From, "now-") {
		from = "now() - " + strings.Replace(queryCtx.TimeRange.From, "now-", "", 1)
	} else {
		if _, ok := tsdb.TryParseUnixMsEpoch(queryCtx.TimeRange.From); ok {
			from = fmt.Sprintf("%sms", queryCtx.TimeRange.From)
		} else {
			from = "now() - " + queryCtx.TimeRange.From
		}
	}
	to := ""

	if queryCtx.TimeRange.To != "now" && queryCtx.TimeRange.To != "" {
		if _, ok := tsdb.TryParseUnixMsEpoch(queryCtx.TimeRange.To); ok {
			to = fmt.Sprintf(" and time < %sms", queryCtx.TimeRange.To)
		} else {
			to = " and time < now() - " + strings.Replace(queryCtx.TimeRange.To, "now-", "", 1)
		}
	}

	return fmt.Sprintf("time > %s%s", from, to)
}

func (query *Query) renderSelectors(queryCtx *tsdb.TsdbQuery) string {
	res := "SELECT "

	var selectors []string
	for _, sel := range query.Selects {
		stk := ""
		for _, s := range *sel {
			stk = s.Render(query, queryCtx, stk)
		}
		selectors = append(selectors, stk)
	}

	return res + strings.Join(selectors, ", ")
}

func (query *Query) renderMeasurement() string {
	var policy string
	if query.Policy == "" || query.Policy == "default" {
		policy = ""
	} else {
		policy = `"` + query.Policy + `".`
	}

	measurement := query.Measurement

	if !regexpMeasurementPattern.Match([]byte(measurement)) {
		measurement = fmt.Sprintf(`"%s"`, measurement)
	}

	return fmt.Sprintf(` FROM %s%s`, policy, measurement)
}

func (query *Query) renderWhereClause() string {
	res := " WHERE "
	conditions := query.renderTags()
	if len(conditions) > 0 {
		if len(conditions) > 1 {
			res += "(" + strings.Join(conditions, " ") + ")"
		} else {
			res += conditions[0]
		}
		res += " AND "
	}

	return res
}

func (query *Query) renderGroupBy(queryContext *tsdb.TsdbQuery) string {
	groupBy := ""
	for i, group := range query.GroupBy {
		if i == 0 {
			groupBy += " GROUP BY"
		}

		if i > 0 && utils.IsInStringArray(group.Type, []string{"field", "time", "tag"}) {
			groupBy += ", " //fill is so very special. fill is a creep, fill is a weirdo
		} else {
			groupBy += " "
		}

		groupBy += group.Render(query, queryContext, "")
	}

	return groupBy
}

func (query *Query) renderTz() string {
	tz := query.Tz
	if tz == "" {
		return ""
	}
	return fmt.Sprintf(" tz('%s')", tz)
}
