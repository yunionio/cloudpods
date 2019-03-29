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

type TillerCreateOptions struct {
	ClusterBaseOptions
	KubeContext string `json:"kube_context"`
	Namespace   string `json:"namespace" default:"kube-system"`
	// Upgrade if Tiller is already installed
	Upgrade bool `json:"upgrade"`
	// Name of service account
	ServiceAccount string `json:"service_account" default:"tiller"`
	// Use the canary Tiller image
	Canary bool `json:"canary_image"`

	// Override Tiller image
	Image string `json:"tiller_image" default:"yunion/tiller:v2.9.1"`
	// Limit the maximum number of revisions saved per release. Use 0 for no limit.
	MaxHistory int `json:"history_max"`
}

func (o TillerCreateOptions) Params() *jsonutils.JSONDict {
	params := o.ClusterBaseOptions.Params()
	if len(o.KubeContext) > 0 {
		params.Add(jsonutils.NewString(o.KubeContext), "kube_context")
	}
	params.Add(jsonutils.NewString(o.Namespace), "namespace")
	params.Add(jsonutils.NewString(o.ServiceAccount), "service_account")
	if o.Canary {
		params.Add(jsonutils.JSONTrue, "canary_image")
	}
	if o.Upgrade {
		params.Add(jsonutils.JSONTrue, "upgrade")
	}
	if len(o.Image) > 0 {
		params.Add(jsonutils.NewString(o.Image), "tiller_image")
	}
	if o.MaxHistory > 0 {
		params.Add(jsonutils.NewInt(int64(o.MaxHistory)), "history_max")
	}
	return params
}
