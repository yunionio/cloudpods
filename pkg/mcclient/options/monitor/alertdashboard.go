package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertDashBoardCreateOptions struct {
	apis.ScopedResourceCreateInput
	NAME    string `help:"Name of bashboard"`
	Refresh string `help:"dashboard query refresh priod e.g. 1m|5m"`
}

func (o *AlertDashBoardCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
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
