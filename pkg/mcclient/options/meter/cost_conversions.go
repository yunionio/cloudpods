package meter

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type CostConversionListOptions struct {
	options.BaseListOptions
}

func (opt *CostConversionListOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type CostConversionCreateOptions struct {
	Name string

	IsPublicCloud   string `help:"public cloud filter of cost conversion" json:"is_public_cloud"`
	Brand           string `help:"brand filter of cost conversion" json:"brand"`
	CloudaccountId  string `help:"cloudaccount filter of cost conversion" json:"cloudaccount_id"`
	CloudproviderId string `help:"cloudprovider filter of cost conversion" json:"cloudprovider_id"`
	DomainIdFilter  string `help:"domain filter of cost conversion" json:"domain_id_filter"`

	EnableDate string  `help:"enable date of conversion ratio, example:202107" json:"ratio"`
	Ratio      float64 `help:"cost conversion ratio" json:"ratio"`
}

func (opt *CostConversionCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type CostConversionUpdateOptions struct {
	ID string `help:"ID of cost conversion" json:"-"`

	Ratio float64 `help:"cost conversion ratio" json:"ratio"`
}

func (opt *CostConversionUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *CostConversionUpdateOptions) GetId() string {
	return opt.ID
}

type CostConversionDeleteOptions struct {
	ID string `help:"ID of cost conversion" json:"-"`
}

func (opt *CostConversionDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *CostConversionDeleteOptions) GetId() string {
	return opt.ID
}
