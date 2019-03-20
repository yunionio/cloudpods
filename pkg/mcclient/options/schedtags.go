package options

import (
	"yunion.io/x/jsonutils"
)

type SchedtagModelListOptions struct {
	BaseListOptions
	Schedtag string `help:"ID or Name of schedtag"`
}

func (o SchedtagModelListOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	return params, nil
}

type SchedtagModelPairOptions struct {
	SCHEDTAG string `help:"Scheduler tag"`
	OBJECT   string `help:"Object id"`
}
