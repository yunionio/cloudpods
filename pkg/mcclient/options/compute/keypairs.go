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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type KeypairList struct {
	options.BaseListOptions
}

func (self *KeypairList) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(self)
}

type KeypairCreate struct {
	NAME      string `help:"Name of keypair to be created"`
	Scheme    string `help:"Scheme of keypair, default is RSA" choices:"RSA" default:"RSA"`
	PublicKey string `help:"Publickey of keypair"`
	Desc      string `help:"Short description of keypair"`
}

func (args *KeypairCreate) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(args.NAME), "name")
	if len(args.Scheme) > 0 {
		params.Add(jsonutils.NewString(args.Scheme), "scheme")
	}

	if len(args.PublicKey) > 0 {
		params.Add(jsonutils.NewString(args.PublicKey), "public_key")
	}

	if len(args.Desc) > 0 {
		params.Add(jsonutils.NewString(args.Desc), "description")
	}
	return params, nil
}

type KeyPairIdOptions struct {
	ID string `help:"ID of keypair"`
}

func (self *KeyPairIdOptions) GetId() string {
	return self.ID
}

func (self *KeyPairIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type KeypairUpdate struct {
	KeyPairIdOptions
	Name string `help:"New name of keypair"`
	Desc string `help:"Short description of keypair"`
}

func (args *KeypairUpdate) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(args.Name) > 0 {
		params.Add(jsonutils.NewString(args.Name), "name")
	}

	if len(args.Desc) > 0 {
		params.Add(jsonutils.NewString(args.Desc), "description")
	}
	return params, nil
}
