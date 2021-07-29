package collectors

import (
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&common.ReportOptions{}, "report-alertrecord", "Report alertrecord history", reportAlertrecord)
}

func reportAlertrecord(session *mcclient.ClientSession, args *common.ReportOptions) error {
	return common.ReportCustomizeCloudMetric(string(common.ALERT_RECORD), session, args)
}
