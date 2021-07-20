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

package jdmon

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	factory := SJdCloudReportFactory{}
	common.RegisterFactory(&factory)
}

type SJdCloudReportFactory struct {
}

func (self *SJdCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *common.ReportOptions, operatorType string) common.ICloudReport {
	return &SJdCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SJdCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_JDCLOUD
}

type SJdCloudReport struct {
	common.CloudReportBase
}

func (self *SJdCloudReport) Report() error {
	var servers []jsonutils.JSONObject
	var err error
	switch common.MonType(self.Operator) {
	case common.RDS:
		servers, err = self.GetAllserverOfThisProvider(&modules.DBInstance)
	default:
		servers, err = self.GetAllserverOfThisProvider(&modules.Servers)
	}
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
		case common.SERVER:
			err = self.collectRegionMetricOfServer(region, servers)
		case common.RDS:
			err = self.collectRegionMetricOfRds(region, servers)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
