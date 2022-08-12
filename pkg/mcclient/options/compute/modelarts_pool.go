package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type PoolListOptions struct {
	options.BaseListOptions
}

func (opts *PoolListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type PoolIdOption struct {
	ID string `help:"Elasticsearch Id"`
}

func (opts *PoolIdOption) GetId() string {
	return opts.ID
}

func (opts *PoolIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
