package conditions

import (
	"context"
	"fmt"
	"sort"

	"yunion.io/x/jsonutils"
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
	allHosts, err := c.getOnecloudResources(context)
	allHosts = c.filterAllResources(context, allHosts)
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

func (c *NoDataQueryCondition) filterAllResources(context *alerting.EvalContext,
	resources []jsonutils.JSONObject) []jsonutils.JSONObject {
	if len(c.Query.Model.Tags) == 0 {
		return resources
	}
	filterIdMap := make(map[string]jsonutils.JSONObject)
	filterQuery := c.getFilterQuery()
	intKey := make([]int, 0)
	if len(filterQuery) != 0 {
		for key, _ := range filterQuery {
			intKey = append(intKey, key)
		}
		sort.Ints(intKey)
		minKey := intKey[0]
		if minKey != 0 {
			filterQuery[0] = minKey - 1
		}
	} else {
		filterQuery[0] = len(c.Query.Model.Tags) - 1
	}
	for start, end := range filterQuery {
		filterResources := c.getFilterResources(context, start, end, resources)
		filterIdMap = c.fillFilterRes(filterResources, filterIdMap)
	}
	filterRes := make([]jsonutils.JSONObject, 0)
	for _, obj := range filterIdMap {
		filterRes = append(filterRes, obj)
	}
	return filterRes
}

func (c *NoDataQueryCondition) fillFilterRes(filterRes []jsonutils.JSONObject,
	filterIdMap map[string]jsonutils.JSONObject) map[string]jsonutils.JSONObject {
	for _, res := range filterRes {
		id, _ := res.GetString("id")
		if _, ok := filterIdMap[id]; !ok {
			filterIdMap[id] = res
		}
	}
	return filterIdMap
}

func (c *NoDataQueryCondition) getFilterQuery() map[int]int {
	length := len(c.Query.Model.Tags)
	tagIndexMap := make(map[int]int)
	for i := 0; i < length; i++ {
		if c.Query.Model.Tags[i].Condition == "OR" {
			andIndex := c.getTheAndOfConditionor(i + 1)
			if andIndex == i+1 {
				tagIndexMap[i] = i
				continue
			}
			if andIndex == length {
				for j := i; j < length; j++ {
					tagIndexMap[j] = j
				}
				break
			}
			tagIndexMap[i] = andIndex
			i = andIndex
		}
	}
	return tagIndexMap
}

func (c *NoDataQueryCondition) getTheAndOfConditionor(start int) int {
	for i := start; i < len(c.Query.Model.Tags); i++ {
		if c.Query.Model.Tags[i].Condition != "AND" {
			return i
		}
	}
	return len(c.Query.Model.Tags)
}

func (c *NoDataQueryCondition) getFilterResources(evalContext *alerting.EvalContext, start int, end int,
	resources []jsonutils.JSONObject) []jsonutils.JSONObject {
	relationMap := c.getTagKeyRelationMap(evalContext)
	tmp := resources
	for i := start; i <= end; i++ {
		tag := c.Query.Model.Tags[i]
		relationKey := relationMap[tag.Key]
		filterObj := make([]jsonutils.JSONObject, 0)
		for _, res := range tmp {
			val, _ := res.GetString(relationKey)
			if c.Query.Model.Tags[i].Operator == "=" {
				if val == c.Query.Model.Tags[i].Value {
					filterObj = append(filterObj, res)
				}
			}
			if c.Query.Model.Tags[i].Operator == "!=" {
				if val != c.Query.Model.Tags[i].Value {
					filterObj = append(filterObj, res)
				}
			}
		}
		tmp = filterObj
		if len(tmp) == 0 {
			return tmp
		}
	}
	return tmp
}

func (c *NoDataQueryCondition) getTagKeyRelationMap(evalContext *alerting.EvalContext) map[string]string {
	relationMap := make(map[string]string)
	switch evalContext.Rule.RuleDescription[0].ResType {
	case monitor.METRIC_RES_TYPE_HOST:
		relationMap = HostTags
	case monitor.METRIC_RES_TYPE_GUEST:
		relationMap = ServerTags
	case monitor.METRIC_RES_TYPE_RDS:
		relationMap = RdsTags
	case monitor.METRIC_RES_TYPE_REDIS:
		relationMap = RedisTags
	case monitor.METRIC_RES_TYPE_OSS:
		relationMap = OssTags
	default:
		relationMap = HostTags
	}
	return relationMap
}

func (c *NoDataQueryCondition) getOnecloudResources(evalContext *alerting.EvalContext) ([]jsonutils.JSONObject, error) {
	var err error
	allResources := make([]jsonutils.JSONObject, 0)
	if len(evalContext.Rule.RuleDescription) == 0 {
		return []jsonutils.JSONObject{}, nil
	}
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewStringArray([]string{"running", "ready"}), "status")
	query.Add(jsonutils.NewString("true"), "admin")
	//if len(c.Query.Model.Tags) != 0 {
	//	query, err = c.convertTagsQuery(evalContext, query)
	//	if err != nil {
	//		return nil, errors.Wrap(err, "NoDataQueryCondition convertTagsQuery error")
	//	}
	//}
	switch evalContext.Rule.RuleDescription[0].ResType {
	case monitor.METRIC_RES_TYPE_HOST:
		query.Set("host-type", jsonutils.NewString(hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR))
		allResources, err = ListAllResources(&mc_mds.Hosts, query)
	case monitor.METRIC_RES_TYPE_GUEST:
		allResources, err = ListAllResources(&mc_mds.Servers, query)
	case monitor.METRIC_RES_TYPE_RDS:
		allResources, err = ListAllResources(&mc_mds.DBInstance, query)
	case monitor.METRIC_RES_TYPE_REDIS:
		allResources, err = ListAllResources(&mc_mds.ElasticCache, query)
	case monitor.METRIC_RES_TYPE_OSS:
		allResources, err = ListAllResources(&mc_mds.Buckets, query)
	default:
		query := jsonutils.NewDict()
		query.Set("brand", jsonutils.NewString(hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND))
		query.Set("host-type", jsonutils.NewString(hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR))
		allResources, err = ListAllResources(&mc_mds.Hosts, query)
	}

	if err != nil {
		return nil, errors.Wrap(err, "NoDataQueryCondition Host list error")
	}
	return allResources, nil
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

func ListAllResources(manager modulebase.Manager, params *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	params.Add(jsonutils.NewString("system"), "scope")
	params.Add(jsonutils.NewInt(0), "limit")
	params.Add(jsonutils.NewBool(true), "details")
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

var (
	ServerTags = map[string]string{
		"host":             "host",
		"host_id":          "host_id",
		"vm_id":            "id",
		"vm_ip":            "ips",
		"vm_name":          "name",
		"zone":             "zone",
		"zone_id":          "zone_id",
		"zone_ext_id":      "zone_ext_id",
		"os_type":          "os_type",
		"status":           "status",
		"cloudregion":      "cloudregion",
		"cloudregion_id":   "cloudregion_id",
		"region_ext_id":    "region_ext_id",
		"tenant":           "tenant",
		"tenant_id":        "tenant_id",
		"brand":            "brand",
		"scaling_group_id": "vm_scaling_group_id",
		"domain_id":        "domain_id",
		"project_domain":   "project_domain",
	}

	HostTags = map[string]string{
		"host_id":        "id",
		"host_ip":        "ips",
		"host":           "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"region_ext_id":  "region_ext_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	RdsTags = map[string]string{
		"host":           "host",
		"host_id":        "host_id",
		"rds_id":         "id",
		"rds_ip":         "ips",
		"rds_name":       "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"region_ext_id":  "region_ext_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	RedisTags = map[string]string{
		"host":           "host",
		"host_id":        "host_id",
		"redis_id":       "id",
		"redis_ip":       "ips",
		"redis_name":     "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"region_ext_id":  "region_ext_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	OssTags = map[string]string{
		"host":           "host",
		"host_id":        "host_id",
		"oss_id":         "id",
		"oss_ip":         "ips",
		"oss_name":       "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"region_ext_id":  "region_ext_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	ElbTags = map[string]string{
		"host":           "host",
		"host_id":        "host_id",
		"elb_id":         "id",
		"elb_ip":         "ips",
		"elb_name":       "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"region":         "region",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	CloudAccountTags = map[string]string{
		"cloudaccount_id":   "id",
		"cloudaccount_name": "name",
		"brand":             "brand",
		"domain_id":         "domain_id",
		"project_domain":    "project_domain",
	}
)
