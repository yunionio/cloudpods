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

package alimon

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	factory := SAliCloudReportFactory{}
	common.RegisterFactory(&factory)
	cluster := &AliK8sClusterHelper{
		K8sClusterMetricBaseHelper: &common.K8sClusterMetricBaseHelper{
			ModuleHelper: map[common.K8sClusterModuleType]common.IK8sClusterModuleHelper{},
		},
	}
	cluster.RegisterModuleHelper(new(AliK8sClusterPodHelper))
	cluster.RegisterModuleHelper(new(AliK8sClusterNodeHelper))
	common.RegisterK8sClusterHelper(cluster)
}

type SAliCloudReportFactory struct {
	common.CommonReportFactory
}

func (self *SAliCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *options.ReportOptions, operatorType string) common.ICloudReport {
	return &SAliCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SAliCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_ALIYUN
}

type SAliCloudReport struct {
	common.CloudReportBase
}

func (self *SAliCloudReport) Report() error {
	var servers []jsonutils.JSONObject
	var err error

	servers, err = self.GetResourceByOperator()

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
		switch common.MonType(self.Operator) {
		case common.REDIS:
			err = self.collectRegionMetricOfRedis(region, servers)
		case common.K8S:
			err = self.collectRegionMetricOfResource(region, servers)
			if err != nil {
				return err
			}
			self.Impl = self
			err = self.CollectRegionMetricOfK8sModules(region, servers)
		default:
			err = self.collectRegionMetricOfResource(region, servers)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

type AliK8sClusterHelper struct {
	*common.K8sClusterMetricBaseHelper
}

func (a AliK8sClusterHelper) HelperBrand() string {
	return compute.CLOUD_PROVIDER_ALIYUN
}

type AliK8sClusterPodHelper struct {
	common.K8sClusterModuleQueryHelper
}

func (q AliK8sClusterPodHelper) MyModuleType() common.K8sClusterModuleType {
	return common.K8S_MODULE_POD
}

func (q AliK8sClusterPodHelper) MyResDimensionId() common.DimensionId {
	return common.DimensionId{
		LocalId: "name",
		ExtId:   "pod",
	}
}

func (q AliK8sClusterPodHelper) MyNamespaceAndMetrics() (string, map[string][]string) {
	return K8S_METRIC_NAMESPACE, aliK8SPodMetricSpecs
}

type AliK8sClusterNodeHelper struct {
	common.K8sClusterModuleQueryHelper
}

func (a AliK8sClusterNodeHelper) MyModuleType() common.K8sClusterModuleType {
	return common.K8S_MODULE_NODE
}

func (a AliK8sClusterNodeHelper) MyResDimensionId() common.DimensionId {
	return common.DimensionId{
		LocalId: "name",
		ExtId:   "node",
	}
}

func (a AliK8sClusterNodeHelper) MyNamespaceAndMetrics() (string, map[string][]string) {
	return K8S_METRIC_NAMESPACE, aliK8SNodeMetricSpecs
}
