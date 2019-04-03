package options

import (
	"fmt"

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

type SchedtagSetOptions struct {
	ID       string   `help:"Id or name of resource"`
	Schedtag []string `help:"Ids of schedtag"`
}

func (o SchedtagSetOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	for idx, tag := range o.Schedtag {
		params.Add(jsonutils.NewString(tag), fmt.Sprintf("schedtag.%d", idx))
	}
	return params, nil
}
