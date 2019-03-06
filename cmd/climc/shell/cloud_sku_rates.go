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
		dataIds := jsonutils.NewArray()
		for _, n := range args.ParamIds {
			dataIds.Add(jsonutils.NewString(n))
		}

		dataKeys := jsonutils.NewArray()
		for _, n := range args.ParamKeys {
			dataKeys.Add(jsonutils.NewString(n))
		}

		params := jsonutils.NewDict()
		params.Add(dataIds, "param_ids")
		params.Add(dataKeys, "param_keys")

		result, err := modules.CloudSkuRates.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.CloudSkuRates.GetColumns(s))
		return nil
	})
}
