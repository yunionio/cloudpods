package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	mq "yunion.io/x/onecloud/pkg/monitor/metricquery"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	UnifiedMonitorManager *SUnifiedMonitorManager
)

func init() {
	UnifiedMonitorManager = &SUnifiedMonitorManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SUnifiedMonitorManager{},
			"",
			"unifiedmonitor",
			"unifiedmonitors",
		),
	}
	UnifiedMonitorManager.SetVirtualObject(UnifiedMonitorManager)
}

type SUnifiedMonitorManager struct {
	db.SVirtualResourceBaseManager
}

type SUnifiedMonitorModel struct {
}

func (self *SUnifiedMonitorManager) AllowGetPropertyDatabases(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}
func (self *SUnifiedMonitorManager) GetPropertyDatabases(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return DataSourceManager.GetDatabases()
}

func (self *SUnifiedMonitorManager) AllowGetPropertyMeasurements(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SUnifiedMonitorManager) GetPropertyMeasurements(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	filter, err := getTagFilterByRequestQuery(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return DataSourceManager.GetMeasurementsWithDescriptionInfos(query, "", filter)
}

func getTagFilterByRequestQuery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (filter string, err error) {

	scope, _ := query.GetString("scope")
	filter, err = filterByScope(ctx, userCred, scope, query)
	return
}

func filterByScope(ctx context.Context, userCred mcclient.TokenCredential, scope string, data jsonutils.JSONObject) (string, error) {
	domainId := jsonutils.GetAnyString(data, []string{"domain_id", "domain", "project_domain_id", "project_domain"})
	projectId := jsonutils.GetAnyString(data, []string{"project_id", "project"})
	if projectId != "" {
		project, err := db.DefaultProjectFetcher(ctx, projectId)
		if err != nil {
			return "", err
		}
		projectId = project.GetProjectId()
		domainId = project.GetProjectDomainId()
	}
	if domainId != "" {
		domain, err := db.DefaultDomainFetcher(ctx, domainId)
		if err != nil {
			return "", err
		}
		domainId = domain.GetProjectDomainId()
		domain.GetProjectId()
	}
	switch scope {
	case "system":
		return "", nil
	case "domain":
		if domainId == "" {
			domainId = userCred.GetProjectDomainId()
		}
		return getProjectIdsFilterByDomain(domainId)
	default:
		if projectId == "" {
			projectId = userCred.GetProjectId()
		}
		return getProjectIdFilterByProject(projectId)

	}
}

func getTenantIdStr(role string, userCred mcclient.TokenCredential) (string, error) {
	if role == "admin" {
		return "", nil
	}
	if role == "domainadmin" {
		domainId := userCred.GetDomainId()
		return getProjectIdsFilterByDomain(domainId)
	}
	if role == "member" {
		tenantId := userCred.GetProjectId()
		return getProjectIdFilterByProject(tenantId)
	}
	return "", errors.ErrNotFound
}

func getProjectIdsFilterByDomain(domainId string) (string, error) {
	//s := auth.GetAdminSession(context.Background(), "", "")
	//params := jsonutils.Marshal(map[string]string{"domain_id": domainId})
	//tenants, err := modules.Projects.List(s, params)
	//if err != nil {
	//	return "", errors.Wrap(err, "Projects.List")
	//}
	//var buffer bytes.Buffer
	//buffer.WriteString("( ")
	//for index, tenant := range tenants.Data {
	//	tenantId, _ := tenant.GetString("id")
	//	if index != len(tenants.Data)-1 {
	//		buffer.WriteString(fmt.Sprintf(" %s =~ /%s/ %s ", "tenant_id", tenantId, "OR"))
	//	} else {
	//		buffer.WriteString(fmt.Sprintf(" %s =~ /%s/ ", "tenant_id", tenantId))
	//	}
	//}
	//buffer.WriteString(" )")
	//return buffer.String(), nil
	return fmt.Sprintf(`"%s" =~ /%s/`, "domain_id", domainId), nil

}

func getProjectIdFilterByProject(projectId string) (string, error) {
	return fmt.Sprintf(`"%s" =~ /%s/`, "tenant_id", projectId), nil
}

func (self *SUnifiedMonitorManager) AllowGetPropertyMetricMeasurement(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SUnifiedMonitorManager) GetPropertyMetricMeasurement(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	metricFunc := monitor.MetricFunc{
		FieldOptType:  monitor.UNIFIED_MONITOR_FIELD_OPT_TYPE,
		FieldOptValue: monitor.UNIFIED_MONITOR_FIELD_OPT_VALUE,
		GroupOptType:  monitor.UNIFIED_MONITOR_GROUPBY_OPT_TYPE,
		GroupOptValue: monitor.UNIFIED_MONITOR_GROUPBY_OPT_VALUE,
	}
	filter, err := getTagFilterByRequestQuery(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	rtn, err := DataSourceManager.GetMetricMeasurement(query, filter)
	if err != nil {
		return nil, err
	}
	rtn.(*jsonutils.JSONDict).Add(jsonutils.Marshal(&metricFunc), "func")
	return rtn, nil
}

func (self *SUnifiedMonitorManager) AllowPerformQuery(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SUnifiedMonitorManager) PerformQuery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	inputQuery := new(monitor.MetricInputQuery)
	err := data.Unmarshal(inputQuery)
	if err != nil {
		return jsonutils.NewDict(), err
	}
	if len(inputQuery.MetricQuery) == 0 {
		return nil, httperrors.NewInputParameterError("no metric_query field in param")
	}
	for _, q := range inputQuery.MetricQuery {
		scope, _ := data.GetString("scope")
		ownId, _ := self.FetchOwnerId(ctx, data)
		if ownId == nil {
			ownId = userCred
		}
		setDefaultValue(q, inputQuery, scope, ownId)
		err = self.ValidateInputQuery(q)
		if err != nil {
			return jsonutils.NewDict(), err
		}
	}

	var groupByTag = make([]string, 0)
	for _, query := range inputQuery.MetricQuery {
		for _, group := range query.Model.GroupBy {
			if group.Type == "tag" {
				groupByTag = append(groupByTag, group.Params[0])
			}
		}
	}

	rtn, err := doQuery(*inputQuery)
	if err != nil {
		return jsonutils.NewDict(), err
	}

	setSerieRowName(&rtn.Series, groupByTag)
	fillSerieTags(&rtn.Series)
	return jsonutils.Marshal(rtn), nil
}

func doQuery(query monitor.MetricInputQuery) (*mq.Metrics, error) {
	conditions := make([]*monitor.AlertCondition, 0)
	for _, q := range query.MetricQuery {
		condition := monitor.AlertCondition{
			Type:  "metricquery",
			Query: *q,
		}
		conditions = append(conditions, &condition)
	}
	factory := mq.GetQueryFactories()["metricquery"]
	metricQ, err := factory(conditions)
	if err != nil {
		return nil, err
	}
	return metricQ.ExecuteQuery()
}

func (self *SUnifiedMonitorManager) ValidateInputQuery(query *monitor.AlertQuery) error {
	if query.From == "" {
		query.From = "1h"
	}
	if query.Model.Interval == "" {
		query.Model.Interval = "5m"
	}
	if query.To == "" {
		query.To = "now"
	}
	if _, err := time.ParseDuration(query.Model.Interval); err != nil {
		return httperrors.NewInputParameterError("Invalid interval format: %s", query.Model.Interval)
	}
	return validators.ValidateSelectOfMetricQuery(*query)
}

func setDefaultValue(query *monitor.AlertQuery, inputQuery *monitor.MetricInputQuery,
	scope string, ownerId mcclient.IIdentityProvider) {
	setDataSourceId(query)
	query.From = inputQuery.From
	query.To = inputQuery.To
	query.Model.Interval = inputQuery.Interval

	if len(query.Model.GroupBy) == 0 {
		query.Model.GroupBy = append(query.Model.GroupBy,
			monitor.MetricQueryPart{
				Type:   "field",
				Params: []string{"*"},
			})
	}

	query.Model.GroupBy = append(query.Model.GroupBy,
		monitor.MetricQueryPart{
			Type:   "time",
			Params: []string{"$interval"},
		},
		monitor.MetricQueryPart{
			Type:   "fill",
			Params: []string{"none"},
		})

	for i, sel := range query.Model.Selects {
		if len(sel) > 1 {
			continue
		}
		sel = append(sel, monitor.MetricQueryPart{
			Type:   "mean",
			Params: []string{},
		})
		query.Model.Selects[i] = sel
	}
	var projectId, domainId string
	switch rbacutils.TRbacScope(scope) {
	case rbacutils.ScopeProject:
		projectId = ownerId.GetProjectId()
		containId := false
		for _, tagFilter := range query.Model.Tags {
			if tagFilter.Key == "tenant_id" {
				containId = true
				break
			}
		}
		if !containId {
			query.Model.Tags = append(query.Model.Tags, monitor.MetricQueryTag{
				Key:       "tenant_id",
				Operator:  "=",
				Value:     projectId,
				Condition: "and",
			})
		}
	case rbacutils.ScopeDomain:
		domainId = ownerId.GetProjectDomainId()
		containId := false
		for _, tagFilter := range query.Model.Tags {
			if tagFilter.Key == "domain_id" {
				containId = true
				break
			}
		}
		if !containId {
			query.Model.Tags = append(query.Model.Tags, monitor.MetricQueryTag{
				Key:       "domain_id",
				Operator:  "=",
				Value:     domainId,
				Condition: "and",
			})
		}
	}
}

func setDataSourceId(query *monitor.AlertQuery) {
	datasource, _ := DataSourceManager.GetDefaultSource()
	query.DataSourceId = datasource.Id
}

func setSerieRowName(series *tsdb.TimeSeriesSlice, groupTag []string) {
	//Add rowname，The front end displays the curve according to rowname
	var index, unknownIndex = 1, 1
	for i, serie := range *series {
		//setRowName by groupTag
		if len(groupTag) != 0 {
			for key, val := range serie.Tags {

				if strings.Contains(strings.Join(groupTag, ","), key) {
					serie.RawName = fmt.Sprintf("%s", val)
					(*series)[i] = serie
					break
				}
			}
			continue
		}
		measurement := strings.Split(serie.Name, ".")[0]
		//sep measurement set RowName by spe param
		measurements, _ := MetricMeasurementManager.getMeasurementByName(measurement)
		if len(measurements) != 0 {
			if key, ok := monitor.MEASUREMENT_TAG_KEYWORD[measurements[0].ResType]; ok {
				serie.RawName = fmt.Sprintf("%d: %s", index, serie.Tags[key])
				(*series)[i] = serie
				index++
				continue
			}
		}
		//other condition set RowName
		for key, val := range serie.Tags {
			if strings.Contains(key, "id") {
				serie.RawName = fmt.Sprintf("%d: %s", index, val)
				(*series)[i] = serie
				index++
				break
			}
		}
		if serie.RawName == "" {
			serie.RawName = fmt.Sprintf("unknown-%d", unknownIndex)
			(*series)[i] = serie
			unknownIndex++
		}
	}

}

func fillSerieTags(series *tsdb.TimeSeriesSlice) {
	for i, serie := range *series {
		for _, tag := range []string{"brand", "platform", "hypervisor"} {
			if val, ok := serie.Tags[tag]; ok {
				serie.Tags["brand"] = val
				break
			}
		}
		(*series)[i] = serie
	}
}
