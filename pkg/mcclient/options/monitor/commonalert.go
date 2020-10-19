package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type CommonAlertListOptions struct {
	options.BaseListOptions
	// 报警类型
	AlertType string `help:"common alert type" choices:"normal|system"`
	Level     string `help:"common alert notify level" choices:"normal|important|fatal"`
}

func (o *CommonAlertListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type CommonAlertShowOptions struct {
	ID string `help:"ID of alart " json:"-"`
}

func (o *CommonAlertShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *CommonAlertShowOptions) GetId() string {
	return o.ID
}

type CommonAlertDeleteOptions struct {
	ID    []string `help:"ID of alart"`
	Force bool     `help:"force to delete alert"`
}

func (o *CommonAlertDeleteOptions) GetIds() []string {
	return o.ID
}

func (o *CommonAlertDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
