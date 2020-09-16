package suggestsysdrivers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/alerting/conditions"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

type InfluxdbBaseDriver struct {
	*baseDriver
}

func NewInfluxdbBaseDriver(driverType monitor.SuggestDriverType, resourceType monitor.MonitorResourceType,
	action monitor.SuggestDriverAction, suggest monitor.MonitorSuggest, rule monitor.SuggestSysRuleCreateInput) *InfluxdbBaseDriver {
	return &InfluxdbBaseDriver{
		baseDriver: newBaseDriver(
			driverType,
			resourceType,
			action,
			suggest,
			rule,
		)}
}

func (drv *InfluxdbBaseDriver) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	if input.ScaleRule == nil {
		return httperrors.NewInputParameterError("no found rule setting ")
	}
	if len(*input.ScaleRule) == 0 {
		return httperrors.NewInputParameterError("no found customize monitor rule")
	}
	for _, scale := range *input.ScaleRule {
		if scale.Database == "" {
			return httperrors.NewInputParameterError("database is missing")
		}
		if scale.Measurement == "" {
			return httperrors.NewInputParameterError("measurement is missing")
		}
		if scale.Field == "" {
			return httperrors.NewInputParameterError("field is missing")
		}
		if !utils.IsInStringArray(getQueryEvalType(scale), validators.EvaluatorDefaultTypes) {
			return httperrors.NewInputParameterError("the evalType is illegal")
		}
		if scale.Threshold == 0 {
			return httperrors.NewInputParameterError("threshold is meaningless")
		}
	}
	return nil
}

func getQueryEvalType(scale monitor.Scale) string {
	typ := ""
	switch scale.EvalType {
	case ">=", ">":
		typ = "gt"
	case "<=", "<":
		typ = "lt"
	}
	return typ
}

func (drv *InfluxdbBaseDriver) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, drv)
}

func (drv *InfluxdbBaseDriver) Run(rule *models.SSuggestSysRule, instance *monitor.SSuggestSysAlertSetting) {
	Run(drv, rule, instance)
}

func (drv *InfluxdbBaseDriver) GetLatestAlerts(rule *models.SSuggestSysRule, instance *monitor.SSuggestSysAlertSetting) ([]jsonutils.JSONObject, error) {
	//scaleEvalMatchs := make([]*monitor.EvalMatch, 0)
	ret := make([]jsonutils.JSONObject, 0)
	firing, evalMatchMap, err := drv.getScaleEvalResult(*instance.ScaleRule)
	if err != nil {
		return ret, errors.Wrap(err, "rule getScaleEvalResult happen error")
	}
	if firing {
		serverArr, err := drv.getResourcesByEvalMatchsMap(evalMatchMap, instance)
		if err != nil {
			return ret, errors.Wrap(err, "rule  getResource error")
		}
		return serverArr, nil
	}
	return ret, nil
}

func (drv *InfluxdbBaseDriver) getScaleEvalResult(scales []monitor.Scale) (bool, map[string][]*monitor.EvalMatch,
	error) {
	firing := false
	scaleEvalMatchs := make(map[string][]*monitor.EvalMatch, 0)
	for index, scale := range scales {
		condition := monitor.AlertCondition{
			Type:      "query",
			Query:     drv.newAlertQuery(scale),
			Evaluator: monitor.Condition{Type: getQueryEvalType(scale), Params: []float64{scale.Threshold}},
			Reducer:   monitor.Condition{Type: "avg"},
			Operator:  scale.Operator,
		}
		factory := alerting.GetConditionFactories()[condition.Type]
		queryCondition, err := factory(&condition, index)
		if err != nil {
			return firing, scaleEvalMatchs, errors.Wrapf(err, "construct query condition %s",
				jsonutils.Marshal(condition))
		}
		duration, _ := time.ParseDuration(condition.Query.From)
		queryCon := queryCondition.(*conditions.QueryCondition)
		queryCon.Reducer = conditions.NewSuggestRuleReducer(queryCon.Reducer.GetType(), duration)
		//evalContext := alerting.NewEvalContext(context.Background(), auth.AdminCredential(), nil)
		evalContext := alerting.EvalContext{
			Ctx:       context.Background(),
			UserCred:  auth.AdminCredential(),
			IsDebug:   true,
			IsTestRun: true,
		}
		conditionResult, err := queryCondition.Eval(&evalContext)
		if err != nil {
			return firing, scaleEvalMatchs, errors.Wrap(err, "condition eval error")
		}
		if index == 0 {
			firing = conditionResult.Firing
		}

		// calculating Firing based on operator
		if conditionResult.Operator == "or" {
			firing = firing || conditionResult.Firing
		} else {
			firing = firing && conditionResult.Firing
		}
		if firing {
			evalMatchs := conditionResult.EvalMatches
			if conditionResult.Operator == "and" {
				if index != 0 {
					evalMatchs = getAndEvalMatches(scaleEvalMatchs, evalMatchs)
					if len(evalMatchs) == 0 {
						return false, scaleEvalMatchs, nil
					}
				}
			} else {
				if index != 0 {
					getOrEvalMatches(scaleEvalMatchs, evalMatchs)
				}
			}
			key := fmt.Sprintf("%s--%d", scale.Field, index)
			scaleEvalMatchs[key] = evalMatchs
		}
	}
	return firing, scaleEvalMatchs, nil
}

func (drv *InfluxdbBaseDriver) getResourcesByEvalMatchsMap(evalMatchsMap map[string][]*monitor.EvalMatch,
	instance *monitor.SSuggestSysAlertSetting) ([]jsonutils.JSONObject, error) {
	matchLength := 0
	var maxEvalMatch []*monitor.EvalMatch
	for _, evalMatchs := range evalMatchsMap {
		if len(evalMatchs) > matchLength {
			matchLength = len(evalMatchs)
			maxEvalMatch = evalMatchs
		}
	}
	resArr := jsonutils.NewArray()
	for _, evalMatch := range maxEvalMatch {
		res, mappingId, mappingVal := drv.getResourceFromEvalMatch(evalMatch)
		if mappingId == "" {
			continue
		}
		suggestSysAlert, err := getSuggestSysAlertFromJson(res, drv)
		if err != nil {
			return nil, errors.Wrap(err, "Scale getSuggestSysAlertFromJson error")
		}
		suggestSysAlert.Action = string(monitor.SCALE_DOWN_DRIVER_ACTION)
		suggestSysAlert.MonitorConfig = jsonutils.Marshal(instance)
		suggestSysAlert.Problem = describeEvalResultTojson(evalMatchsMap, mappingId, mappingVal)
		resArr.Add(jsonutils.Marshal(suggestSysAlert))
	}
	return resArr.GetArray()
}

func (drv *InfluxdbBaseDriver) getResourceFromEvalMatch(evalMatch *monitor.EvalMatch) (jsonutils.JSONObject, string, string) {
	idTag := getMetricIdTag(evalMatch.Tags)
	var server jsonutils.JSONObject
	mappingId := ""
	mappingVal := ""
	for id, val := range idTag {
		serverobj, err := drv.getResourceById(id)
		if err != nil {
			continue
		}
		server = serverobj
		mappingId = id
		mappingVal = val
		break
	}
	return server, mappingId, mappingVal
}

func (drv *InfluxdbBaseDriver) getResourceById(id string) (jsonutils.JSONObject, error) {
	switch drv.GetResourceType() {
	case monitor.SCALE_MONTITOR_RES_TYPE:
		return getResource(id, &modules.Servers)
	case monitor.REDIS_UNREASONABLE_MONITOR_RES_TYPE:
		return getResource(id, &modules.ElasticCache)
	case monitor.RDS_UNREASONABLE_MONITOR_RES_TYPE:
		return getResource(id, &modules.DBInstance)
	case monitor.OSS_UNREASONABLE_MONITOR_RES_TYPE:
		return getResource(id, &modules.Buckets)
	}
	return nil, fmt.Errorf("unsupporttd to get resource by the driver type:%s", string(drv.GetResourceType()))
}

func describeEvalResultTojson(evalMatchsMap map[string][]*monitor.EvalMatch, mappingId, mappingVal string) jsonutils.JSONObject {
	problem := jsonutils.NewDict()
	for _, evalMatchs := range evalMatchsMap {
		for _, evalMatch := range evalMatchs {
			idTag := getMetricIdTag(evalMatch.Tags)
			if val, ok := idTag[mappingId]; ok {
				if val == mappingVal {
					problem.Add(jsonutils.NewFloat(*evalMatch.Value), evalMatch.Metric)
				}
			}
		}
	}
	return problem
}

func getResource(id string, manager modulebase.Manager) (jsonutils.JSONObject, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	server, err := manager.GetById(session, id, query)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func getMetricIdTag(tags map[string]string) map[string]string {
	idTags := make(map[string]string, 0)
	for key, val := range tags {
		if strings.HasSuffix(key, "_id") {
			idTags[key] = val
		}
	}
	return idTags
}

func getAndEvalMatches(scaleEvalMatchs map[string][]*monitor.EvalMatch, andscaleEvalMatchs []*monitor.EvalMatch) []*monitor.EvalMatch {
	for key, evalMatchs := range scaleEvalMatchs {
		andscaleEvalMatchs = getAndEvalMatches_(evalMatchs, andscaleEvalMatchs)
		if len(andscaleEvalMatchs) == 0 {
			return andscaleEvalMatchs
		}
		scaleEvalMatchs[key] = getAndEvalMatches_(andscaleEvalMatchs, evalMatchs)
	}
	return andscaleEvalMatchs
}

func getOrEvalMatches(scaleEvalMatchs map[string][]*monitor.EvalMatch, orscaleEvalMatchs []*monitor.EvalMatch) {
	for key, evalMatchs := range scaleEvalMatchs {
		matches_ := getOrEvalMatches_(evalMatchs, orscaleEvalMatchs)
		scaleEvalMatchs[key] = append(evalMatchs, matches_...)
	}
}

//by first  param to scale other param's length
func getAndEvalMatches_(scaleEvalMatchs, andscaleEvalMatchs []*monitor.EvalMatch) []*monitor.EvalMatch {
	resEvalMatchs := make([]*monitor.EvalMatch, 0)
	for _, evalMatch := range scaleEvalMatchs {
		idTags := getMetricIdTag(evalMatch.Tags)
	twoLoop:
		for _, andEvalMatch := range andscaleEvalMatchs {
			andIdTags := getMetricIdTag(andEvalMatch.Tags)
			//all the idTags must be equals
			for key, val := range idTags {
				if andVal, ok := andIdTags[key]; ok {
					if val != andVal {
						continue twoLoop
					}
				}
				continue twoLoop
			}
			resEvalMatchs = append(resEvalMatchs, andEvalMatch)
		}
	}
	return resEvalMatchs
}

func getOrEvalMatches_(scaleEvalMatchs, orscaleEvalMatchs []*monitor.EvalMatch) []*monitor.EvalMatch {
	resEvalMatchs := make([]*monitor.EvalMatch, 0)
	containsEvalMatchMap := make(map[int]string)
	for _, evalMatch := range scaleEvalMatchs {
		idTags := getMetricIdTag(evalMatch.Tags)
	twoLoop:
		for i, orEvalMatch := range orscaleEvalMatchs {
			orIdTags := getMetricIdTag(orEvalMatch.Tags)
			//all the idTags must be equals
			for key, val := range idTags {
				if orVal, ok := orIdTags[key]; ok {
					if val != orVal {
						continue twoLoop
					}
				}
				continue twoLoop
			}
			containsEvalMatchMap[i] = ""
		}
	}
	for i, andEvalMatch := range orscaleEvalMatchs {
		if _, ok := containsEvalMatchMap[i]; ok {
			continue
		}
		resEvalMatchs = append(resEvalMatchs, andEvalMatch)
	}
	return resEvalMatchs
}

func (drv *InfluxdbBaseDriver) newAlertQuery(scale monitor.Scale) monitor.AlertQuery {
	suggestSysRules, _ := models.SuggestSysRuleManager.GetRules(drv.GetType())
	datasource, _ := models.DataSourceManager.GetDefaultSource()
	return monitor.AlertQuery{
		Model:        newMetricQuery(scale),
		DataSourceId: datasource.Id,
		From:         suggestSysRules[0].TimeFrom,
		To:           "now",
	}
}

func newMetricQuery(scale monitor.Scale) monitor.MetricQuery {
	sels := make([]monitor.MetricQuerySelect, 0)
	sels = append(sels, monitor.NewMetricQuerySelect(monitor.MetricQueryPart{Type: "field", Params: []string{scale.Field}}))
	return monitor.MetricQuery{
		Database:    scale.Database,
		Measurement: scale.Measurement,
		Selects:     sels,
		GroupBy: []monitor.MetricQueryPart{
			{
				Type:   "field",
				Params: []string{"*"},
			},
		},
	}
}

func (drv *InfluxdbBaseDriver) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential,
	suggestSysAlert *models.SSuggestSysAlert,
	params *jsonutils.JSONDict) error {
	log.Println("InfluxdbBaseDriver StartResolveTask do nothing")
	return nil
}

func (drv *InfluxdbBaseDriver) Resolve(data *models.SSuggestSysAlert) error {
	log.Println("InfluxdbBaseDriver Resolve do nothing")
	return nil
}
