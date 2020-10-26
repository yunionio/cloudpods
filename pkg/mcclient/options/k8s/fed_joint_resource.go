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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type FedJointClusterBaseListOptions struct {
	options.BaseListOptions
	FederatedResourceId string `help:"ID or Name of federated resource" json:"federatedresource_id"`
	ClusterId           string `help:"ID or Name of cluster"`
	NamespaceId         string `help:"ID or Name of namespace"`
	ResourceId          string `help:"ID or Name of resource"`
}

func (o *FedJointClusterBaseListOptions) GetMasterOpt() string {
	return o.FederatedResourceId
}

func (o *FedJointClusterBaseListOptions) GetSlaveOpt() string {
	return o.ClusterId
}

func (o *FedJointClusterBaseListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type FedNamespaceJointClusterListOpt struct {
	FedJointClusterBaseListOptions
	FederatednamespaceId string `help:"ID or Name of federatednamespace"`
}

func (o *FedNamespaceJointClusterListOpt) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type FedResourceJointClusterShowOptions struct {
	FedResourceIdOptions
	FedResourceClusterShowOptions
}

func (o FedResourceJointClusterShowOptions) GetMasterId() string {
	return o.FEDRESOURCE
}

func (o FedResourceJointClusterShowOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type FedResourceIdOptions struct {
	FEDRESOURCE string `help:"ID or Name of federated resource"`
}

func (o FedResourceIdOptions) GetId() string {
	return o.FEDRESOURCE
}

func (o FedResourceIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type FedResourceUpdateOptions struct {
	FedResourceIdOptions
}

func (o FedResourceUpdateOptions) GetUpdateFields() []string {
	return []string{"spec"}
}

type FedResourceJointClusterAttachOptions struct {
	FedResourceIdOptions
	FedResourceClusterJointOptions
}

func (o *FedResourceJointClusterAttachOptions) GetId() string {
	return o.FEDRESOURCE
}

func (o *FedResourceJointClusterAttachOptions) Params() (jsonutils.JSONObject, error) {
	return o.FedResourceClusterJointOptions.Params()
}

type FedResourceJointClusterDetachOptions struct {
	FedResourceIdOptions
	FedResourceClusterJointOptions
}

func (o *FedResourceJointClusterDetachOptions) GetId() string {
	return o.FEDRESOURCE
}

func (o *FedResourceJointClusterDetachOptions) Params() (jsonutils.JSONObject, error) {
	return o.FedResourceClusterJointOptions.Params()
}
