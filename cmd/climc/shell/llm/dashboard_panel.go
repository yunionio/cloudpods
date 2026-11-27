// Copyright 2026 Yunion
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

package llm

import (
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	shell.R(&options.DashboardPanelGenerateOptions{}, "dashboard-panel-generate", "Generate custom dashboard panel JS via LLM (does not save)", dashboardPanelGenerate)
	shell.R(&options.DashboardPanelValidateOptions{}, "dashboard-panel-validate", "Validate custom dashboard panel JS", dashboardPanelValidate)
	shell.R(&options.DashboardPanelSaveOptions{}, "dashboard-panel-save", "Validate and save custom dashboard panel JS to yunionconf", dashboardPanelSave)
}

func dashboardPanelGenerate(s *mcclient.ClientSession, args *options.DashboardPanelGenerateOptions) error {
	body, err := args.Params()
	if err != nil {
		return err
	}
	ret, err := modules.DashboardPanel.Generate(s, body)
	if err != nil {
		return err
	}
	printutils.PrintJSONObject(ret)
	return nil
}

func dashboardPanelValidate(s *mcclient.ClientSession, args *options.DashboardPanelValidateOptions) error {
	body, err := args.Params()
	if err != nil {
		return err
	}
	ret, err := modules.DashboardPanel.Validate(s, body)
	if err != nil {
		return err
	}
	printutils.PrintJSONObject(ret)
	return nil
}

func dashboardPanelSave(s *mcclient.ClientSession, args *options.DashboardPanelSaveOptions) error {
	body, err := args.Params()
	if err != nil {
		return err
	}
	ret, err := modules.DashboardPanel.Save(s, body)
	if err != nil {
		return err
	}
	printutils.PrintJSONObject(ret)
	return nil
}
