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

type FedClusterRoleBindingCreatOpt struct {
	FedResourceCreateOptions
	RoleRef RoleRef `json:"roleRef"`
	Subject Subject `help:"Subject is role bind subject, e.g: User=jane"`
}

func (o *FedClusterRoleBindingCreatOpt) Params() (jsonutils.JSONObject, error) {
	input, err := o.ToInput()
	if err != nil {
		return nil, err
	}
	return input.JSON(input), nil
}

func (o *FedClusterRoleBindingCreatOpt) ToInput() (*FedClusterRoleBindingCreateInput, error) {
	input := &FedClusterRoleBindingCreateInput{
		FedResourceCreateOptions: o.FedResourceCreateOptions,
	}
	if err := validateRoleRef(&o.RoleRef); err != nil {
		return nil, err
	}
	if err := validateSubject(&o.Subject); err != nil {
		return nil, err
	}
	input.Spec.Template.RoleRef = o.RoleRef
	subs := []Subject{o.Subject}
	input.Spec.Template.Subjects = subs
	return input, nil
}

type FedClusterRoleBindingCreateInput struct {
	FedResourceCreateOptions
	Spec FedClusterRoleBindingSpec `json:"spec"`
}

type FedClusterRoleBindingSpec struct {
	Template RoleBindingTemplate `json:"template"`
}
