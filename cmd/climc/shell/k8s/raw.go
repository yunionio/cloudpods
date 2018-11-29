package k8s

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

type RawOpt struct {
	o.NamespaceWithClusterOptions
	KIND string `help:"resource kind"`
	NAME string `help:"instance name"`
}

type rawGetOpt struct {
	RawOpt
	Output string `help:"Output format" short-token:"o" default:"yaml" choices:"json|yaml"`
}

type rawDeleteOpt struct {
	RawOpt
}

type rawPutOpt struct {
	RawOpt
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

	doPut := func(s *mcclient.ClientSession, args *rawPutOpt) error {
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
	}

	R(&rawPutOpt{}, "k8s-put", "Update k8s resource instance", func(s *mcclient.ClientSession, args *rawPutOpt) error {
		return doPut(s, args)
	})

	R(&RawOpt{}, "k8s-edit", "Edit k8s resource instance", func(s *mcclient.ClientSession, args *RawOpt) error {
		obj, err := k8s.RawResource.Get(s, args.KIND, args.Namespace, args.NAME, args.Cluster)
		if err != nil {
			return err
		}
		tempfile, err := ioutil.TempFile("", fmt.Sprintf("k8s-%s-%s.json", args.KIND, args.NAME))
		if err != nil {
			return err
		}
		defer os.Remove(tempfile.Name())
		if _, err := tempfile.Write([]byte(obj.PrettyString())); err != nil {
			return err
		}
		if err := tempfile.Close(); err != nil {
			return err
		}

		cmd := exec.Command("vim", tempfile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return err
		}
		return doPut(s, &rawPutOpt{RawOpt: *args, File: tempfile.Name()})
	})
}
