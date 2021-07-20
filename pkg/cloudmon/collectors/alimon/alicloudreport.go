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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	factory := SAliCloudReportFactory{}
	common.RegisterFactory(&factory)
}

type SAliCloudReportFactory struct {
}

func (self *SAliCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *common.ReportOptions, operatorType string) common.ICloudReport {
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
	switch self.Operator {
	case "redis":
		servers, err = self.GetAllserverOfThisProvider(&modules.ElasticCache)
	case "rds":
		servers, err = self.GetAllserverOfThisProvider(&modules.DBInstance)
	case "oss":
		servers, err = self.GetAllserverOfThisProvider(&modules.Buckets)
	case "elb":
		servers, err = self.GetAllserverOfThisProvider(&modules.Loadbalancers)
	default:
		servers, err = self.GetAllserverOfThisProvider(&modules.Servers)
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
		switch self.Operator {
		case "server":
			err = self.collectRegionMetricOfHost(region, servers)
		case "redis":
			err = self.collectRegionMetricOfRedis(region, servers)
		case "rds":
			err = self.collectRegionMetricOfRds(region, servers)
		case "oss":
			err = self.collectRegionMetricOfOss(region, servers)
		case "elb":
			err = self.collectRegionMetricOfElb(region, servers)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
