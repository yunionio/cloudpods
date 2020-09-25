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

	"yunion.io/x/onecloud/pkg/apis"
)

type ClusterResourceCreateOptions struct {
	CLUSTER string `default:"$K8S_CLUSTER" help:"Kubernetes cluster name"`
	NAME    string `help:"Name of resource"`
}

func (o ClusterResourceCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.CLUSTER), "cluster_id")
	params.Add(jsonutils.NewString(o.NAME), "name")
	return params, nil
}

type NamespaceResourceCreateOptions struct {
	apis.DomainLevelResourceCreateInput
	CLUSTER   string `default:"$K8S_CLUSTER" help:"Kubernetes cluster name"`
	NAMESPACE string `help:"Namespace of resource"`
}

func (o NamespaceResourceCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o.DomainLevelResourceCreateInput).(*jsonutils.JSONDict)
	params.Add(jsonutils.NewString(o.CLUSTER), "cluster_id")
	params.Add(jsonutils.NewString(o.NAMESPACE), "namespace_id")
	return params, nil
}

type ClusterResourceUpdateOptions struct {
	ID string `help:"Id of resource"`
}

func (o ClusterResourceUpdateOptions) GetId() string {
	return o.ID
}

type NamespaceResourceUpdateOptions struct {
	ClusterResourceUpdateOptions
}
