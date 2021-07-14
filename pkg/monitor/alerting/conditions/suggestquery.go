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

package conditions

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

func init() {
	alerting.RegisterCondition("suggest_query", func(model *monitor.AlertCondition, index int) (alerting.Condition,
		error) {
		return newSuggestQueryCondition(model, index)
	})
}

type SuggestQueryCondition struct {
	*QueryCondition
}

func newSuggestQueryCondition(model *monitor.AlertCondition, index int) (*SuggestQueryCondition, error) {
	queryCondition, err := newQueryCondition(model, index)
	if err != nil {
		return nil, err
	}
	condition := new(SuggestQueryCondition)
	condition.QueryCondition = queryCondition
	return condition, nil
}

func (c *SuggestQueryCondition) Eval(context *alerting.EvalContext) (*alerting.ConditionResult, error) {
	timeRange := tsdb.NewTimeRange(c.Query.From, c.Query.To)
	ret, err := c.executeQuery(context, timeRange)
	if err != nil {
		return nil, err
	}
	seriesList := ret.series
	emptySeriesCount := 0
	evalMatchCount := 0
	var matches []*monitor.EvalMatch
	for _, series := range seriesList {
		reducedValue, _ := c.Reducer.Reduce(series)
		evalMatch := c.Evaluator.Eval(reducedValue)
		if reducedValue == nil {
			emptySeriesCount++
		}
		if context.IsTestRun {
			context.Logs = append(context.Logs, &monitor.ResultLogEntry{
				Message: fmt.Sprintf("Condition[%d]: Eval: %v, Metric: %s, Value: %v", c.Index, evalMatch, series.Name, reducedValue),
			})
		}
		if evalMatch {
			evalMatchCount++
		}
		if evalMatch {
			matches = append(matches, &monitor.EvalMatch{
				Metric: series.Name,
				Value:  reducedValue,
				Tags:   series.Tags,
			})
		}
	}

	// handle no series special case
	if len(seriesList) == 0 {
		// eval condition for null value
		evalMatch := c.Evaluator.Eval(nil)
		if context.IsTestRun {
			context.Logs = append(context.Logs, &monitor.ResultLogEntry{
				Message: fmt.Sprintf("Condition: Eval: %v, Query returned No Series (reduced to null/no value)", evalMatch),
			})
		}
		if evalMatch {
			evalMatchCount++
			matches = append(matches, &monitor.EvalMatch{
				Metric: "NoData",
				Value:  nil,
			})
		}
	}
	return &alerting.ConditionResult{
		Firing:      evalMatchCount > 0,
		NoDataFound: emptySeriesCount == len(seriesList),
		Operator:    c.Operator,
		EvalMatches: matches,
	}, nil
}
