package models

import (
	"bytes"
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
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	mq "yunion.io/x/onecloud/pkg/monitor/metricquery"
	"yunion.io/x/onecloud/pkg/monitor/validators"
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
	filter, err := filterByCredential(userCred)
	if err != nil {
		return nil, err
	}
	return DataSourceManager.GetMeasurements(query, filter)
}

func filterByCredential(userCred mcclient.TokenCredential) (string, error) {
	roles := userCred.GetRoles()
	roleStr := strings.Join(roles, ",")
	if strings.Contains(roleStr, "admin") {
		return getTenantIdStr("admin", userCred)
	}
	if strings.Contains(roleStr, "domainadmin") {
		return getTenantIdStr("admin", userCred)
	}
	if strings.Contains(roleStr, "member") {
		return getTenantIdStr("admin", userCred)
	}
	return "", errors.Wrap(errors.ErrNotFound, "user role")
}

func getTenantIdStr(role string, userCred mcclient.TokenCredential) (string, error) {
	if role == "admin" {
		return "", nil
	}
	if role == "domainadmin" {
		domainId := userCred.GetDomainId()
		s := auth.GetAdminSession(context.Background(), "", "")
		params := jsonutils.Marshal(map[string]string{"domain_id": domainId})
		tenants, err := modules.Projects.List(s, params)
		if err != nil {
			return "", errors.Wrap(err, "Projects.List")
		}
		var buffer bytes.Buffer
		for index, tenant := range tenants.Data {
			tenantId, _ := tenant.GetString("id")
			if index != len(tenants.Data)-1 {
				buffer.WriteString(fmt.Sprintf("%s=%s %s", "tenant_id", tenantId, "OR"))
			} else {
				buffer.WriteString(fmt.Sprintf("%s=%s", "tenant_id", tenantId))
			}
		}
		return buffer.String(), nil
	}
	if role == "member" {
		tenantId := userCred.GetProjectId()
		return fmt.Sprintf("%s=%s", "tenant_id", tenantId), nil
	}
	return "", errors.ErrNotFound
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
	rtn, err := DataSourceManager.GetMetricMeasurement(query)
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
		setDefaultValue(q, inputQuery)
		err = self.ValidateInputQuery(*q)
		if err != nil {
			return jsonutils.NewDict(), err
		}
	}
	rtn, err := doQuery(*inputQuery)
	if err != nil {
		return jsonutils.NewDict(), err
	}
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

func (self *SUnifiedMonitorManager) ValidateInputQuery(query monitor.AlertQuery) error {
	if query.From == "" {
		query.From = "30m"
	}
	if query.Model.Interval == "" {
		query.Model.Interval = "5m"
	}
	if _, err := time.ParseDuration(query.Model.Interval); err != nil {
		return httperrors.NewInputParameterError("Invalid interval format: %s", query.Model.Interval)
	}
	return validators.ValidateAlertConditionQuery(query)
}

func setDefaultValue(query *monitor.AlertQuery, inputQuery *monitor.MetricInputQuery) {
	setDataSourceId(query)
	query.From = inputQuery.From
	query.To = "now"
	query.Model.Interval = inputQuery.Interval
	if len(query.Model.GroupBy) == 0 {
		query.Model.GroupBy = append(query.Model.GroupBy, monitor.MetricQueryPart{
			Type:   "field",
			Params: []string{"*"},
		})
	}
	//query.Model.GroupBy = append(query.Model.GroupBy, monitor.MetricQueryPart{
	//	Type:   "time",
	//	Params: []string{inputQuery.Interval},
	//}, monitor.MetricQueryPart{
	//	Type:   "fill",
	//	Params: []string{"none"},
	//})
}

func setDataSourceId(query *monitor.AlertQuery) {
	datasource, _ := DataSourceManager.GetDefaultSource()
	query.DataSourceId = datasource.Id
}
