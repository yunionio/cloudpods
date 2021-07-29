package apsaramon

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	factory := SApsaraCloudReportFactory{}
	common.RegisterFactory(&factory)
}

type SApsaraCloudReportFactory struct {
}

func (self *SApsaraCloudReportFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *common.ReportOptions, operatorType string) common.ICloudReport {
	return &SApsaraCloudReport{
		common.CloudReportBase{
			SProvider: provider,
			Session:   session,
			Args:      args,
			Operator:  operatorType,
		},
	}
}

func (self *SApsaraCloudReportFactory) GetId() string {
	return compute.CLOUD_PROVIDER_APSARA
}

type SApsaraCloudReport struct {
	common.CloudReportBase
}

func (self *SApsaraCloudReport) Report() error {
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
