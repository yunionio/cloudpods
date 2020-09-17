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

package k8s

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type ConfigMapCreateOptions struct {
	NamespaceResourceCreateOptions
	DataKey   []string `help:"configmap data key"`
	DataValue []string `help:"configmap data value"`
}

func (o *ConfigMapCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := o.NamespaceResourceCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	if len(o.DataKey) != len(o.DataValue) {
		return nil, errors.Error("data key and value must paired")
	}
	data := jsonutils.NewDict()
	for i := range o.DataKey {
		key := o.DataKey[i]
		val := o.DataValue[i]
		data.Add(jsonutils.NewString(val), key)
	}
	params.(*jsonutils.JSONDict).Add(data, "data")
	return params, nil
}
