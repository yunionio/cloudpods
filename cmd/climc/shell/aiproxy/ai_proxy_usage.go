// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aiproxy

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
	apoptions "yunion.io/x/onecloud/pkg/mcclient/options/aiproxy"
)

func init() {
	cmd := shell.NewResourceCmd(&apmodules.AIProxyUsage)
	cmd.ClassShow(new(apoptions.AiProxyUsageOverviewOptions))
	cmd.ClassShow(new(apoptions.AiProxyUsageAnalysisOptions))
	cmd.RunWithDesc("event-list", "List ai_proxy_usage events", new(apoptions.AiProxyUsageEventListOptions),
		func(s *mcclient.ClientSession, args *apoptions.AiProxyUsageEventListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := apmodules.AIProxyUsage.Get(s, args.GetId(), params)
			if err != nil {
				return err
			}
			shell.PrintList(modulebase.JSON2ListResult(ret), []string{
				"id",
				"timestamp",
				"model",
				"provider",
				"source",
				"api_key_label",
				"input_tokens",
				"output_tokens",
				"total_tokens",
				"latency_ms",
				"result",
				"status_code",
			})
			return nil
		})
}
