package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	type CloudSkuRateListOptions struct {
		ParamIds  []string `help:"param_id of the cloudSkuRate" nargs:"+"`
		ParamKeys []string `help:"param_key of the cloudSkuRate" nargs:"+"`
	}

	R(&CloudSkuRateListOptions{}, "cloud-sku-rate-list", "list cloud-sku-rates", func(s *mcclient.ClientSession, args *CloudSkuRateListOptions) error {

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewStringArray(args.ParamIds), "param_ids")
		params.Add(jsonutils.NewStringArray(args.ParamKeys), "param_keys")

		result, err := modules.CloudSkuRates.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.CloudSkuRates.GetColumns(s))
		return nil
	})
}
