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

package options

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type SharableProjectizedResourceBaseCreateInput struct {
	apis.ProjectizedResourceCreateInput
	apis.SharableResourceBaseCreateInput
}

func (opts *SharableProjectizedResourceBaseCreateInput) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts.SharableResourceBaseCreateInput)
	if err != nil {
		return nil, err
	}

	projectInput, err := optionsStructToParams(opts.ProjectizedResourceCreateInput.ProjectizedResourceInput)
	if err != nil {
		return nil, err
	}

	domainInput, err := optionsStructToParams(opts.ProjectizedResourceCreateInput.DomainizedResourceInput)
	if err != nil {
		return nil, err
	}

	params.Update(projectInput)
	params.Update(domainInput)
	return params, nil
}

type SharableResourcePublicBaseOptions struct {
	Scope          string   `help:"sharing scope" choices:"system|domain|project"`
	SharedProjects []string `help:"Share to projects"`
	SharedDomains  []string `help:"Share to domains"`
}
