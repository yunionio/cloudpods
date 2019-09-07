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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
	"yunion.io/x/onecloud/pkg/util/printutils"
)

func init() {
	// cluster resources
	initCluster()
	initKubeCluster()
	initKubeMachine()
	initNode()
	initKubeCerts()

	// helm resources
	initTiller()
	initRepo()
	initChart()
	initRelease()
	initReleaseApps()

	// kubernetes original resources
	initRaw()
	initConfigMap()
	initDeployment()
	initStatefulset()
	initPod()
	initService()
	initIngress()
	initNamespace()
	initK8sNode()
	initSecret()
	initStorageClass()
	initPV()
	initPVC()
	initJob()
	initCronJob()

	initApp()
}

var (
	R                 = shell.R
	printList         = printutils.PrintJSONList
	printObject       = printutils.PrintJSONObject
	printBatchResults = printutils.PrintJSONBatchResults
)

func resourceCmdN(prefix, suffix string) string {
	return fmt.Sprintf("k8s-%s-%s", prefix, suffix)
}

func kubeResourceCmdN(prefix, suffix string) string {
	return fmt.Sprintf("kube-%s-%s", prefix, suffix)
}

func clusterContext(clusterId string) modulebase.ManagerContext {
	return modulebase.ManagerContext{
		InstanceManager: k8s.Clusters,
		InstanceId:      clusterId,
	}
}

func printObjectYAML(obj jsonutils.JSONObject) {
	fmt.Println(obj.YAMLString())
}

type Cmd struct {
	Options  interface{}
	Command  string
	Desc     string
	Callback interface{}
}

func NewCommand(options interface{}, command string, desc string, callback interface{}) *Cmd {
	return &Cmd{
		Options:  options,
		Command:  command,
		Desc:     desc,
		Callback: callback,
	}
}

func (c Cmd) R() {
	R(c.Options, c.Command, c.Desc, c.Callback)
}

type ShellCommands struct {
	Commands           []*Cmd
	CommandNameFactory func(suffix string) string
}

func NewShellCommands(cmdN func(suffix string) string) *ShellCommands {
	c := &ShellCommands{
		CommandNameFactory: cmdN,
	}
	c.Commands = make([]*Cmd, 0)
	return c
}

func (c *ShellCommands) AddR(rs ...*Cmd) *ShellCommands {
	for _, r := range rs {
		r.R()
		c.Commands = append(c.Commands, r)
	}
	return c
}

func initK8sClusterResource(kind string, manager modulebase.Manager) *ShellCommands {
	cmdN := NewCmdNameFactory(kind)
	return NewShellCommands(cmdN.Do).AddR(
		NewK8sResourceListCmd(cmdN, manager),
		NewK8sResourceGetCmd(cmdN, manager),
		NewK8sResourceDeleteCmd(cmdN, manager),
	)
}

func NewK8sResourceListCmd(cmdN CmdNameFactory, manager modulebase.Manager) *Cmd {
	return NewCommand(
		&o.ResourceListOptions{},
		cmdN.Do("list"),
		fmt.Sprintf("List k8s %s", cmdN.Kind),
		func(s *mcclient.ClientSession, args *o.ResourceListOptions) error {
			ret, err := manager.List(s, args.Params())
			if err != nil {
				return err
			}
			PrintListResultTable(ret, manager.(k8s.ListPrinter), s)
			return nil
		},
	)
}

type CmdNameFactory struct {
	Kind string
	Do   func(string) string
}

func NewCmdNameFactory(kind string) CmdNameFactory {
	return CmdNameFactory{
		Kind: kind,
		Do: func(suffix string) string {
			return resourceCmdN(kind, suffix)
		},
	}
}

func NewK8sNsResourceListCmd(cmdN CmdNameFactory, manager modulebase.Manager) *Cmd {
	return NewCommand(
		&o.NamespaceResourceListOptions{},
		cmdN.Do("list"),
		fmt.Sprintf("List k8s %s", cmdN.Kind),
		func(s *mcclient.ClientSession, args *o.NamespaceResourceListOptions) error {
			ret, err := manager.List(s, args.Params())
			if err != nil {
				return err
			}
			PrintListResultTable(ret, manager.(k8s.ListPrinter), s)
			return nil
		},
	)
}

func NewK8sResourceGetCmd(cmdN CmdNameFactory, manager modulebase.Manager) *Cmd {
	return NewCommand(
		&o.ResourceGetOptions{},
		cmdN.Do("show"),
		fmt.Sprintf("Show k8s %s", cmdN.Kind),
		func(s *mcclient.ClientSession, args *o.ResourceGetOptions) error {
			ret, err := manager.Get(s, args.NAME, args.Params())
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		},
	)
}

func NewK8sNsResourceGetCmd(cmdN CmdNameFactory, manager modulebase.Manager) *Cmd {
	return NewCommand(
		&o.NamespaceResourceGetOptions{},
		cmdN.Do("show"),
		fmt.Sprintf("Show k8s %s", cmdN.Kind),
		func(s *mcclient.ClientSession, args *o.NamespaceResourceGetOptions) error {
			ret, err := manager.Get(s, args.NAME, args.Params())
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		},
	)
}

func NewK8sResourceDeleteCmd(cmdN CmdNameFactory, manager modulebase.Manager) *Cmd {
	return NewCommand(
		&o.ResourceDeleteOptions{},
		cmdN.Do("delete"),
		fmt.Sprintf("Delete k8s %s", cmdN.Kind),
		func(s *mcclient.ClientSession, args *o.ResourceDeleteOptions) error {
			ret := manager.BatchDelete(s, args.NAME, args.Params())
			printBatchResults(ret, manager.GetColumns(s))
			return nil
		},
	)
}

func NewK8sNsResourceDeleteCmd(cmdN CmdNameFactory, manager modulebase.Manager) *Cmd {
	deleteCmd := NewCommand(
		&o.NamespaceResourceDeleteOptions{},
		cmdN.Do("delete"),
		fmt.Sprintf("Delete k8s %s", cmdN.Kind),
		func(s *mcclient.ClientSession, args *o.NamespaceResourceDeleteOptions) error {
			ret := manager.BatchDelete(s, args.NAME, args.Params())
			printBatchResults(ret, manager.GetColumns(s))
			return nil
		},
	)
	return deleteCmd
}

func initK8sNamespaceResource(kind string, manager modulebase.Manager) *ShellCommands {
	cmdN := NewCmdNameFactory(kind)
	return NewShellCommands(cmdN.Do).AddR(
		NewK8sNsResourceListCmd(cmdN, manager),
		NewK8sNsResourceGetCmd(cmdN, manager),
		NewK8sNsResourceDeleteCmd(cmdN, manager),
	)
}
