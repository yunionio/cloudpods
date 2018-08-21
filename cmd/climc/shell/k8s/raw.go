package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

type rawGetOpt struct {
	namespaceOptions
	KIND string `help:"resource kind"`
	NAME string `help:"instance name"`
}

type rawDeleteOpt struct {
	rawGetOpt
}

func initRaw() {
	R(&rawGetOpt{}, "k8s-get", "Get k8s resource instance raw info", func(s *mcclient.ClientSession, args *rawGetOpt) error {
		obj, err := k8s.RawResource.Get(s, args.KIND, args.Namespace, args.NAME, nil, args.ClusterContext())
		if err != nil {
			return err
		}
		printObjectYAML(obj)
		return nil
	})

	R(&rawDeleteOpt{}, "k8s-delete", "Delete k8s resource instance", func(s *mcclient.ClientSession, args *rawDeleteOpt) error {
		err := k8s.RawResource.Delete(s, args.KIND, args.Namespace, args.NAME, nil, args.ClusterContext())
		if err != nil {
			return err
		}
		return nil
	})
}
