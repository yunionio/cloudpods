package monitor

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertDashBoardCreateOptions struct {
	NAME     string `help:"Name of bashboard"`
	Metric   string `help:"Metric name, include measurement and field, e.g. vm_cpu.usage_active" required:"true"`
	Database string
	Interval string `help:"query aggregation interval e.g. 1m|5s"`
	From     string `help:"query start time e.g. 5m|6h"`
	To       string `help:"query end time"`
	Refresh  string `help:"dashboard query refresh priod e.g. 1m|5m"`
	Scope    string
}

func (o *AlertDashBoardCreateOptions) Params() (jsonutils.JSONObject, error) {
	createInput := new(monitor.AlertDashBoardCreateInput)
	createInput.Name = o.NAME
	createInput.Scope = o.Scope
	createInput.Refresh = o.Refresh
	createInput.From = o.From
	createInput.To = o.To
	alertQuery := new(monitor.CommonAlertQuery)
	metrics := strings.Split(o.Metric, ".")
	if len(metrics) != 2 {
		return nil, errors.Wrap(httperrors.ErrBadRequest, "metric")
	}
	measurement := metrics[0]
	field := metrics[1]
	sels := make([]monitor.MetricQuerySelect, 0)
	sels = append(sels, monitor.NewMetricQuerySelect(
		monitor.MetricQueryPart{
			Type:   "field",
			Params: []string{field},
		}))
	q := monitor.MetricQuery{
		Database:    o.Database,
		Measurement: measurement,
		Selects:     sels,
	}
	tmp := new(monitor.AlertQuery)
	tmp.Model = q
	alertQuery.AlertQuery = tmp
	createInput.MetricQuery = make([]*monitor.CommonAlertQuery, 0)
	createInput.MetricQuery = append(createInput.MetricQuery, alertQuery)
	return jsonutils.Marshal(createInput), nil
}

type AlertDashBoardListOptions struct {
	options.BaseListOptions
}

func (o *AlertDashBoardListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AlertDashBoardShowOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *AlertDashBoardShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *AlertDashBoardShowOptions) GetId() string {
	return o.ID
}

type AlertDashBoardDeleteOptions struct {
	ID string `json:"-"`
}

func (o *AlertDashBoardDeleteOptions) GetId() string {
	return o.ID
}

func (o *AlertDashBoardDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
