package shell

import (
	"yunion.io/x/jsonutils"

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
	printLbAclList := func(list *modules.ListResult, columns []string) {
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
