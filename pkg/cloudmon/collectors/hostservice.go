package collectors

import (
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&common.ReportOptions{}, "report-host", "Report Host", reportHost)
}

//入口函数[aliyun、huawei]
func reportHost(session *mcclient.ClientSession, args *common.ReportOptions) error {
	return common.ReportCloudMetricOfoperatorType(string(common.HOST), session, args)
}
