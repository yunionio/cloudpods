package suggestsysdrivers

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

type ScaleDown struct {
	monitor.ScaleRule
}

func NewScaleDownDriver() models.ISuggestSysRuleDriver {
	return &ScaleDown{
		ScaleRule: []monitor.Scale{},
	}
}

func (_ *ScaleDown) GetType() string {
	return monitor.SCALE_DOWN

}

func (rule *ScaleDown) GetResourceType() string {
	return string(monitor.SCALE_MONTITOR_RES_TYPE)
}

func (rule *ScaleDown) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
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

func (rule *ScaleDown) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, rule)
}

func (rule *ScaleDown) Run(instance *monitor.SSuggestSysAlertSetting) {
	oldAlert, err := getLastAlerts(rule)
	if err != nil {
		log.Errorln(err)
		return
	}
	newAlert, err := rule.getLatestAlerts(instance)
	if err != nil {
		log.Errorln(err)
		return
	}
	DealAlertData(rule.GetType(), oldAlert, newAlert.Value())
}

func (rule *ScaleDown) getLatestAlerts(instance *monitor.SSuggestSysAlertSetting) (*jsonutils.JSONArray, error) {
	//scaleEvalMatchs := make([]*monitor.EvalMatch, 0)
	firing, evalMatchMap, err := rule.getScaleEvalResult(*instance.ScaleRule)
	if err != nil {
		return jsonutils.NewArray(), errors.Wrap(err, "rule getScaleEvalResult happen error")
	}
	if firing {

		serverArr, err := rule.getResourcesByEvalMatchsMap(evalMatchMap, instance)
		if err != nil {
			return jsonutils.NewArray(), errors.Wrap(err, "rule  getResource error")
		}
		return serverArr, nil
	}
	return jsonutils.NewArray(), nil
}

func (rule *ScaleDown) getScaleEvalResult(scales []monitor.Scale) (bool, map[string][]*monitor.EvalMatch, error) {
	firing := false
	scaleEvalMatchs := make(map[string][]*monitor.EvalMatch, 0)
	for index, scale := range scales {
		condition := monitor.AlertCondition{
			Type:      "query",
			Query:     rule.newAlertQuery(scale),
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
			}
			key := fmt.Sprintf("%s--%d", scale.Field, index)
			scaleEvalMatchs[key] = evalMatchs
		}
	}
	return firing, scaleEvalMatchs, nil
}

func (rule *ScaleDown) getResourcesByEvalMatchsMap(evalMatchsMap map[string][]*monitor.EvalMatch, instance *monitor.SSuggestSysAlertSetting) (*jsonutils.JSONArray, error) {
	matchLength := 0
	var maxEvalMatch []*monitor.EvalMatch
	for _, evalMatchs := range evalMatchsMap {
		if len(evalMatchs) > matchLength {
			matchLength = len(evalMatchs)
			maxEvalMatch = evalMatchs
		}
	}
	serverArr := jsonutils.NewArray()
	for _, evalMatch := range maxEvalMatch {
		server, mappingId, mappingVal := getServerFromEvalMatch(evalMatch)
		if mappingId == "" {
			continue
		}
		suggestSysAlert, err := getSuggestSysAlertFromJson(server, rule)
		if err != nil {
			return serverArr, errors.Wrap(err, "Scale getSuggestSysAlertFromJson error")
		}
		suggestSysAlert.Action = monitor.SCALE_DOWN_DRIVER_ACTION
		suggestSysAlert.MonitorConfig = jsonutils.Marshal(instance)
		suggestSysAlert.Problem = describeEvalResultTojson(evalMatchsMap, mappingId, mappingVal)
		serverArr.Add(jsonutils.Marshal(suggestSysAlert))
	}
	return serverArr, nil
}

func getServerFromEvalMatch(evalMatch *monitor.EvalMatch) (jsonutils.JSONObject, string, string) {
	idTag := getMetricIdTag(evalMatch.Tags)
	var server jsonutils.JSONObject
	mappingId := ""
	mappingVal := ""
	for id, val := range idTag {
		serverobj, err := getVm(val)
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

func getVm(id string) (jsonutils.JSONObject, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	server, err := modules.Servers.GetById(session, id, query)
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

//by first  param to scale other param's length
func getAndEvalMatches_(scaleEvalMatchs, andscaleEvalMatchs []*monitor.EvalMatch) []*monitor.EvalMatch {
	resEvalMatchs := make([]*monitor.EvalMatch, 0)
	for _, evalMatch := range scaleEvalMatchs {
		idTags := getMetricIdTag(evalMatch.Tags)
		for _, andEvalMatch := range andscaleEvalMatchs {
			andIdTags := getMetricIdTag(evalMatch.Tags)
			for key, val := range idTags {
				if andVal, ok := andIdTags[key]; ok {
					if val == andVal {
						resEvalMatchs = append(resEvalMatchs, andEvalMatch)
						break
					}
				}
			}
		}
	}
	return resEvalMatchs
}

func (rule *ScaleDown) newAlertQuery(scale monitor.Scale) monitor.AlertQuery {
	suggestSysRules, _ := models.SuggestSysRuleManager.GetRules(rule.GetType())
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

func (rule *ScaleDown) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential,
	suggestSysAlert *models.SSuggestSysAlert,
	params *jsonutils.JSONDict) error {
	log.Println("scaleDown StartResolveTask do nothing")
	return nil
}

func (rule *ScaleDown) Resolve(data *models.SSuggestSysAlert) error {
	log.Println("scaleDown Resolve do nothing")
	return nil
}
