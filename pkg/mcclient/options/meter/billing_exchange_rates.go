package meter

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type BillingExchangeRateListOptions struct {
	options.BaseListOptions
}

func (opt *BillingExchangeRateListOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type BillingExchangeRateUpdateOptions struct {
	ID string `help:"ID of billing exchange rate" json:"-"`

	Rate float64 `help:"exchange rate" json:"rate"`
}

func (opt *BillingExchangeRateUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *BillingExchangeRateUpdateOptions) GetId() string {
	return opt.ID
}
