package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type ResTagDetailListOptions struct {
		options.BaseListOptions
		RESTYPE string `help:"query res_type = server"`
	}
	R(&ResTagDetailListOptions{}, "restagdetail-list", "List all res tag detail", func(s *mcclient.ClientSession, args *ResTagDetailListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.RESTYPE) > 0 {
			params.Add(jsonutils.NewString(args.RESTYPE), "res_type")
		}

		result, err := modules.ResTagDetails.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ResTagDetails.GetColumns(s))
		return nil
	})
}
