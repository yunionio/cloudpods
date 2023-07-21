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

package options

import (
	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

type ElasticCacheAccountCreateOptions struct {
	Elasticcache     string `help:"elastic cache instance id"`
	Name             string
	Password         string
	AccountPrivilege string `help:"account privilege" choices:"read|write|repl" default:"read"`
}

type ElasticCacheBackupCreateOptions struct {
	Elasticcache string `help:"elastic cache instance id"`
	Name         string
}

type ElasticCacheAclCreateOptions struct {
	Elasticcache string `help:"elastic cache instance id"`
	Name         string
	IpList       string `help:"elastic cache acl ip list, split by ','"`
}

type ElasticCacheAclUpdateOptions struct {
	Id     string `help:"elastic cache acl id"`
	IpList string `help:"elastic cache acl ip list, split by ','"`
}

type ElasticCacheParameterUpdateOptions struct {
	Id    string `help:"elastic cache parameter id"`
	Value string `help:"elastic cache parameter value"`
}

type ElasticCacheIdOptions struct {
	ID string
}

type ElasticCacheRemoteUpdateOptions struct {
	ID string `json:"-"`
	computeapi.ElasticcacheRemoteUpdateInput
}

type ElasticCacheAutoRenewOptions struct {
	ElasticCacheIdOptions
	AutoRenew bool `help:"Set elastic cache auto renew or manual renew"`
}

func (o *ElasticCacheAutoRenewOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("auto_renew", jsonutils.NewBool(o.AutoRenew))
	return params, nil
}

type ElasticCacheRenewOptions struct {
	ID       string `json:"-"`
	Duration string `help:"valid duration of the elastic cache, e.g. 1H, 1D, 1W, 1M, 1Y, ADMIN ONLY option"`
}
