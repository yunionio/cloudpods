package collectors

import (
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&options.ReportOptions{}, "report-k8s", "Report k8s", reporK8s)
}

func reporK8s(session *mcclient.ClientSession, args *options.ReportOptions) error {
	return common.ReportCloudMetricOfoperatorType(string(common.K8S), session, args)
}
