package cloudaccountmon

import (
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	factory := SCloudAccountFactory{}
	common.RegisterFactory(&factory)
}

type SCloudAccountFactory struct {
}

func (self *SCloudAccountFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *common.ReportOptions,
	operatorType string) common.ICloudReport {
	return &SCloudAccountReport{
		common.CloudReportBase{
			SProvider: nil,
			Session:   session,
			Args:      args,
			Operator:  ClOUDACCOUNT_ID,
		},
	}
}

func (S SCloudAccountFactory) GetId() string {
	return ClOUDACCOUNT_ID
}

type SCloudAccountReport struct {
	common.CloudReportBase
}

func (self *SCloudAccountReport) Report() error {
	accounts, err := self.GetAllCloudAccount(&modules.Cloudaccounts)
	if err != nil {
		return err
	}
	return self.collectMetric(accounts)
}
