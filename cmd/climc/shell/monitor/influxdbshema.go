package monitor

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	aN := cmdN("influxdb")

	R(&options.InfluxdbShemaListOptions{}, aN("list"), "list influxdb info",
		func(s *mcclient.ClientSession, args *options.InfluxdbShemaListOptions) error {
			retn := jsonutils.NewDict()
			retn.Add(jsonutils.NewStringArray(api.PROPERTY_TYPE), "ID")
			printObject(retn)
			return nil
		})

	R(&options.InfluxdbShemaShowOptions{}, aN("show"), "Show influxdb info",
		func(s *mcclient.ClientSession, args *options.InfluxdbShemaShowOptions) error {
			params, _ := args.Params()
			ret, err := modules.InfluxdbShemaManager.Get(s, args.ID, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
}
