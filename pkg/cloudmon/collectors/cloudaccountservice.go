package collectors

import (
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&common.ReportOptions{}, "report-cloudaccount", "Report cloud account", reportCloudAccount)
}

func reportCloudAccount(session *mcclient.ClientSession, args *common.ReportOptions) error {
	return common.ReportCustomizeCloudMetric(string(common.CLOUDACCOUNT), session, args)
}
