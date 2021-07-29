package collectors

import (
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&common.ReportOptions{}, "report-oss", "Report Oss", reportOss)
}
func reportOss(session *mcclient.ClientSession, args *common.ReportOptions) error {
	return common.ReportCloudMetricOfoperatorType(string(common.OSS), session, args)
}
