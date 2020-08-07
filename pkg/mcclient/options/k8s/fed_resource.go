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
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type FedResourceListOptions struct {
	options.BaseListOptions
}

func (o *FedResourceListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type FedNamespaceResourceListOptions struct {
	FedResourceListOptions
	Federatednamespace string `json:"federatednamespace"`
}

func (o *FedNamespaceResourceListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type FedResourceCreateOptions struct {
	apis.DomainLevelResourceCreateInput
}

type FedResourceClusterJointOptions struct {
	ClusterId string `json:"cluster_id" positional:"true" required:"true" help:"ID or name of cluster"`
}

func (o FedResourceClusterJointOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type FedResourceClusterShowOptions struct {
	CLUSTER string `help:"ID or Name of cluster"`
}

func (o FedResourceClusterShowOptions) GetSlaveId() string {
	return o.CLUSTER
}

type FedNamespaceResourceCreateOptions struct {
	FEDNAMESPACE string `help:"Federatednamespace id or name"`
	FedResourceCreateOptions
}

func (o FedNamespaceResourceCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o.FedResourceCreateOptions)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString(o.FEDNAMESPACE), "federatednamespace_id")
	return params, nil
}

func (o FedNamespaceResourceCreateOptions) ToInput() FedNamespaceResourceCreateInput {
	return FedNamespaceResourceCreateInput{
		FedResourceCreateOptions: o.FedResourceCreateOptions,
		FederatednamespaceId:     o.FEDNAMESPACE,
	}
}

type FedNamespaceResourceCreateInput struct {
	FedResourceCreateOptions
	FederatednamespaceId string `json:"federatednamespace_id"`
}
