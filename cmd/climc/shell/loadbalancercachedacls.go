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
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	lbAclConvert := func(jd *jsonutils.JSONDict) error {
		jaeso, err := jd.Get("acl_entries")
		if err != nil {
			return err
		}
		aclEntries := options.AclEntries{}
		err = jaeso.Unmarshal(&aclEntries)
		if err != nil {
			return err
		}
		aclTextLines := aclEntries.String()
		jd.Set("acl_entries", jsonutils.NewString(aclTextLines))
		return nil
	}

	printLbAcl := func(jsonObj jsonutils.JSONObject) {
		jd, ok := jsonObj.(*jsonutils.JSONDict)
		if !ok {
			printObject(jsonObj)
			return
		}
		err := lbAclConvert(jd)
		if err != nil {
			printObject(jsonObj)
			return
		}
		printObject(jd)
	}
	printLbAclList := func(list *modulebase.ListResult, columns []string) {
		data := list.Data
		for _, jsonObj := range data {
			jd := jsonObj.(*jsonutils.JSONDict)
			err := lbAclConvert(jd)
			if err != nil {
				printList(list, columns)
				return
			}
		}
		printList(list, columns)
	}

	R(&options.LoadbalancerAclGetOptions{}, "lbacl-cache-show", "Show cached lbacl", func(s *mcclient.ClientSession, opts *options.LoadbalancerAclGetOptions) error {
		lbacl, err := modules.LoadbalancerCachedAcls.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printLbAcl(lbacl)
		return nil
	})

	type CachedLoadbalancerAclListOptions struct {
		options.LoadbalancerAclListOptions
		AclId string `help:"local acl id" `
	}

	R(&CachedLoadbalancerAclListOptions{}, "lbacl-cache-list", "List cached lbacls", func(s *mcclient.ClientSession, opts *CachedLoadbalancerAclListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerCachedAcls.List(s, params)
		if err != nil {
			return err
		}
		printLbAclList(result, modules.LoadbalancerCachedAcls.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerAclDeleteOptions{}, "lbacl-cache-purge", "Purge cached lbacl", func(s *mcclient.ClientSession, opts *options.LoadbalancerAclDeleteOptions) error {
		lbacl, err := modules.LoadbalancerCachedAcls.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printLbAcl(lbacl)
		return nil
	})
	R(&options.LoadbalancerAclDeleteOptions{}, "lbacl-cache-delete", "Delete cached lbacl", func(s *mcclient.ClientSession, opts *options.LoadbalancerAclDeleteOptions) error {
		lbacl, err := modules.LoadbalancerCachedAcls.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printLbAcl(lbacl)
		return nil
	})
}
