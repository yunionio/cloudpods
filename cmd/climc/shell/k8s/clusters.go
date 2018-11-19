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

func initCluster() {
	cmdN := func(suffix string) string {
		return kubeResourceCmdN("cluster", suffix)
	}

	R(&o.ClusterListOptions{}, cmdN("list"), "List k8s infra clusters", func(s *mcclient.ClientSession, args *o.ClusterListOptions) error {
		result, err := k8s.Clusters.List(s, args.Params())
		if err != nil {
			return err
		}
		printList(result, k8s.Clusters.GetColumns(s))
		return nil
	})

	R(&o.ClusterCreateOptions{}, cmdN("create"), "Create k8s cluster", func(s *mcclient.ClientSession, args *o.ClusterCreateOptions) error {
		cluster, err := k8s.Clusters.Create(s, args.Params())
		if err != nil {
			return err
		}
		printObject(cluster)
		return nil
	})

	R(&o.ClusterImportOptions{}, cmdN("import"), "Import exists YKE deployed kubernetes cluster", func(s *mcclient.ClientSession, args *o.ClusterImportOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}

		cluster, err := k8s.Clusters.PerformAction(s, args.NAME, "import", params)
		if err != nil {
			return err
		}
		printObject(cluster)
		return nil
	})

	R(&o.ClusterUpdateOptions{}, cmdN("update"), "Update k8s cluster", func(s *mcclient.ClientSession, args *o.ClusterUpdateOptions) error {
		cluster, err := k8s.Clusters.Update(s, args.NAME, args.Params())
		if err != nil {
			return err
		}
		printObject(cluster)
		return nil
	})

	R(&o.ClusterDeployOptions{}, cmdN("deploy"), "Start deploy a cluster", func(s *mcclient.ClientSession, args *o.ClusterDeployOptions) error {
		ret, err := k8s.Clusters.PerformAction(s, args.ID, "deploy", args.Params())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("show"), "Show details of a cluster", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		result, err := k8s.Clusters.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&o.ClusterDeleteOptions{}, cmdN("delete"), "Delete cluster", func(s *mcclient.ClientSession, args *o.ClusterDeleteOptions) error {
		ret := k8s.Clusters.BatchDeleteWithParam(s, args.ID, nil, nil)
		printBatchResults(ret, k8s.Clusters.GetColumns(s))
		return nil
	})

	R(&o.ClusterKubeconfigOptions{}, cmdN("kubeconfig"), "Generate kubeconfig of a cluster", func(s *mcclient.ClientSession, args *o.ClusterKubeconfigOptions) error {
		ret, err := k8s.Clusters.PerformAction(s, args.ID, "generate-kubeconfig", args.Params())
		if err != nil {
			return err
		}
		conf, err := ret.GetString("kubeconfig")
		if err != nil {
			return err
		}
		fmt.Println(conf)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("engineconfig"), "Get kubernetes engine config of a cluster", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.Clusters.GetSpecific(s, args.ID, "engine-config", nil)
		if err != nil {
			return err
		}
		conf, err := ret.GetString("config")
		if err != nil {
			return err
		}
		fmt.Println(conf)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("webhookauthurl"), "Get cluster kubernetes api server webhook auth url", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.Clusters.GetSpecific(s, args.ID, "webhook-auth-url", nil)
		if err != nil {
			return err
		}
		url, err := ret.GetString("url")
		if err != nil {
			return err
		}
		fmt.Println(url)
		return nil
	})

	R(&o.ClusterAddNodesOptions{}, cmdN("addnodes"), "Add nodes to cluster", func(s *mcclient.ClientSession, args *o.ClusterAddNodesOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.Clusters.PerformAction(s, args.ID, "add-nodes", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.ClusterDeleteNodesOptions{}, cmdN("delete-nodes"), "Delete nodes in cluster", func(s *mcclient.ClientSession, args *o.ClusterDeleteNodesOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.Clusters.PerformAction(s, args.ID, "delete-nodes", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("engineconfig-edit"), "Edit and update kubernetes engine config of a cluster", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.Clusters.GetSpecific(s, args.ID, "engine-config", nil)
		if err != nil {
			return err
		}
		conf, err := ret.GetString("config")
		if err != nil {
			return err
		}
		tempfile, err := ioutil.TempFile("", fmt.Sprintf("%s-engineconfig", args.ID))
		if err != nil {
			return err
		}
		defer os.Remove(tempfile.Name())

		if _, err := tempfile.Write([]byte(conf)); err != nil {
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

		params := jsonutils.NewDict()
		configBytes, err := ioutil.ReadFile(tempfile.Name())
		if err != nil {
			return err
		}
		params.Add(jsonutils.NewString(string(configBytes)), "config")
		result, err := k8s.Clusters.PerformAction(s, args.ID, "update-engine-config", params)
		if err != nil {
			return err
		}
		conf, err = result.GetString("config")
		if err != nil {
			return err
		}
		fmt.Println(conf)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("sync-config"), "Sync kubernetes cluster by engineconfig", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.Clusters.PerformAction(s, args.ID, "sync-config", nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.ClusterRestartAgentsOptions{}, cmdN("restart-agent"), "Restart node agents in cluster", func(s *mcclient.ClientSession, args *o.ClusterRestartAgentsOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.Clusters.PerformAction(s, args.ID, "restart-agent", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
