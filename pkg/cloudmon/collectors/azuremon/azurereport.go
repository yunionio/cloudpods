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

package azuremon

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	factory := SAzureCloudReportFactory{}
	common.RegisterFactory(&factory)

	cluster := &AzureK8sClusterHelper{
		K8sClusterMetricBaseHelper: &common.K8sClusterMetricBaseHelper{
			ModuleHelper: map[common.K8sClusterModuleType]common.IK8sClusterModuleHelper{},
		},
	}
	cluster.RegisterModuleHelper(new(AzureK8sClusterPodHelper))
	cluster.RegisterModuleHelper(new(AzureK8sClusterNodeHelper))
	common.RegisterK8sClusterHelper(cluster)
}

type SAzureCloudReportFactory struct {
	common.CommonReportFactory
}

func (self *SAzureCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *options.ReportOptions, operatorType string) common.ICloudReport {
	return &SAzureCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SAzureCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_AZURE
}

type SAzureCloudReport struct {
	common.CloudReportBase
}

func (self *SAzureCloudReport) Report() error {
	servers, err := self.GetResourceByOperator()
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
		switch common.MonType(self.Operator) {
		case common.K8S:
			self.Impl = self
			err = self.CollectRegionMetricOfK8sModules(region, servers)
		default:
			err = common.CollectRegionMetricAsync(self.Args.Batch, region, servers, self)

		}
		if err != nil {
			return err
		}
	}
	return nil
}

type AzureK8sClusterHelper struct {
	*common.K8sClusterMetricBaseHelper
}

func (a AzureK8sClusterHelper) HelperBrand() string {
	return compute.CLOUD_PROVIDER_AZURE
}

type ik8sModuleFilterHelper interface {
	filter(object jsonutils.JSONObject) string
}

type AzureK8sClusterPodHelper struct {
	common.K8sClusterModuleQueryHelper
}

func (a AzureK8sClusterPodHelper) filter(resource jsonutils.JSONObject) string {
	parentName, _ := resource.GetString("name")
	return fmt.Sprintf(`controllerName eq '%s'`, parentName)
}

func (a AzureK8sClusterPodHelper) MyModuleType() common.K8sClusterModuleType {
	return common.K8S_MODULE_POD
}

func (a AzureK8sClusterPodHelper) MyResDimensionId() common.DimensionId {
	return common.DimensionId{
		LocalId: "",
		ExtId:   "",
	}
}

func (a AzureK8sClusterPodHelper) MyNamespaceAndMetrics() (string, map[string][]string) {
	return K8S_POD_METRIC_NAMESPACE, azureK8SPodMetricSpecs
}

func (q AzureK8sClusterPodHelper) MyResourceFilterQuery(resource jsonutils.JSONObject) jsonutils.JSONObject {
	query := jsonutils.NewDict()
	parentName, _ := resource.GetString("name")
	kind, _ := resource.GetString("kind")
	namespace, _ := resource.GetString("namespace")
	query.Set("owner_name", jsonutils.NewString(parentName))
	query.Set("owner_kind", jsonutils.NewString(kind))
	query.Set("namespace", jsonutils.NewString(namespace))
	return query
}

type AzureK8sClusterNodeHelper struct {
	common.K8sClusterModuleQueryHelper
}

func (a AzureK8sClusterNodeHelper) filter(resource jsonutils.JSONObject) string {
	parentName, _ := resource.GetString("name")
	return fmt.Sprintf(`node eq '%s'`, parentName)
}

func (a AzureK8sClusterNodeHelper) MyModuleType() common.K8sClusterModuleType {
	return common.K8S_MODULE_NODE
}

func (a AzureK8sClusterNodeHelper) MyResDimensionId() common.DimensionId {
	return common.DimensionId{
		LocalId: "",
		ExtId:   "",
	}
}

func (a AzureK8sClusterNodeHelper) MyNamespaceAndMetrics() (string, map[string][]string) {
	return K8S_NODE_METRIC_NAMESPACE, azureK8SNodeMetricSpecs
}
