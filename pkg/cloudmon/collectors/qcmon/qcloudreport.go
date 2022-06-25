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

package qcmon

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	factory := SQCloudReportFactory{}
	common.RegisterFactory(&factory)

	cluster := &QcloudK8sClusterHelper{
		K8sClusterMetricBaseHelper: &common.K8sClusterMetricBaseHelper{
			ModuleHelper: map[common.K8sClusterModuleType]common.IK8sClusterModuleHelper{},
		},
	}
	cluster.RegisterModuleHelper(new(QcloudK8sClusterDeployHelper))
	//cluster.RegisterModuleHelper(new(QcloudK8sClusterContainerHelper))
	cluster.RegisterModuleHelper(new(QcloudK8sClusterPodHelper))
	cluster.RegisterModuleHelper(new(QcloudK8sClusterNodeHelper))

	common.RegisterK8sClusterHelper(cluster)

}

type SQCloudReportFactory struct {
	common.CommonReportFactory
}

func (self *SQCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *options.ReportOptions, operatorType string) common.ICloudReport {
	return &SQCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SQCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_QCLOUD
}

type SQCloudReport struct {
	common.CloudReportBase
}

func (self *SQCloudReport) Report() error {
	var servers []jsonutils.JSONObject
	var err error
	servers, err = self.GetResourceByOperator()
	if err != nil {
		return err
	}
	providerInstance, err := self.InitProviderInstance()
	if err != nil {
		return err
	}
	regionList, regionServerMap, err := self.GetAllRegionOfServers(servers, providerInstance)
	if err != nil {
		return err
	}
	for _, region := range regionList {
		servers := regionServerMap[region.GetGlobalId()]
		var err error
		switch common.MonType(self.Operator) {
		case common.K8S:
			err = self.collectRegionMetricOfK8S(region, servers)
			if err != nil {
				return err
			}
			self.Impl = self
			err = self.CollectRegionMetricOfK8sModules(region, servers)
		default:
			err = self.collectRegionMetricOfHost(region, servers)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

type QcloudK8sClusterHelper struct {
	*common.K8sClusterMetricBaseHelper
}

func (q QcloudK8sClusterHelper) HelperBrand() string {
	return compute.CLOUD_PROVIDER_QCLOUD
}

type QcloudK8sClusterDeployHelper struct {
	common.K8sClusterModuleQueryHelper
}

func (q QcloudK8sClusterDeployHelper) MyModuleType() common.K8sClusterModuleType {
	return common.K8S_MODULE_DEPLOY
}

func (q QcloudK8sClusterDeployHelper) MyResDimensionId() common.DimensionId {
	return common.DimensionId{
		LocalId: "name",
		ExtId:   "workload_name",
	}
}

func (q QcloudK8sClusterDeployHelper) MyNamespaceAndMetrics() (string, map[string][]string) {
	return K8S_METRIC_NAMESPACE, tecentK8SDeployMetricSpecs
}

type QcloudK8sClusterPodHelper struct {
	common.K8sClusterModuleQueryHelper
}

func (q QcloudK8sClusterPodHelper) MyNamespaceAndMetrics() (string, map[string][]string) {
	return K8S_METRIC_NAMESPACE, tecentK8SPodMetricSpecs
}

func (q QcloudK8sClusterPodHelper) MyModuleType() common.K8sClusterModuleType {
	return common.K8S_MODULE_POD
}

func (q QcloudK8sClusterPodHelper) MyResDimensionId() common.DimensionId {
	return common.DimensionId{
		LocalId: "nodeName,name",
		ExtId:   "node,pod_name",
	}
}

type QcloudK8sClusterContainerHelper struct {
	common.K8sClusterModuleQueryHelper
}

func (q QcloudK8sClusterContainerHelper) MyNamespaceAndMetrics() (string, map[string][]string) {
	return K8S_METRIC_NAMESPACE, tecentK8SContainerMetricSpecs
}

func (q QcloudK8sClusterContainerHelper) MyModuleType() common.K8sClusterModuleType {
	return common.K8S_MODULE_CONTAINER
}

func (q QcloudK8sClusterContainerHelper) MyResDimensionId() common.DimensionId {
	return common.DimensionId{
		LocalId: "labels.k8s-app,name",
		ExtId:   "workload_name,container_name",
	}
}

type QcloudK8sClusterNodeHelper struct {
	common.K8sClusterModuleQueryHelper
}

func (q QcloudK8sClusterNodeHelper) MyModuleType() common.K8sClusterModuleType {
	return common.K8S_MODULE_NODE
}

func (q QcloudK8sClusterNodeHelper) MyResDimensionId() common.DimensionId {
	return common.DimensionId{
		LocalId: "name",
		ExtId:   "node",
	}
}

func (q QcloudK8sClusterNodeHelper) MyNamespaceAndMetrics() (string, map[string][]string) {
	return K8S_METRIC_NAMESPACE, tecentK8SNodeMetricSpecs
}
