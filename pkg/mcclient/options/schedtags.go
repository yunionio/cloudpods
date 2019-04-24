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
	"fmt"

	"yunion.io/x/jsonutils"
)

type SchedtagModelListOptions struct {
	BaseListOptions
	Schedtag string `help:"ID or Name of schedtag"`
}

func (o SchedtagModelListOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	return params, nil
}

type SchedtagModelPairOptions struct {
	SCHEDTAG string `help:"Scheduler tag"`
	OBJECT   string `help:"Object id"`
}

type SchedtagSetOptions struct {
	ID       string   `help:"Id or name of resource"`
	Schedtag []string `help:"Ids of schedtag"`
}

func (o SchedtagSetOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	for idx, tag := range o.Schedtag {
		params.Add(jsonutils.NewString(tag), fmt.Sprintf("schedtag.%d", idx))
	}
	return params, nil
}
