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
	ID string `help:"ModelartsPool Id"`
}

func (opts *ModelartsPoolIdOption) GetId() string {
	return opts.ID
}

func (opts *ModelartsPoolIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ModelartsPoolCreateOption struct {
	Name         string `help:"Name"`
	ManagerId    string `help:"Manager Id"`
	InstanceType string `help:"Instance Type"`
	IsTrain      *bool
	IsInfer      *bool
	IsNotebook   *bool
}

func (opts *ModelartsPoolCreateOption) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ModelartsPoolDeleteOption struct {
	ID     string `help:"Id"`
	PoolId string `help:"Pool Id"`
}

func (opts *ModelartsPoolDeleteOption) GetId() string {
	return opts.ID
}

func (opts *ModelartsPoolDeleteOption) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}
