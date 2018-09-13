package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initChart() {
	cmdN := func(suffix string) string {
		return resourceCmdN("chart", suffix)
	}

	R(&o.ChartListOptions{}, cmdN("list"), "List k8s helm global charts", func(s *mcclient.ClientSession, args *o.ChartListOptions) error {
		charts, err := k8s.Charts.List(s, args.Params())
		if err != nil {
			return err
		}

		PrintListResultTable(charts, k8s.Charts, s)
		return nil
	})

	R(&o.ChartGetOptions{}, cmdN("show"), "Show details of a chart", func(s *mcclient.ClientSession, args *o.ChartGetOptions) error {
		chart, err := k8s.Charts.Get(s, args.NAME, args.Params())
		if err != nil {
			return err
		}
		printObjectYAML(chart)
		return nil
	})
}
