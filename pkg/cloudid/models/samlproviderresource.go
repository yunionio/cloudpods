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

package models

import "yunion.io/x/pkg/errors"

// +onecloud:swagger-gen-ignore
type SAMLProviderResourceBaseManager struct {
}

type SAMLProviderResourceBase struct {
	SAMLProviderId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" json:"saml_provider_id"`
}

func (self *SAMLProviderResourceBase) GetSAMLProvider() (*SSAMLProvider, error) {
	sp, err := SAMLProviderManager.FetchById(self.SAMLProviderId)
	if err != nil {
		return nil, errors.Wrap(err, "SAMLProviderManager.FetchById")
	}
	return sp.(*SSAMLProvider), nil
}
