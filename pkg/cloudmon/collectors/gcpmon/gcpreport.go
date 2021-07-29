package gcpmon

import (
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	factory := SGoogleCloudReportFactory{}
	common.RegisterFactory(&factory)
}

type SGoogleCloudReportFactory struct {
}

func (self *SGoogleCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *common.ReportOptions, operatorType string) common.ICloudReport {
	return &SGoogleCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SGoogleCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_GOOGLE
}

type SGoogleCloudReport struct {
	common.CloudReportBase
}

func (self *SGoogleCloudReport) Report() error {
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
		servers := regionServerMap[region.GetGlobalId()]
		switch self.Operator {
		case "server":
			err = self.collectRegionMetricOfHost(region, servers)
			//case "redis":
			//	err = self.collectRegionMetricOfRedis(region, servers)
			//case "rds":
			//	err = self.collectRegionMetricOfRds(region, servers)
			//case "oss":
			//	err = self.collectRegionMetricOfOss(region, servers)
			//case "elb":
			//	err = self.collectRegionMetricOfElb(region, servers)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
