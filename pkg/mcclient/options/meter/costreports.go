package meter

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type CostReportListOptions struct {
	options.BaseListOptions
}

func (opt *CostReportListOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type CostReportCreateOptions struct {
	Scope string `help:"scope of cost report" json:"scope"`

	PeriodType string   `help:"period of cost report send, example:month/week/day" json:"period_type"`
	Day        int      `help:"day of cost report send" json:"day"`
	ColonTimer string   `help:"hour and minute of cost report send, example:HH:mm" json:"colon_timer"`
	Emails     []string `help:"emails of cost report send" json:"emails"`
	StartRun   bool     `help:"whether cost report sends instantly" json:"start_run"`
}

func (opt *CostReportCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type CostReportUpdateOptions struct {
	ID string `help:"ID of cost report" json:"-"`

	PeriodType string   `help:"period of cost report send, example:month/week/day" json:"period_type"`
	Day        int      `help:"day of cost report send" json:"day"`
	ColonTimer string   `help:"hour and minute of cost report send, example:HH:mm" json:"colon_timer"`
	Emails     []string `help:"emails of cost report send" json:"emails"`
	StartRun   bool     `help:"whether cost report sends instantly" json:"start_run"`
}

func (opt *CostReportUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *CostReportUpdateOptions) GetId() string {
	return opt.ID
}

type CostReportDeleteOptions struct {
	ID string `help:"ID of cost report" json:"-"`
}

func (opt *CostReportDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *CostReportDeleteOptions) GetId() string {
	return opt.ID
}
