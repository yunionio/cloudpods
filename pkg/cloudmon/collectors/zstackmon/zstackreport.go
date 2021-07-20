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

package zstackmon

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	factory := SZStackCloudReportFactory{}
	common.RegisterFactory(&factory)
}

type SZStackCloudReportFactory struct {
}

func (self *SZStackCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *common.ReportOptions, operatorType string) common.ICloudReport {
	return &SZStackCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SZStackCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_ZSTACK
}

type SZStackCloudReport struct {
	common.CloudReportBase
}

func (self *SZStackCloudReport) Report() error {
	switch self.Operator {
	case string(common.SERVER):
		return self.getServerMetrics()
	case string(common.HOST):
		return self.getHoseMetrics()
	}
	return nil
}

func (self *SZStackCloudReport) getServerMetrics() error {
	servers, err := self.GetAllserverOfThisProvider(&modules.Servers)
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
		err = common.CollectRegionMetricAsync(self.Args.Batch, region, regionServerMap[region.GetGlobalId()], self)
		if err != nil {
			log.Errorln(err)
			continue
		}
	}
	return nil
}

func (self *SZStackCloudReport) getHoseMetrics() error {
	hosts, err := self.GetAllHostOfThisProvider(&modules.Hosts)
	if err != nil {
		return err
	}
	providerInstance, err := self.InitProviderInstance()
	if err != nil {
		return err
	}
	regionList, regionServerMap, err := self.GetAllRegionOfServers(hosts, providerInstance)
	if err != nil {
		return err
	}
	for _, region := range regionList {
		err = common.CollectRegionMetricAsync(self.Args.Batch, region, regionServerMap[region.GetGlobalId()], self)
		if err != nil {
			log.Errorln(err)
			continue
		}
	}
	return nil
}
