package monitor

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertRecordListOptions struct {
	options.BaseListOptions

	AlertId  string   `help:"id of alert"`
	Level    string   `help:"alert level"`
	State    string   `help:"alert state"`
	ResTypes []string `json:"res_types"`
	ResName  string   `json:"res_name"`
	Alerting bool     `json:"alerting"`
}

func (o *AlertRecordListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AlertRecordShowOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *AlertRecordShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *AlertRecordShowOptions) GetId() string {
	return o.ID
}

type AlertRecordTotalOptions struct {
	ID string `help:"total-alert" json:"-"`
	options.BaseListOptions
}

func (o *AlertRecordTotalOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

func (o *AlertRecordTotalOptions) GetId() string {
	return o.ID
}

type AlertRecordShieldListOptions struct {
	options.BaseListOptions

	AlertId   string   `help:"id of alert"`
	Alertname string   `json:"alertname"`
	ResName   string   `json:"res_name"`
	ResTypes  []string `json:"res_types"`
}

func (o *AlertRecordShieldListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AlertRecordShieldShowOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *AlertRecordShieldShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *AlertRecordShieldShowOptions) GetId() string {
	return o.ID
}

type AlertRecordShieldDeleteOptions struct {
	ID string `json:"-"`
}

func (o *AlertRecordShieldDeleteOptions) GetId() string {
	return o.ID
}

func (o *AlertRecordShieldDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AlertRecordShieldCreateOptions struct {
	apis.ScopedResourceCreateInput

	AlertId string `json:"alert_id" help:"common alert Id" required:"true"`
	ResType string `json:"res_type" help:"resource tyge" choices:"host|guest|redis|oss|rds|cloudaccount"`
	ResName string `json:"res_name" help:"resource name" required:"true"`

	ShieldPeriod string `json:"shield_period" help:"shield time eg:'1m,2h'" required:"true"`
}

func (o *AlertRecordShieldCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	duration, err := time.ParseDuration(o.ShieldPeriod)
	if err != nil {
		return nil, errors.Wrap(err, "parse shield_period err")
	}
	startTime := time.Now()
	endTime := startTime.Add(duration)
	params.Add(jsonutils.NewTimeString(startTime), "start_time")
	params.Add(jsonutils.NewTimeString(endTime), "end_time")
	return params, nil
}
