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

package oauth2

import (
	"context"

	api "yunion.io/x/onecloud/pkg/apis/identity"
)

type IOAuth2DriverFactory interface {
	NewDriver(appId string, secret string) IOAuth2Driver
	TemplateName() string
	IdpAttributeOptions() api.SIdpAttributeOptions
	ValidateConfig(conf api.SOAuth2IdpConfigOptions) error
}

type IOAuth2Driver interface {
	Authenticate(ctx context.Context, code string) (map[string][]string, error)
	GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error)
}

type IOAuth2Synchronizer interface {
	Sync(ctx context.Context, idpId string) error
}

type SOAuth2BaseDriver struct {
	AppId  string
	Secret string
}

var (
	oauth2DriverFactories = make(map[string]IOAuth2DriverFactory)
)

func Register(factory IOAuth2DriverFactory) {
	oauth2DriverFactories[factory.TemplateName()] = factory
}

func findDriverFactory(template string) IOAuth2DriverFactory {
	if factory, ok := oauth2DriverFactories[template]; ok {
		return factory
	}
	return nil
}
