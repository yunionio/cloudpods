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

package idp

import (
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

type SSAMLServiceProvider struct {
	desc samlutils.EntityDescriptor

	Username string
}

func (sp *SSAMLServiceProvider) GetEntityId() string {
	return sp.desc.EntityId
}

func (sp *SSAMLServiceProvider) GetPostAssertionConsumerServiceUrl() string {
	for _, srv := range sp.desc.SPSSODescriptor.AssertionConsumerServices {
		if srv.Binding == samlutils.BINDING_HTTP_POST {
			return srv.Location
		}
	}
	return ""
}

func (sp *SSAMLServiceProvider) IsValid() error {
	if sp.desc.SPSSODescriptor == nil {
		return errors.Wrap(httperrors.ErrInputParameter, "missing SPSSODescriptor")
	}
	if sp.GetEntityId() == "" {
		return errors.Wrap(httperrors.ErrInputParameter, "empty entityID")
	}
	if sp.GetPostAssertionConsumerServiceUrl() == "" {
		return errors.Wrap(httperrors.ErrInputParameter, "empty HTTP_Post AssertionConsumerServiceUrl")
	}
	return nil
}
