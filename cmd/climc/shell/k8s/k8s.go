package k8s

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/util/printutils"
)

func init() {
	initCluster()
	initNode()
	initPod()
}

type BaseListOptions shell.BaseListOptions

type clusterBaseOptions struct {
	Cluster string `default:"$K8S_CLUSTER" help:"Kubernetes cluster name"`
}

func (o clusterBaseOptions) ClusterContext() []modules.ManagerContext {
	return []modules.ManagerContext{clusterContext(o.Cluster)}
}

type k8sBaseListOptions struct {
	clusterBaseOptions
	Limit int `default:"20" help:"Page limit"`
}

var (
	R                 = shell.R
	printList         = printutils.PrintJSONList
	printObject       = printutils.PrintJSONObject
	printBatchResults = printutils.PrintJSONBatchResults
)

func FetchPagingParams(o BaseListOptions) jsonutils.JSONObject {
	return shell.FetchPagingParams(shell.BaseListOptions(o))
}

func resourceCmdN(prefix, suffix string) string {
	return fmt.Sprintf("k8s-%s-%s", prefix, suffix)
}

func clusterContext(clusterId string) modules.ManagerContext {
	return modules.ManagerContext{
		InstanceManager: k8s.Clusters,
		InstanceId:      clusterId,
	}
}
