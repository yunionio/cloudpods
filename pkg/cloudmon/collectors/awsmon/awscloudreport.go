package awsmon

import (
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	factory := SAwsCloudReportFactory{}
	common.RegisterFactory(&factory)
}

type SAwsCloudReportFactory struct {
}

func (self *SAwsCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *common.ReportOptions, operatorType string) common.ICloudReport {
	return &SAwsCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SAwsCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_AWS
}

type SAwsCloudReport struct {
	common.CloudReportBase
}

func (self *SAwsCloudReport) Report() error {
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
			err = common.CollectRegionMetricAsync(self.Args.Batch, region, servers, self)
			//err = self.collectRegionMetricOfHost(region, servers)
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
