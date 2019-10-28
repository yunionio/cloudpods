// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/ghodss/yaml"

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
		jsonBytes, err := yaml.YAMLToJSON(content)
		if err != nil {
			return err
		}
		body, err := jsonutils.Parse(jsonBytes)
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
		yamlBytes, err := k8s.RawResource.GetYAML(s, args.KIND, args.Namespace, args.NAME, args.Cluster)
		if err != nil {
			return err
		}
		tempfile, err := ioutil.TempFile("", fmt.Sprintf("k8s-%s-%s.yaml", args.KIND, args.NAME))
		if err != nil {
			return err
		}
		defer os.Remove(tempfile.Name())
		if _, err := tempfile.Write(yamlBytes); err != nil {
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
