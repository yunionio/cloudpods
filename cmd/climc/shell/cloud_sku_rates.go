package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	type CloudSkuRateListOptions struct {
		PARAMKEYS string `help:"param_keys of the cloudSkuRate"`
	}

	R(&CloudSkuRateListOptions{}, "cloud-sku-rate-list", "list cloud-sku-rates", func(s *mcclient.ClientSession, args *CloudSkuRateListOptions) error {

		params := jsonutils.NewDict()

		params.Add(jsonutils.NewString(args.PARAMKEYS), "param_keys")

		result, err := modules.CloudSkuRates.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.CloudSkuRates.GetColumns(s))
		return nil
	})
}
