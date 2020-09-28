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
)

type PVCListOptions struct {
	NamespaceResourceListOptions
	Unused bool `help:"Filter unused pvc"`
}

func (o PVCListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := o.NamespaceResourceListOptions.Params()
	if err != nil {
		return nil, err
	}
	if o.Unused {
		params.(*jsonutils.JSONDict).Add(jsonutils.JSONTrue, "unused")
	}
	return params, nil
}

type PVCCreateOptions struct {
	NamespaceWithClusterOptions
	NAME         string `help:"Name of PVC"`
	SIZE         string `help:"Storage size, e.g. 10Gi"`
	StorageClass string `help:"PVC StorageClassName"`
}

func (o PVCCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(o.SIZE), "size")
	if o.StorageClass != "" {
		params.Add(jsonutils.NewString(o.StorageClass), "storageClass")
	}
	return params, nil
}
