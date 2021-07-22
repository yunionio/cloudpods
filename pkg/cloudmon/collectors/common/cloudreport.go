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

	"github.com/aws/aws-sdk-go/aws/request"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

func init() {
	cloudReportTable = make(map[string]ICloudReportFactory)
}

type ICloudReportFactory interface {
	NewCloudReport(provider *SProvider, session *mcclient.ClientSession, args *ReportOptions,
		operatorType string) ICloudReport
	GetId() string
}

type ICloudReport interface {
	Report() error
	GetAllserverOfThisProvider(manager modulebase.Manager) ([]jsonutils.JSONObject, error)
	InitProviderInstance() (cloudprovider.ICloudProvider, error)
	GetAllRegionOfServers(servers []jsonutils.JSONObject, providerInstance cloudprovider.ICloudProvider) (
		[]cloudprovider.ICloudRegion, map[string][]jsonutils.JSONObject, error)
	CollectRegionMetric(region cloudprovider.ICloudRegion,
		servers []jsonutils.JSONObject) error
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

var cloudReportTable map[string]ICloudReportFactory

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
