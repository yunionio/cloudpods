package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ModelartsPoolListOptions struct {
	options.BaseListOptions
}

func (opts *ModelartsPoolListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ModelartsPoolIdOption struct {
	ID string `help:"Elasticsearch Id"`
}

func (opts *ModelartsPoolIdOption) GetId() string {
	return opts.ID
}

func (opts *ModelartsPoolIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ModelartsCreateOption struct {
	Name string `help:"name"`
}

func (opts *ModelartsCreateOption) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}
