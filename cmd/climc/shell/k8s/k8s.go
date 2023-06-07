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
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func init() {
	// cluster resources
	initKubeCluster()
	initKubeMachine()
	initKubeCerts()

	// helm resources
	initTiller()
	initRepo()
	initChart()
	initRelease()

	// kubernetes original resources
	initRaw()
	initConfigMap()
	initDeployment()
	initStatefulset()
	initDaemonSet()
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
	initEvent()
	initRbac()

	initApp()

	// container registry
	initContainerRegistry()
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

func clusterContext(clusterId string) modulebase.ManagerContext {
	return modulebase.ManagerContext{
		InstanceManager: k8s.KubeClusters,
		InstanceId:      clusterId,
	}
}

func printObjectYAML(obj jsonutils.JSONObject) {
	fmt.Println(obj.YAMLString())
}

func initK8sClusterResource(kind string, manager modulebase.Manager) *K8sResourceCmd {
	cmd := NewK8sResourceCmd(manager)
	cmd.SetKeyword(kind)
	cmd.List(new(o.ResourceListOptions))
	cmd.Show(new(o.ResourceGetOptions))
	cmd.BatchDeleteWithParam(new(o.ResourceDeleteOptions))
	cmd.ShowEvent()
	return cmd
}

func initK8sNamespaceResource(kind string, manager k8s.IClusterResourceManager) *K8sResourceCmd {
	cmd := NewK8sResourceCmd(manager)
	cmd.SetKeyword(kind)
	cmd.List(new(o.NamespaceResourceListOptions))
	cmd.Show(new(o.NamespaceResourceGetOptions))
	cmd.BatchDeleteWithParam(new(o.NamespaceResourceDeleteOptions))
	cmd.ShowRaw(new(o.NamespaceResourceGetOptions))
	cmd.EditRaw(new(o.NamespaceResourceGetOptions))
	cmd.ShowEvent()
	return cmd
}
