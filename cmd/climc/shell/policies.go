package shell

import (
	"io/ioutil"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type PolicyListOptions struct {
		Limit  int64  `help:"Limit, default 0, i.e. no limit"`
		Offset int64  `help:"Offset, default 0, i.e. no offset"`
		Search string `help:"Search by name"`
		Type   string `help:"filter by type"`
	}
	R(&PolicyListOptions{}, "policy-list", "List all policies", func(s *mcclient.ClientSession, args *PolicyListOptions) error {
		params := jsonutils.NewDict()
		if len(args.Search) > 0 {
			params.Add(jsonutils.NewString(args.Search), "type__icontains")
		}
		if args.Limit > 0 {
			params.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			params.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "type")
		}
		result, err := modules.Policies.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Policies.GetColumns(s))
		return nil
	})

	type PolicyCreateOptions struct {
		TYPE string `help:"type of the policy"`
		FILE string `help:"path to policy file"`
	}
	R(&PolicyCreateOptions{}, "policy-create", "Create a new policy", func(s *mcclient.ClientSession, args *PolicyCreateOptions) error {
		blob, err := ioutil.ReadFile(args.FILE)
		if err != nil {
			return err
		}
		jsonBlob, err := jsonutils.ParseYAML(string(blob))
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TYPE), "type")
		params.Add(jsonutils.NewString(jsonBlob.String()), "blob")

		result, err := modules.Policies.Create(s, params)
		if err != nil {
			return err
		}

		printObject(result)

		return nil
	})

	type PolicyPatchOptions struct {
		ID   string `help:"ID of policy"`
		File string `help:"path to policy file"`
		Type string `help:"policy type"`
	}
	R(&PolicyPatchOptions{}, "policy-patch", "Patch policy", func(s *mcclient.ClientSession, args *PolicyPatchOptions) error {
		params := jsonutils.NewDict()
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "type")
		}
		if len(args.File) > 0 {
			blob, err := ioutil.ReadFile(args.File)
			if err != nil {
				return err
			}
			jsonBlob, err := jsonutils.ParseYAML(string(blob))
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(jsonBlob.String()), "blob")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Policies.Patch(s, args.ID, params)
		if err != nil {
			return err
		}

		printObject(result)

		return nil
	})

	type PolicyDeleteOptions struct {
		ID string `help:"ID of policy"`
	}
	R(&PolicyDeleteOptions{}, "policy-delete", "Delete policy", func(s *mcclient.ClientSession, args *PolicyDeleteOptions) error {
		result, err := modules.Policies.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
