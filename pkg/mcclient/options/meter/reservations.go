package meter

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ReservationListOptions struct {
	options.BaseListOptions

	CloudaccountId   string `help:"cloudaccount id of reservation" json:"cloudaccount_id"`
	ResourceType     string `help:"resource type of reservation" json:"resource_type"`
	ReservationYears string `help:"number of reservation years" json:"reservation_years"`
	LookbackDays     string `help:"number of previous days will be consider" json:"lookback_days"`
	PaymentOption    string `help:"payment option of reservation, example:all_upfront/partial_upfront/no_upfront" json:"payment_option"`
	OfferingClass    string `help:"offering class of reservation, example:standard/convertible" json:"offering_class"`
}

func (opt *ReservationListOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}
