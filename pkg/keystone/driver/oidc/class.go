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

package oidc

import (
	"context"

	"yunion.io/x/onecloud/pkg/util/oidcutils/client"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/driver/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SOIDCDriverClass struct{}

func (self *SOIDCDriverClass) SingletonInstance() bool {
	return false
}

func (self *SOIDCDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncOnAuth
}

func (self *SOIDCDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewOIDCDriver(idpId, idpName, template, targetDomainId, conf)
}

func (self *SOIDCDriverClass) Name() string {
	return api.IdentityDriverOIDC
}

func (self *SOIDCDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, tconf api.TConfigs) (api.TConfigs, error) {
	conf := api.SOIDCIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf[api.IdentityDriverOIDC])
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return tconf, errors.Wrap(err, "unmarshal config")
	}
	cli := client.NewOIDCClient(conf.ClientId, conf.ClientSecret, 30, false)
	if len(conf.Endpoint) > 0 {
		err := cli.FetchConfiguration(ctx, conf.Endpoint)
		if err != nil {
			return tconf, errors.Wrapf(err, "invaoid endpoint %s", conf.Endpoint)
		}
	}
	conf.SIdpAttributeOptions, err = utils.ValidateConfig(conf.SIdpAttributeOptions, userCred)
	if err != nil {
		return tconf, errors.Wrap(err, "ValidateConfig")
	}
	nconf := make(map[string]jsonutils.JSONObject)
	err = jsonutils.Marshal(conf).Unmarshal(&nconf)
	if err != nil {
		return tconf, errors.Wrap(err, "Unmarshal new config")
	}
	tconf[api.IdentityDriverOIDC] = nconf
	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SOIDCDriverClass{})
}
