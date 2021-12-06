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

package common

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/request"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type K8sClusterModuleType string

const (
	K8S_MODULE_DEPLOY    = K8sClusterModuleType("deploy")
	K8S_MODULE_POD       = K8sClusterModuleType("pod")
	K8S_MODULE_CONTAINER = K8sClusterModuleType("container")
	K8S_MODULE_NS        = K8sClusterModuleType("ns")
	K8S_MODULE_NODE      = K8sClusterModuleType("node")
	K8S_MODULE_DAEMONSET = K8sClusterModuleType("daemonset")
)

var (
	cloudReportTable      map[string]ICloudReportFactory
	k8sClusterHelperTable map[string]IK8sClusterMetricHelper
)

func init() {
	cloudReportTable = make(map[string]ICloudReportFactory)

	k8sClusterHelperTable = make(map[string]IK8sClusterMetricHelper)
}

type IRoutineFactory interface {
	MyRoutineFunc() RoutineFunc
}

type ICloudReportFactory interface {
	NewCloudReport(provider *SProvider, session *mcclient.ClientSession, args *options.ReportOptions,
		operatorType string) ICloudReport
	GetId() string
	MyRoutineInteval(monOptions options.CloudMonOptions) time.Duration
}

type IK8sClusterMetricHelper interface {
	HelperBrand() string
	MyModuleHelper() map[K8sClusterModuleType]IK8sClusterModuleHelper
	RegisterModuleHelper(helper IK8sClusterModuleHelper)
}

type K8sClusterMetricBaseHelper struct {
	ModuleHelper map[K8sClusterModuleType]IK8sClusterModuleHelper
}

func (h *K8sClusterMetricBaseHelper) MyModuleHelper() map[K8sClusterModuleType]IK8sClusterModuleHelper {
	return h.ModuleHelper
}

func (h *K8sClusterMetricBaseHelper) RegisterModuleHelper(helper IK8sClusterModuleHelper) {
	h.ModuleHelper[helper.MyModuleType()] = helper
}

type IK8sClusterModuleHelper interface {
	MyModuleType() K8sClusterModuleType
	/**
	DimensionId.LocalId 由「,」分割表示多个组装后Dimension
	LocalId和ExtId 「,」分割长度需一致
	*/
	MyResDimensionId() DimensionId
	MyNamespaceAndMetrics() (string, map[string][]string)
	MyResourceFilterQuery(res jsonutils.JSONObject) jsonutils.JSONObject
}

type K8sClusterModuleQueryHelper struct {
}

func (q K8sClusterModuleQueryHelper) MyResourceFilterQuery(jsonutils.JSONObject) jsonutils.JSONObject {
	return nil
}

type DimensionId struct {
	LocalId string
	ExtId   string
}

type CommonReportFactory struct {
}

func (co *CommonReportFactory) MyRoutineInteval(monOptions options.CloudMonOptions) time.Duration {
	interval64, _ := strconv.ParseInt(monOptions.Interval, 10, 32)
	duration := time.Duration(interval64) * time.Minute
	return duration
}

type ICloudReport interface {
	Report() error
	GetAllserverOfThisProvider(manager modulebase.Manager, query *jsonutils.JSONDict) ([]jsonutils.JSONObject, error)
	InitProviderInstance() (cloudprovider.ICloudProvider, error)
	GetAllRegionOfServers(servers []jsonutils.JSONObject, providerInstance cloudprovider.ICloudProvider) (
		[]cloudprovider.ICloudRegion, map[string][]jsonutils.JSONObject, error)
	CollectRegionMetric(region cloudprovider.ICloudRegion,
		servers []jsonutils.JSONObject) error
}

type ICloudReportK8s interface {
	CollectRegionMetricOfK8sModules(region cloudprovider.ICloudRegion,
		clusters []jsonutils.JSONObject) error
	CollectK8sModuleMetric(region cloudprovider.ICloudRegion, cluster jsonutils.JSONObject,
		helper IK8sClusterModuleHelper) error
}

type SProvider struct {
	compute.CloudproviderDetails
	//Id        string `json:"id"
	//Name      string `json:"name"`
	//Provider  string `json:"provider"`
	//Account   string `json:"account"`
	//Secret    string `json:"secret"`
	//AccessUrl string `json:"access_url"`
}

func RegisterFactory(factory ICloudReportFactory) {
	cloudReportTable[factory.GetId()] = factory
}

func GetCloudReportFactory(provider string) (ICloudReportFactory, error) {
	factory, ok := cloudReportTable[provider]
	if ok {
		return factory, nil
	}
	log.Errorf("Provider %s not registerd", provider)
	return nil, fmt.Errorf("No such cloudReport %s", provider)
}

func (s *SProvider) Validate() error {
	invalidParams := request.ErrInvalidParams{Context: "SProvider"}
	if s.Id == "" {
		invalidParams.Add(request.NewErrParamRequired("Id"))
	}
	if s.Name == "" {
		invalidParams.Add(request.NewErrParamRequired("Name"))
	}
	if s.Account == "" {
		invalidParams.Add(request.NewErrParamRequired("Account"))
	}
	//if s.AccessUrl == "" {
	//	invalidParams.Add(request.NewErrParamRequired("AccountUrl"))
	//}
	if s.Provider == "" {
		invalidParams.Add(request.NewErrParamRequired("Provider"))
	}
	if s.Secret == "" {
		invalidParams.Add(request.NewErrParamRequired("Secret"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	}
	return nil
}

func RegisterK8sClusterHelper(helper IK8sClusterMetricHelper) {
	k8sClusterHelperTable[helper.HelperBrand()] = helper
}

func GetK8sClusterHelper(brand string) (IK8sClusterMetricHelper, error) {
	helper, ok := k8sClusterHelperTable[brand]
	if ok {
		return helper, nil
	}
	log.Errorf("brand %s not registerd", brand)
	return nil, fmt.Errorf("No such K8sClusterMetricHelper %s", brand)
}
