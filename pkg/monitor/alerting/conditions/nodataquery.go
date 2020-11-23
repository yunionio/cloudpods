package conditions

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mc_mds "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

const (
	NO_DATA = "nodata"

	HOST_TAG_NAME  = "name"
	HOST_TAG_IP    = "access_ip"
	HOST_TAG_BRAND = "brand"
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
		for key, val := range series.Tags {
			if strings.Contains(key, "host_id") {
				reducedValue, valStrArr := c.Reducer.Reduce(series)
				match, err := c.NewEvalMatch(context, *series, nil, reducedValue, valStrArr)
				if err != nil {
					return nil, errors.Wrap(err, "NoDataQueryCondition NewEvalMatch error")
				}
				log.Errorf("nodata match:%#v", match)
				normalHostIds[val] = match
				continue serLoop
			}
		}
	}
	allHosts, err := c.getOnecloudHosts()
	if err != nil {
		return nil, errors.Wrap(err, "NoDataQueryCondition getOnecloudHosts error")
	}
	for _, host := range allHosts {
		id, _ := host.GetString("id")
		evalMatch, err := c.NewNoDataEvalMatch(context, host)
		if err != nil {
			return nil, errors.Wrap(err, "NewNoDataEvalMatch error")
		}
		if normalMatch, ok := normalHostIds[id]; !ok {
			c.createEvalMatchTagFromHostJson(evalMatch, host)
			matches = append(matches, evalMatch)
		} else {
			c.createEvalMatchTagFromHostJson(normalMatch, host)
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

func (c *NoDataQueryCondition) getOnecloudHosts() ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Set("brand", jsonutils.NewString(hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND))
	allHosts, err := ListAllResources(&mc_mds.Hosts, query)
	if err != nil {
		return nil, errors.Wrap(err, "NoDataQueryCondition Host list error")
	}
	return allHosts, nil
}

func ListAllResources(manager modulebase.Manager, params *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	params.Add(jsonutils.NewString("system"), "scope")
	params.Add(jsonutils.NewInt(0), "limit")
	var count int
	session := auth.GetAdminSession(context.Background(), "", "")
	objs := make([]jsonutils.JSONObject, 0)
	for {
		params.Set("offset", jsonutils.NewInt(int64(count)))
		result, err := manager.List(session, params)
		if err != nil {
			return nil, errors.Wrapf(err, "list %s resources with params %s", manager.KeyString(), params.String())
		}
		for _, data := range result.Data {
			objs = append(objs, data)
		}
		total := result.Total
		count = count + len(result.Data)
		if count >= total {
			break
		}
	}
	return objs, nil
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
	msg := fmt.Sprintf("%s.%s %s %s", alertDetails.Measurement, alertDetails.Field,
		alertDetails.Comparator, c.RationalizeValueFromUnit(alertDetails.Threshold, evalMatch.Unit, ""))
	if len(context.Rule.Message) == 0 {
		context.Rule.Message = msg
	}
	//evalMatch.Condition = c.GenerateFormatCond(meta, queryKeyInfo).String()
	evalMatch.ValueStr = NO_DATA
	evalMatch.MeasurementDesc = alertDetails.MeasurementDisplayName
	evalMatch.FieldDesc = alertDetails.FieldDescription.DisplayName
	return evalMatch, nil
}

func (c *NoDataQueryCondition) createEvalMatchTagFromHostJson(evalMatch *monitor.EvalMatch, host jsonutils.JSONObject) {
	evalMatch.Tags = make(map[string]string, 0)

	ip, _ := host.GetString(HOST_TAG_IP)
	name, _ := host.GetString(HOST_TAG_NAME)
	brand, _ := host.GetString(HOST_TAG_BRAND)
	evalMatch.Tags["ip"] = ip
	evalMatch.Tags[HOST_TAG_NAME] = name
	evalMatch.Tags["host"] = name
	evalMatch.Tags[HOST_TAG_BRAND] = brand

	evalMatch.Tags[hostconsts.TELEGRAF_TAG_KEY_RES_TYPE] = hostconsts.TELEGRAF_TAG_ONECLOUD_RES_TYPE
	evalMatch.Tags[hostconsts.TELEGRAF_TAG_KEY_HOST_TYPE] = hostconsts.TELEGRAF_TAG_ONECLOUD_HOST_TYPE_HOST
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
