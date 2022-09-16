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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ProjectMappingListOptions struct {
	options.BaseListOptions
}

func (opts *ProjectMappingListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ProjectMappingCreateOption struct {
	options.BaseCreateOptions
	RULES_FILE string
}

func (opts *ProjectMappingCreateOption) Params() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	ret.Update(jsonutils.Marshal(opts.BaseCreateOptions))
	data, err := os.ReadFile(opts.RULES_FILE)
	if err != nil {
		return nil, err
	}
	rules, err := jsonutils.Parse(data)
	if err != nil {
		return nil, err
	}
	ret.Add(rules, "rules")
	return ret, nil
}

type ProjectMappingUpdateOption struct {
	options.BaseUpdateOptions
	RulesFile string
}

func (opts *ProjectMappingUpdateOption) Params() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	ret.Update(jsonutils.Marshal(opts.BaseUpdateOptions))
	if len(opts.RulesFile) > 0 {
		data, err := os.ReadFile(opts.RulesFile)
		if err != nil {
			return nil, err
		}
		rules, err := jsonutils.Parse(data)
		if err != nil {
			return nil, err
		}
		ret.Add(rules, "rules")
	}
	return ret, nil
}
