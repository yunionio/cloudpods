package suggestion

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AnalysisPredictConfigOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
	options.BaseListOptions
	QueryType string `help:"query_type of the analysis" choices:"expense_trend" json:"query_type"`
	StartDate string `help:"start_date of the analysis" json:"start_date"`
	EndDate   string `help:"end_date of the analysis" json:"end_date"`
	DataType  string `help:"data_type of the analysis" choices:"day|month" json:"data_type"`
}

func (o *AnalysisPredictConfigOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

func (o *AnalysisPredictConfigOptions) GetId() string {
	return o.ID
}
