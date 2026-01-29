package llm

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DifyListOptions struct {
	LLMBaseListOptions

	DifySku string `help:"filter by dify sku"`
}

func (o *DifyListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(o)
	if err != nil {
		return nil, err
	}
	if o.Used != nil {
		params.Set("unused", jsonutils.JSONFalse)
	}
	return params, nil
}

type DifyShowOptions struct {
	options.BaseShowOptions
}

func (o *DifyShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifyCreateOptions struct {
	LLMBaseCreateOptions

	DIFY_SKU_ID string `help:"dify sku id or name" json:"dify_sku_id"`
}

func (o *DifyCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o)

	if len(o.Net) > 0 {
		nets := make([]*computeapi.NetworkConfig, 0)
		for i, n := range o.Net {
			net, err := cmdline.ParseNetworkConfig(n, i)
			if err != nil {
				return nil, errors.Wrapf(err, "parse network config %s", n)
			}
			nets = append(nets, net)
		}
		params.(*jsonutils.JSONDict).Add(jsonutils.Marshal(nets), "nets")
	}

	return params, nil
}

func (o *DifyCreateOptions) GetCountParam() int {
	return o.Count
}

type DifyDeleteOptions struct {
	options.BaseIdOptions
}

func (o *DifyDeleteOptions) GetId() string {
	return o.ID
}

func (o *DifyDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifyStartOptions struct {
	options.BaseIdsOptions
}

func (o *DifyStartOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type DifyStopOptions struct {
	options.BaseIdsOptions
}

func (o *DifyStopOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
