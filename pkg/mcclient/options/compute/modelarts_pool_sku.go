package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ModelartsPoolSkuListOptions struct {
	options.BaseListOptions
}

func (opts *ModelartsPoolSkuListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}
