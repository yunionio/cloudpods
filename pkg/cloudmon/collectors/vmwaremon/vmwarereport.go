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

package vmwaremon

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	factory := SEsxiCloudReportFactory{}
	common.RegisterFactory(&factory)
}

type SEsxiCloudReportFactory struct {
}

func (self *SEsxiCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *common.ReportOptions, operatorType string) common.ICloudReport {
	return &SEsxiCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SEsxiCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_VMWARE
}

type SEsxiCloudReport struct {
	common.CloudReportBase
}

func (self *SEsxiCloudReport) Report() error {
	var err error
	switch self.Operator {
	case string(common.SERVER):
		//servers, err := self.getAllserverOfThisProvider(&modules.Servers)
		servers, err := self.GetAllserverOfThisProvider(&modules.Servers)
		if err != nil {
			return err
		}
		err = self.CollectServerMetricByProvider(servers)
	case string(common.HOST):
		hosts, err := self.GetAllHostOfThisProvider(&modules.Hosts)
		if err != nil {
			return err
		}
		//err = common.CollectRegionMetricAsync(40, nil, hosts, self)
		err = self.CollectRegionHostMetricAsync(hosts)
	}
	if err != nil {
		return err
	}
	return nil
}

func (self *SEsxiCloudReport) CollectServerMetricByProvider(servers []jsonutils.JSONObject) error {
	return self.collectRegionMetricOfServerBatch("", servers)

}

func (self *SEsxiCloudReport) CollectRegionHostMetricAsync(servers []jsonutils.JSONObject) error {
	log.Errorf("cloudproviderid: %s,%s count:%d", self.SProvider.Id, self.getMonType(), len(servers))
	if self.Args.Batch == 0 {
		self.Args.Batch = 100
	}
	for i := 0; i < len(servers); i += self.Args.Batch {
		tmp := i + self.Args.Batch
		if tmp > len(servers) {
			tmp = len(servers)
		}
		err := self.collectRegionMetricOfServerBatch("", servers[i:tmp])
		if err != nil {
			log.Errorln(err)
		}
	}
	return nil
}
