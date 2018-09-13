package k8s

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

type rawOpt struct {
	namespaceOptions
	KIND string `help:"resource kind"`
	NAME string `help:"instance name"`
}

type rawGetOpt struct {
	rawOpt
	Output string `help:"Output format" short-token:"o" default:"yaml" choices:"json|yaml"`
}

type rawDeleteOpt struct {
	rawOpt
}

type rawPutOpt struct {
	rawOpt
	File string `help:"Resource json description file" short-token:"f"`
}

func initRaw() {
	R(&rawGetOpt{}, "k8s-get", "Get k8s resource instance raw info", func(s *mcclient.ClientSession, args *rawGetOpt) error {
		obj, err := k8s.RawResource.Get(s, args.KIND, args.Namespace, args.NAME, args.Cluster)
		if err != nil {
			return err
		}
		if args.Output == "json" {
			fmt.Println(obj.PrettyString())
		} else {
			printObjectYAML(obj)
		}
		return nil
	})

	R(&rawDeleteOpt{}, "k8s-delete", "Delete k8s resource instance", func(s *mcclient.ClientSession, args *rawDeleteOpt) error {
		err := k8s.RawResource.Delete(s, args.KIND, args.Namespace, args.NAME, args.Cluster)
		if err != nil {
			return err
		}
		return nil
	})

	R(&rawPutOpt{}, "k8s-put", "Update k8s resource instance", func(s *mcclient.ClientSession, args *rawPutOpt) error {
		content, err := ioutil.ReadFile(args.File)
		if err != nil {
			return err
		}
		body, err := jsonutils.Parse(content)
		if err != nil {
			return err
		}
		err = k8s.RawResource.Put(s, args.KIND, args.Namespace, args.NAME, body, args.Cluster)
		if err != nil {
			return err
		}
		return nil
	})
}
