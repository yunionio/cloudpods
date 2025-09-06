package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DifyCreateOptions struct {
	PodCreateOptions
}

func (o *DifyCreateOptions) Params() (jsonutils.JSONObject, error) {
	input, err := o.PodCreateOptions.Params()
	if err != nil {
		return nil, err
	}

	return jsonutils.Marshal(input), nil
}

type DifyListOptions struct {
	options.BaseListOptions
	GuestId string `json:"guest_id" help:"guest(pod) id or name"`
}

func (o *DifyListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type DifyIdOptions struct {
	ID string `help:"ID or name of the dify" json:"-"`
}

func (o *DifyIdOptions) GetId() string {
	return o.ID
}

func (o *DifyIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DifyShowOptions struct {
	DifyIdOptions
}
