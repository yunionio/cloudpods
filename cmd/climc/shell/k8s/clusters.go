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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initKubeCluster() {
	cmdN := func(action string) string {
		return fmt.Sprintf("kubecluster-%s", action)
	}
	R(&o.ClusterListOptions{}, cmdN("list"), "List k8s clusters", func(s *mcclient.ClientSession, args *o.ClusterListOptions) error {
		result, err := k8s.KubeClusters.List(s, args.Params())
		if err != nil {
			return err
		}
		printList(result, k8s.KubeClusters.GetColumns(s))
		return nil
	})

	R(&o.IdentOptions{}, cmdN("show"), "Show details of a cluster", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		result, err := k8s.KubeClusters.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&o.KubeClusterCreateOptions{}, cmdN("create"), "Create k8s cluster", func(s *mcclient.ClientSession, args *o.KubeClusterCreateOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		cluster, err := k8s.KubeClusters.Create(s, params)
		if err != nil {
			return err
		}
		printObject(cluster)
		return nil
	})

	R(&o.KubeClusterImportOptions{}, cmdN("import"), "Import k8s cluster", func(s *mcclient.ClientSession, args *o.KubeClusterImportOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		cluster, err := k8s.KubeClusters.Create(s, params)
		if err != nil {
			return err
		}
		printObject(cluster)
		return nil
	})

	R(&o.ClusterDeleteOptions{}, cmdN("delete"), "Delete cluster", func(s *mcclient.ClientSession, args *o.ClusterDeleteOptions) error {
		ret := k8s.KubeClusters.BatchDeleteWithParam(s, args.ID, nil, nil)
		printBatchResults(ret, k8s.KubeClusters.GetColumns(s))
		return nil
	})

	R(&o.KubeClusterAddMachinesOptions{}, cmdN("add-machines"), "Add machines to cluster", func(s *mcclient.ClientSession, args *o.KubeClusterAddMachinesOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.KubeClusters.PerformAction(s, args.ID, "add-machines", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.KubeClusterDeleteMachinesOptions{}, cmdN("delete-machines"), "Delete machines in cluster", func(s *mcclient.ClientSession, args *o.KubeClusterDeleteMachinesOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.KubeClusters.PerformAction(s, args.ID, "delete-machines", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("terminate"), "Terminate cluster", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.KubeClusters.PerformAction(s, args.ID, "terminate", nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("kubeconfig"), "Generate kubeconfig of a cluster", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.KubeClusters.GetSpecific(s, args.ID, "kubeconfig", nil)
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

	R(&o.IdentOptions{}, cmdN("addons"), "Get addon manifest of a cluster", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.KubeClusters.GetSpecific(s, args.ID, "addons", nil)
		if err != nil {
			return err
		}
		conf, err := ret.GetString("addons")
		if err != nil {
			return err
		}
		fmt.Println(conf)
		return nil
	})

	R(&o.ClusterK8sVersions{}, cmdN("k8s-versions"), "Get kubernetes deployable versions", func(s *mcclient.ClientSession, args *o.ClusterK8sVersions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.PROVIDER), "provider")
		ret, err := k8s.KubeClusters.Get(s, "k8s-versions", params)
		if err != nil {
			return err
		}
		fmt.Println(ret.String())
		return nil
	})

	R(&o.ClusterK8sVersions{}, cmdN("usable-instances"), "Get deploy usable instance", func(s *mcclient.ClientSession, args *o.ClusterK8sVersions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.PROVIDER), "provider")
		ret, err := k8s.KubeClusters.Get(s, "usable-instances", params)
		if err != nil {
			return err
		}
		fmt.Println(ret.String())
		return nil
	})

	R(&o.ClusterCheckOptions{}, cmdN("check-system-ready"), "Check system cluster status", func(s *mcclient.ClientSession, args *o.ClusterCheckOptions) error {
		ret, err := k8s.KubeClusters.PerformClassAction(s, "check-system-ready", nil)
		if err != nil {
			return err
		}
		fmt.Println(ret.String())
		return nil
	})

	R(&o.IdentOptions{}, cmdN("apply-addons"), "Apply base requirements addons", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.KubeClusters.PerformAction(s, args.ID, "apply-addons", nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("syncstatus"), "Sync cluster status", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.KubeClusters.PerformAction(s, args.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("public"), "Make cluster public", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.KubeClusters.PerformAction(s, args.ID, "public", nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("private"), "Make cluster private", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.KubeClusters.PerformAction(s, args.ID, "private", nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
