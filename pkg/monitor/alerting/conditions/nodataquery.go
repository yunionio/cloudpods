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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

const (
	NO_DATA = "nodata"

	HOST_TAG_NAME   = "name"
	HOST_TAG_IP     = "access_ip"
	RESOURCE_TAG_IP = "ips"
	HOST_TAG_BRAND  = "brand"
)

func init() {
	alerting.RegisterCondition("nodata_query", func(model *monitor.AlertCondition, index int) (alerting.Condition,
		error) {
		return newNoDataQueryCondition(model, index)
	})
}

type NoDataQueryCondition struct {
	*QueryCondition
}

func (c *NoDataQueryCondition) Eval(context *alerting.EvalContext) (*alerting.ConditionResult, error) {
	timeRange := tsdb.NewTimeRange(c.Query.From, c.Query.To)

	ret, err := c.executeQuery(context, timeRange)
	if err != nil {
		return nil, err
	}
	seriesList := ret.series
	var matches []*monitor.EvalMatch
	var alertOkmatches []*monitor.EvalMatch
	normalHostIds := make(map[string]*monitor.EvalMatch, 0)
serLoop:
	for _, series := range seriesList {
		tagId := monitor.MEASUREMENT_TAG_ID[context.Rule.RuleDescription[0].ResType]
		if len(tagId) == 0 {
			tagId = "host_id"
		}
		for key, val := range series.Tags {
			if key == tagId {
				if len(context.Rule.RuleDescription) == 0 {
					return &alerting.ConditionResult{
						Firing:             false,
						NoDataFound:        true,
						Operator:           c.Operator,
						EvalMatches:        matches,
						AlertOkEvalMatches: alertOkmatches,
					}, nil
				}

				reducedValue, valStrArr := c.Reducer.Reduce(series)
				match, err := c.NewEvalMatch(context, *series, nil, reducedValue, valStrArr)
				if err != nil {
					return nil, errors.Wrap(err, "NoDataQueryCondition NewEvalMatch error")
				}
				normalHostIds[val] = match
				continue serLoop
			}
		}
	}
	allResources, err := c.GetQueryResources()
	if err != nil {
		return nil, errors.Wrap(err, "GetQueryResources err")
	}
	for _, host := range allResources {
		id, _ := host.GetString("id")
		evalMatch, err := c.NewNoDataEvalMatch(context, host)
		if err != nil {
			return nil, errors.Wrap(err, "NewNoDataEvalMatch error")
		}
		if normalMatch, ok := normalHostIds[id]; !ok {
			c.createEvalMatchTagFromHostJson(context, evalMatch, host)
			matches = append(matches, evalMatch)
		} else {
			c.createEvalMatchTagFromHostJson(context, normalMatch, host)
			alertOkmatches = append(alertOkmatches, normalMatch)
		}
	}
	return &alerting.ConditionResult{
		Firing:             len(matches) > 0,
		NoDataFound:        0 == len(seriesList),
		Operator:           c.Operator,
		EvalMatches:        matches,
		AlertOkEvalMatches: alertOkmatches,
	}, nil
}

func (c *NoDataQueryCondition) convertTagsQuery(evalContext *alerting.EvalContext,
	query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	alertDetails, err := c.GetCommonAlertDetails(evalContext)
	if err != nil {
		return nil, err
	}
	for i, _ := range c.Query.Model.Tags {
		filterCount := 0
		if c.Query.Model.Tags[i].Operator == "=" {
			if tag, ok := monitor.MEASUREMENT_TAG_KEYWORD[alertDetails.ResType]; ok {
				if c.Query.Model.Tags[i].Key == tag {
					query.Set("name", jsonutils.NewString(c.Query.Model.Tags[i].Value))
					continue
				}
			}
		}
		query.Set(fmt.Sprintf("filter.%d", filterCount),
			jsonutils.NewString(fmt.Sprintf("%s.notin(%s)", c.Query.Model.Tags[i].Key, c.Query.Model.Tags[i].Value)))
		filterCount++
	}
	return query, nil
}

func (c *NoDataQueryCondition) NewNoDataEvalMatch(context *alerting.EvalContext, host jsonutils.JSONObject) (*monitor.EvalMatch, error) {
	evalMatch := new(monitor.EvalMatch)
	alert, err := models.CommonAlertManager.GetAlert(context.Rule.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetAlert to NewEvalMatch error")
	}
	settings, _ := alert.GetSettings()
	alertDetails := alert.GetCommonAlertMetricDetailsFromAlertCondition(c.Index, &settings.Conditions[c.Index])
	evalMatch.Metric = fmt.Sprintf("%s.%s", alertDetails.Measurement, alertDetails.Field)
	queryKeyInfo := ""
	if len(alertDetails.MeasurementDisplayName) > 0 && len(alertDetails.FieldDescription.DisplayName) > 0 {
		queryKeyInfo = fmt.Sprintf("%s.%s", alertDetails.MeasurementDisplayName, alertDetails.FieldDescription.DisplayName)
	}
	if len(queryKeyInfo) == 0 {
		queryKeyInfo = evalMatch.Metric
	}
	evalMatch.Unit = alertDetails.FieldDescription.Unit
	msg := fmt.Sprintf("%s.%s %s ", alertDetails.Measurement, alertDetails.Field,
		alertDetails.Comparator)
	context.Rule.Message = msg

	//evalMatch.Condition = c.GenerateFormatCond(meta, queryKeyInfo).String()
	evalMatch.ValueStr = NO_DATA
	return evalMatch, nil
}

func (c *NoDataQueryCondition) createEvalMatchTagFromHostJson(evalContext *alerting.EvalContext, evalMatch *monitor.EvalMatch,
	host jsonutils.JSONObject) {
	evalMatch.Tags = make(map[string]string, 0)

	ip, _ := host.GetString(HOST_TAG_IP)
	if ip == "" {
		ip, _ = host.GetString(RESOURCE_TAG_IP)
	}
	name, _ := host.GetString(HOST_TAG_NAME)
	brand, _ := host.GetString(HOST_TAG_BRAND)
	evalMatch.Tags["ip"] = ip
	evalMatch.Tags[HOST_TAG_NAME] = name
	evalMatch.Tags[HOST_TAG_BRAND] = brand

	switch evalContext.Rule.RuleDescription[0].ResType {
	case monitor.METRIC_RES_TYPE_GUEST:
	case monitor.METRIC_RES_TYPE_RDS:
	case monitor.METRIC_RES_TYPE_REDIS:
	case monitor.METRIC_RES_TYPE_OSS:
	default:
		evalMatch.Tags["host"] = name
		evalMatch.Tags[hostconsts.TELEGRAF_TAG_KEY_RES_TYPE] = hostconsts.TELEGRAF_TAG_ONECLOUD_RES_TYPE
		evalMatch.Tags[hostconsts.TELEGRAF_TAG_KEY_HOST_TYPE] = hostconsts.TELEGRAF_TAG_ONECLOUD_HOST_TYPE_HOST
	}
}

func newNoDataQueryCondition(model *monitor.AlertCondition, index int) (*NoDataQueryCondition, error) {
	queryCondition, err := newQueryCondition(model, index)
	if err != nil {
		return nil, err
	}
	condition := new(NoDataQueryCondition)
	condition.QueryCondition = queryCondition
	return condition, nil
}
