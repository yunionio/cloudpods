package k8s

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initReleaseApps() {
	cmdN := func(app string) string {
		return resourceCmdN(fmt.Sprintf("app-%s", app), "create")
	}
	R(&o.AppCreateOptions{}, cmdN("meter"), "Create yunion meter helm release", func(s *mcclient.ClientSession, args *o.AppCreateOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.MeterReleaseApps.Create(s, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.AppCreateOptions{}, cmdN("servicetree"), "Create yunion servicetree helm release", func(s *mcclient.ClientSession, args *o.AppCreateOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.ServicetreeReleaseApps.Create(s, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.AppCreateOptions{}, cmdN("notify"), "Create yunion notify helm release", func(s *mcclient.ClientSession, args *o.AppCreateOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.NotifyReleaseApps.Create(s, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
