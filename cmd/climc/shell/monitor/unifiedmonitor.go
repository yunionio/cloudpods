package monitor

import (
	"fmt"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(monitor.UnifiedMonitorManager).WithKeyword("simple-query")
	cmd.GetWithCustomShow("simplequery", func(result jsonutils.JSONObject) {
		rr := make(map[string]string)
		err := result.Unmarshal(&rr)
		if err != nil {
			fmt.Println(err)
			return
		}
		// 输出结果
		for _, v := range rr {
			fmt.Printf("%s\n", v)
		}
	}, &options.SimpleQueryTest{})
}
