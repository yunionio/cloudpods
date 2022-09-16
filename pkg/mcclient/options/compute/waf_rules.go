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

package compute

import (
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type WafRuleListOptions struct {
	options.BaseListOptions

	WafInstanceId  string
	WafRuleGroupId string
}

func (opts *WafRuleListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type WafRuleOptions struct {
	RULE_FILE string
}

func (opts *WafRuleOptions) Params() (jsonutils.JSONObject, error) {
	data, err := os.ReadFile(opts.RULE_FILE)
	if err != nil {
		return nil, errors.Wrapf(err, "ioutils.ReadFile")
	}
	ret, err := jsonutils.Parse(data)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type WafRuleUpdateOptions struct {
	options.BaseIdOptions
	WafRuleOptions
}

func (opts *WafRuleUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return opts.WafRuleOptions.Params()
}
