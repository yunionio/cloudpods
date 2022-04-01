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

package bingomon

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	factory := SBingoCloudReportFactory{}
	common.RegisterFactory(&factory)
}

type SBingoCloudReportFactory struct {
	common.CommonReportFactory
}

func (self *SBingoCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *options.ReportOptions, operatorType string) common.ICloudReport {
	return &SBingoCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SBingoCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_BINGO_CLOUD
}

type SBingoCloudReport struct {
	common.CloudReportBase
}

func (self *SBingoCloudReport) Report() error {
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
		case common.SERVER:
			err = self.collectRegionMetricOfResource(region, servers)
		case common.HOST:
			err = self.collectRegionMetricOfResource(region, servers)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
