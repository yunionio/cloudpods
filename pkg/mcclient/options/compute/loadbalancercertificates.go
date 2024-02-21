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

package compute

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func loadbalancerCertificateLoadFiles(cert, pkey string, allowEmpty bool) (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	pathM := map[string]string{
		"certificate": cert,
		"private_key": pkey,
	}
	for fieldName, path := range pathM {
		if path == "" {
			if allowEmpty {
				continue
			} else {
				return nil, fmt.Errorf("%s: empty path", fieldName)
			}
		}
		d, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: read %s: %s", fieldName, path, err)
		}
		if len(d) == 0 {
			return nil, fmt.Errorf("%s: empty file %s", fieldName, path)
		}
		params.Set(fieldName, jsonutils.NewString(string(d)))
	}
	return params, nil
}

type LoadbalancerCertificateCreateOptions struct {
	options.SharableProjectizedResourceBaseCreateInput

	NAME string

	Manager string `json:"manager_id"`
	Region  string `json:"cloudregion"`

	Cert string `required:"true" json:"-" help:"path to certificate file"`
	Pkey string `required:"true" json:"-" help:"path to private key file"`
}

func (opts *LoadbalancerCertificateCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}

	sp, err := opts.SharableProjectizedResourceBaseCreateInput.Params()
	if err != nil {
		return nil, err
	}

	params.Update(sp)

	paramsCertKey, err := loadbalancerCertificateLoadFiles(opts.Cert, opts.Pkey, false)
	if err != nil {
		return nil, err
	}
	params.Update(paramsCertKey)
	return params, nil
}

type LoadbalancerCertificateIdOptions struct {
	ID string `json:"-"`
}

func (opts *LoadbalancerCertificateIdOptions) GetId() string {
	return opts.ID
}

func (opts *LoadbalancerCertificateIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type LoadbalancerCertificateDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerCertificateListOptions struct {
	options.BaseListOptions

	PublicKeyAlgorithm string
	PublicKeyBitLen    *int
	SignatureAlgorithm string
	Cloudregion        string
	Usable             *bool `help:"List certificates are usable"`
}

func (opts *LoadbalancerCertificateListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type LoadbalancerCertificateUpdateOptions struct {
	LoadbalancerCertificateIdOptions
	Name string

	Cert string `json:"-" help:"path to certificate file"`
	Pkey string `json:"-" help:"path to private key file"`
}

func (opts *LoadbalancerCertificateUpdateOptions) Params() (jsonutils.JSONObject, error) {
	paramsCertKey, err := loadbalancerCertificateLoadFiles(opts.Cert, opts.Pkey, true)
	if err != nil {
		return nil, err
	}

	return paramsCertKey, nil
}

type LoadbalancerCertificatePublicOptions struct {
	LoadbalancerCertificateIdOptions
	options.SharableResourcePublicBaseOptions
}

func (opts *LoadbalancerCertificatePublicOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.SharableResourcePublicBaseOptions), nil
}

type LoadbalancerCertificatePrivateOptions struct {
	ID string `json:"-"`
}

type LoadbalancerCachedCertificateListOptions struct {
	LoadbalancerCertificateListOptions

	CertificateId string `help:"related certificate id"`
}

func (opts *LoadbalancerCachedCertificateListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type LoadbalancerCachedCertificateCreateOptions struct {
	CLOUDPROVIDER string `help:"cloud provider id"`
	CLOUDREGION   string `help:"cloud region id"`
	CERTIFICATE   string `help:"certificate id"`
}

func (opts *LoadbalancerCachedCertificateCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}
