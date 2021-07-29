package collectors

import (
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&common.ReportOptions{}, "report-storage", "Report Storage", reportStorage)
}

func reportStorage(session *mcclient.ClientSession, args *common.ReportOptions) error {
	return common.ReportCustomizeCloudMetric(string(common.STORAGE), session, args)
}
