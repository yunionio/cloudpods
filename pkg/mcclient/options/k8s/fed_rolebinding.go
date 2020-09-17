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

const (
	RBACAPIGroup = "rbac.authorization.k8s.io"
)

type FedRoleBindingCreateOpt struct {
	FedNamespaceResourceCreateOptions
	RoleRef RoleRef `json:"roleRef"`
	Subject Subject `help:"Subject is role bind subject, e.g: User=jane"`
}

func (o *FedRoleBindingCreateOpt) ToInput() (*FedRoleBindingCreateInput, error) {
	input := &FedRoleBindingCreateInput{
		FedNamespaceResourceCreateOptions: o.FedNamespaceResourceCreateOptions,
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

func (o *FedRoleBindingCreateOpt) Params() (jsonutils.JSONObject, error) {
	input, err := o.ToInput()
	if err != nil {
		return nil, err
	}
	return input.JSON(input), nil
}

func validateRoleRef(ref *RoleRef) error {
	if ref.APIGroup == "" {
		ref.APIGroup = RBACAPIGroup
	}
	if ref.Kind == "" {
		return errors.Errorf("roleRef kind must specified")
	}
	if ref.Name == "" {
		return errors.Errorf("roleRef name must specified")
	}
	return nil
}

func validateSubject(sub *Subject) error {
	if sub.APIGroup == "" {
		sub.APIGroup = RBACAPIGroup
	}
	if sub.Kind == "" {
		return errors.Errorf("subject kind must specified")
	}
	if sub.Name == "" {
		return errors.Errorf("subject  name must specified")
	}
	return nil
}

type RoleRef struct {
	Kind     string `help:"Role kind" choices:"ClusterRole|Role" json:"kind"`
	Name     string `help:"Name is the name of role" json:"name"`
	APIGroup string `json:"apiGroup"`
}

type FedRoleBindingCreateInput struct {
	FedNamespaceResourceCreateOptions
	Spec FedRoleBindingSpec `json:"spec"`
}

type FedRoleBindingSpec struct {
	Template RoleBindingTemplate `json:"template"`
}

type RoleBindingTemplate struct {
	RoleRef  RoleRef   `json:"roleRef"`
	Subjects []Subject `json:"subjects"`
}

type Subject struct {
	Kind      string `json:"kind"`
	APIGroup  string `json:"apiGroup"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
