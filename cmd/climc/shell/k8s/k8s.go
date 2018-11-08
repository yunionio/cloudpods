package k8s

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
	"yunion.io/x/onecloud/pkg/util/printutils"
)

func init() {
	// cluster resources
	initCluster()
	initNode()

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

func clusterContext(clusterId string) modules.ManagerContext {
	return modules.ManagerContext{
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

func initK8sClusterResource(kind string, manager modules.Manager) *ShellCommands {
	cmdN := func(suffix string) string {
		return resourceCmdN(kind, suffix)
	}

	// List resource
	listCmd := NewCommand(
		&o.ResourceListOptions{},
		cmdN("list"),
		fmt.Sprintf("List k8s %s", kind),
		func(s *mcclient.ClientSession, args *o.ResourceListOptions) error {
			ret, err := manager.List(s, args.Params())
			if err != nil {
				return err
			}
			PrintListResultTable(ret, manager.(k8s.ListPrinter), s)
			return nil
		},
	)

	// Get resource details
	getCmd := NewCommand(
		&o.ResourceGetOptions{},
		cmdN("show"),
		fmt.Sprintf("Show k8s %s", kind),
		func(s *mcclient.ClientSession, args *o.ResourceGetOptions) error {
			ret, err := manager.Get(s, args.NAME, args.Params())
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		},
	)

	// Delete resource
	deleteCmd := NewCommand(
		&o.ResourceDeleteOptions{},
		cmdN("delete"),
		fmt.Sprintf("Delete k8s %s", kind),
		func(s *mcclient.ClientSession, args *o.ResourceDeleteOptions) error {
			ret := manager.BatchDelete(s, args.NAME, args.Params())
			printBatchResults(ret, manager.GetColumns(s))
			return nil
		},
	)

	return NewShellCommands(cmdN).AddR(listCmd, getCmd, deleteCmd)
}

func initK8sNamespaceResource(kind string, manager modules.Manager) *ShellCommands {
	cmdN := func(suffix string) string {
		return resourceCmdN(kind, suffix)
	}

	// List resource
	listCmd := NewCommand(
		&o.NamespaceResourceListOptions{},
		cmdN("list"),
		fmt.Sprintf("List k8s %s", kind),
		func(s *mcclient.ClientSession, args *o.NamespaceResourceListOptions) error {
			ret, err := manager.List(s, args.Params())
			if err != nil {
				return err
			}
			PrintListResultTable(ret, manager.(k8s.ListPrinter), s)
			return nil
		},
	)

	// Get resource details
	getCmd := NewCommand(
		&o.NamespaceResourceGetOptions{},
		cmdN("show"),
		fmt.Sprintf("Show k8s %s", kind),
		func(s *mcclient.ClientSession, args *o.NamespaceResourceGetOptions) error {
			ret, err := manager.Get(s, args.NAME, args.Params())
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		},
	)

	// Delete resource
	deleteCmd := NewCommand(
		&o.NamespaceResourceDeleteOptions{},
		cmdN("delete"),
		fmt.Sprintf("Delete k8s %s", kind),
		func(s *mcclient.ClientSession, args *o.NamespaceResourceDeleteOptions) error {
			ret := manager.BatchDelete(s, args.NAME, args.Params())
			printBatchResults(ret, manager.GetColumns(s))
			return nil
		},
	)
	return NewShellCommands(cmdN).AddR(listCmd, getCmd, deleteCmd)
}
