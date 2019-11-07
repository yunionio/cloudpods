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
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"
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
	NAME string

	Cert string `required:"true" json:"-" help:"path to certificate file"`
	Pkey string `required:"true" json:"-" help:"path to private key file"`
}

func (opts *LoadbalancerCertificateCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
	if err != nil {
		return nil, err
	}
	paramsCertKey, err := loadbalancerCertificateLoadFiles(opts.Cert, opts.Pkey, false)
	if err != nil {
		return nil, err
	}
	params.Update(paramsCertKey)
	return params, nil
}

type LoadbalancerCertificateGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerCertificateDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerCertificateListOptions struct {
	BaseListOptions

	PublicKeyAlgorithm string
	PublicKeyBitLen    *int
	SignatureAlgorithm string
	Cloudregion        string
	Usable             *bool `help:"List certificates are usable"`
}

type LoadbalancerCertificateUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	Cert string `json:"-" help:"path to certificate file"`
	Pkey string `json:"-" help:"path to private key file"`
}

func (opts *LoadbalancerCertificateUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	paramsCertKey, err := loadbalancerCertificateLoadFiles(opts.Cert, opts.Pkey, true)
	if err != nil {
		return nil, err
	}
	return paramsCertKey, nil
}
