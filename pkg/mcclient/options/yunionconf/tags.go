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

package yunionconf

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type TagCreateOptions struct {
	options.BaseCreateOptions
	Values []string
}

func (o TagCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type TagUpdateOptions struct {
	options.BaseUpdateOptions
	Values []string
}

func (o TagUpdateOptions) Params() (jsonutils.JSONObject, error) {
	param, err := o.BaseUpdateOptions.Params()
	if err != nil {
		return nil, err
	}
	params := param.(*jsonutils.JSONDict)
	if len(o.Values) > 0 {
		params.Set("values", jsonutils.Marshal(o.Values))
	}
	return params, nil
}
