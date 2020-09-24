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
	corev1 "k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
)

type FedNamespaceListOptions struct {
	FedResourceListOptions
}

type FedNamespaceCreateOptions struct {
	FedResourceCreateOptions
	Spec FedNamespaceSpec `json:"spec,allowempty"`
}

func (o *FedNamespaceCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type FedNamespaceSpec struct {
	Template NamespaceTemplate `json:"template,allowempty"`
}

type NamespaceTemplate struct {
	Spec corev1.NamespaceSpec `json:"spec,allowempty"`
}
