package shell

import (
	"fmt"

	"github.com/yunionio/jsonutils"

	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules/k8s"
)

func init() {
	initCluster()
	initNode()
}

func initCluster() {
	cmdN := func(suffix string) string {
		return fmt.Sprintf("k8s-cluster-%s", suffix)
	}
	type listOpt struct {
		BaseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s clusters", func(s *mcclient.ClientSession, args *listOpt) error {
		args.Details = true
		params := FetchPagingParams(args.BaseListOptions)
		result, err := k8s.Clusters.List(s, params)
		if err != nil {
			return err
		}
		printList(result, k8s.Clusters.GetColumns(s))
		return nil
	})

	type createOpt struct {
		NAME       string `help:"Name of cluster"`
		Mode       string `help:"Cluster mode" choices:"internal"`
		K8sVersion string `help:"Cluster kubernetes components version" choices:"v1.8.10|v1.9.5|v1.10.0" default:"v1.9.5"`
		InfraImage string `help:"Cluster kubelet infra container image"`
		Cidr       string `help:"Cluster service CIDR, e.g. 10.43.0.0/16"`
		Domain     string `help:"Cluster pod domain, e.g. cluster.local"`
	}
	R(&createOpt{}, cmdN("create"), "Create k8s cluster", func(s *mcclient.ClientSession, args *createOpt) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if args.Mode != "" {
			params.Add(jsonutils.NewString(args.Mode), "mode")
		}
		if args.K8sVersion != "" {
			params.Add(jsonutils.NewString(args.K8sVersion), "k8s_version")
		}
		if args.InfraImage != "" {
			params.Add(jsonutils.NewString(args.InfraImage), "infra_container_image")
		}
		if args.Cidr != "" {
			params.Add(jsonutils.NewString(args.Cidr), "cluster_cidr")
		}
		if args.Domain != "" {
			params.Add(jsonutils.NewString(args.Domain), "cluster_domain")
		}
		cluster, err := k8s.Clusters.Create(s, params)
		if err != nil {
			return err
		}
		printObject(cluster)
		return nil
	})

	type updateOpt struct {
		NAME       string `help:"Name of cluster"`
		K8sVersion string `help:"Cluster kubernetes components version" choices:"v1.8.10|v1.9.5|v1.10.0" default:"v1.9.5"`
	}
	R(&updateOpt{}, cmdN("update"), "Update k8s cluster", func(s *mcclient.ClientSession, args *updateOpt) error {
		params := jsonutils.NewDict()
		if args.K8sVersion != "" {
			params.Add(jsonutils.NewString(args.K8sVersion), "k8s_version")
		}
		cluster, err := k8s.Clusters.Update(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObject(cluster)
		return nil
	})

	type identOpt struct {
		ID string `help:"ID or name of the cluster"`
	}
	type deployOpt struct {
		identOpt
		Force bool `help:"Force deploy"`
	}
	R(&deployOpt{}, cmdN("deploy"), "Start deploy a cluster", func(s *mcclient.ClientSession, args *deployOpt) error {
		params := jsonutils.NewDict()
		if args.Force {
			params.Add(jsonutils.JSONTrue, "force")
		}
		ret, err := k8s.Clusters.PerformAction(s, args.ID, "deploy", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	type identsOpt struct {
		ID []string `help:"ID of clusters to operate"`
	}

	type getOpt struct {
		identOpt
	}
	R(&getOpt{}, cmdN("show"), "Show details of a cluster", func(s *mcclient.ClientSession, args *getOpt) error {
		result, err := k8s.Clusters.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type deleteOpt struct {
		identsOpt
	}
	R(&deleteOpt{}, cmdN("delete"), "Delete cluster", func(s *mcclient.ClientSession, args *deleteOpt) error {
		ret := k8s.Clusters.BatchDeleteWithParam(s, args.ID, nil, nil)
		printBatchResults(ret, k8s.Clusters.GetColumns(s))
		return nil
	})

	R(&getOpt{}, cmdN("kubeconfig"), "Generate kubeconfig of a cluster", func(s *mcclient.ClientSession, args *getOpt) error {
		ret, err := k8s.Clusters.PerformAction(s, args.ID, "generate-kubeconfig", nil)
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

	R(&getOpt{}, cmdN("engineconfig"), "Get kubernetes engine config of a cluster", func(s *mcclient.ClientSession, args *getOpt) error {
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

	R(&getOpt{}, cmdN("webhookauthurl"), "Get cluster kubernetes api server webhook auth url", func(s *mcclient.ClientSession, args *getOpt) error {
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
}

func initNode() {
	cmdN := func(suffix string) string {
		return fmt.Sprintf("k8s-node-%s", suffix)
	}
	type listOpt struct {
		BaseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s node", func(s *mcclient.ClientSession, args *listOpt) error {
		args.Details = true
		params := FetchPagingParams(args.BaseListOptions)
		result, err := k8s.Nodes.List(s, params)
		if err != nil {
			return err
		}
		printList(result, k8s.Nodes.GetColumns(s))
		return nil
	})

	type dockerConfig struct {
		RegistryMirrors    []string `json:"registry-mirrors"`
		InsecureRegistries []string `json:"insecure-registries"`
	}

	type createOpt struct {
		CLUSTER          string   `help:"Cluster id"`
		Etcd             bool     `help:"Etcd role"`
		Controlplane     bool     `help:"Controlplane role"`
		Worker           bool     `help:"Worker role"`
		AllRole          bool     `help:"All roles"`
		HostnameOverride string   `help:"Worker node overrided hostname"`
		Host             string   `help:"Yunion host server name or id"`
		Name             string   `help:"Name of node"`
		RegistryMirror   []string `help:"Docker registry mirrors, e.g. 'https://registry.docker-cn.com'"`
		InsecureRegistry []string `help:"Docker insecure registry"`
	}
	R(&createOpt{}, cmdN("create"), "Create k8s cluster node", func(s *mcclient.ClientSession, args *createOpt) error {
		params := jsonutils.NewDict()
		if args.Name != "" {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		params.Add(jsonutils.NewString(args.CLUSTER), "cluster")

		dockerConf := dockerConfig{}
		for _, rm := range args.RegistryMirror {
			dockerConf.RegistryMirrors = append(dockerConf.RegistryMirrors, rm)
		}
		for _, im := range args.InsecureRegistry {
			dockerConf.InsecureRegistries = append(dockerConf.InsecureRegistries, im)
		}
		confObj := jsonutils.Marshal(dockerConf)
		params.Add(confObj, "dockerd_config")

		roles := jsonutils.NewArray()
		if args.AllRole {
			roles.Add(jsonutils.NewString("etcd"), jsonutils.NewString("controlplane"), jsonutils.NewString("worker"))
		} else {
			if args.Etcd {
				roles.Add(jsonutils.NewString("etcd"))
			}
			if args.Controlplane {
				roles.Add(jsonutils.NewString("controlplane"))
			}
			if args.Worker {
				roles.Add(jsonutils.NewString("worker"))
			}
		}
		params.Add(roles, "roles")
		if args.HostnameOverride != "" {
			params.Add(jsonutils.NewString(args.HostnameOverride), "hostname_override")
		}
		if args.Host != "" {
			params.Add(jsonutils.NewString(args.Host), "host")
		}
		node, err := k8s.Nodes.Create(s, params)
		if err != nil {
			return err
		}
		printObject(node)
		return nil
	})

	type identOpt struct {
		ID string `help:"ID or name of the node"`
	}

	type identsOpt struct {
		ID []string `help:"ID or name of the nodes"`
	}
	type deleteOpt struct {
		identsOpt
	}
	R(&deleteOpt{}, cmdN("delete"), "Delete node", func(s *mcclient.ClientSession, args *deleteOpt) error {
		ret := k8s.Nodes.BatchDeleteWithParam(s, args.ID, nil, nil)
		printBatchResults(ret, k8s.Nodes.GetColumns(s))
		return nil
	})

	R(&identOpt{}, cmdN("show"), "Show node details", func(s *mcclient.ClientSession, args *identOpt) error {
		obj, err := k8s.Nodes.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(obj)
		return nil
	})

	R(&identOpt{}, cmdN("dockerconfig"), "Get docker daemon config", func(s *mcclient.ClientSession, args *identOpt) error {
		ret, err := k8s.Nodes.GetSpecific(s, args.ID, "docker-config", nil)
		if err != nil {
			return err
		}
		fmt.Println(ret)
		return nil
	})
}
