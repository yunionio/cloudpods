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

package shell

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type KeypairList struct {
		options.BaseListOptions
	}

	R(&KeypairList{}, "keypair-list", "List keypairs.", func(s *mcclient.ClientSession, args *KeypairList) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.Keypairs.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.Keypairs.GetColumns(s))
		return nil
	})

	type KeypairCreate struct {
		NAME   string `help:"Name of keypair to be created"`
		Scheme string `help:"Scheme of keypair, default is RSA" choices:"RSA|DSA" default:"RSA"`
		Desc   string `help:"Short description of keypair"`
	}

	R(&KeypairCreate{}, "keypair-create", "Create a new keypair", func(s *mcclient.ClientSession, args *KeypairCreate) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Scheme) > 0 {
			params.Add(jsonutils.NewString(args.Scheme), "scheme")
		}

		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}

		result, e := modules.Keypairs.Create(s, params)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	type KeypairUpdate struct {
		ID   string `help:"ID of keypair to be updated"`
		Name string `help:"New name of keypair"`
		Desc string `help:"Short description of keypair"`
	}

	R(&KeypairUpdate{}, "keypair-update", "Update a keypair", func(s *mcclient.ClientSession, args *KeypairUpdate) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}

		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}

		result, e := modules.Keypairs.Update(s, args.ID, params)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	type KeypairDelete struct {
		ID string `help:"ID of keypair to be deleted"`
	}

	R(&KeypairDelete{}, "keypair-delete", "Delete a keypair", func(s *mcclient.ClientSession, args *KeypairDelete) error {
		result, e := modules.Keypairs.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	type KeypairShow struct {
		ID string `help:"ID of keypair to be shown"`
	}

	R(&KeypairShow{}, "keypair-show", "Show details of a keypair", func(s *mcclient.ClientSession, args *KeypairShow) error {
		result, e := modules.Keypairs.Get(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	type KeypairImport struct {
		NAME      string `help:"Name of keypair to be imported"`
		PublicKey string `help:"Filename of public key file, or public key can be supplied via stdin"`
		Desc      string `help:"Short description of keypair"`
	}

	R(&KeypairImport{}, "keypair-import", "Create a new keypair with a existing public key", func(s *mcclient.ClientSession, args *KeypairImport) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.PublicKey) > 0 {
			content, e := ioutil.ReadFile(args.PublicKey)
			if e != nil {
				params.Add(jsonutils.NewString(args.PublicKey), "public_key")
			} else {
				params.Add(jsonutils.NewString(string(content)), "public_key")
			}
		} else {
			return fmt.Errorf("no public key provided")
		}

		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}

		result, e := modules.Keypairs.Create(s, params)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	type KeypairPrivateKey struct {
		ID string `help:"ID of keypair to fetch"`
	}

	R(&KeypairPrivateKey{}, "keypair-privatekey", "Fetch the private key of a keypair, this can be done once only", func(s *mcclient.ClientSession, args *KeypairPrivateKey) error {
		result, e := modules.Keypairs.GetSpecific(s, args.ID, "privatekey", nil)
		if e != nil {
			return e
		}
		key, e := result.GetString("private_key")
		if e != nil {
			return fmt.Errorf("Private key has been fetched")
		}
		fmt.Printf("%s", key)
		return nil
	})
}
