package shell

import (
	"io/ioutil"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type PolicyListOptions struct {
	}
	R(&PolicyListOptions{}, "policy-list", "List all policies", func(s *mcclient.ClientSession, args *PolicyListOptions) error {
		result, err := modules.Policies.List(s, nil)
		if err != nil {
			return err
		}
		printList(result, modules.Policies.GetColumns(s))
		return nil
	})

	type PolicyCreateOptions struct {
		FILE string `help:"path to policy file"`
	}
	R(&PolicyCreateOptions{}, "policy-create", "Create a new policy", func(s *mcclient.ClientSession, args *PolicyCreateOptions) error {
		blob, err := ioutil.ReadFile(args.FILE)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString("application/json"), "type")
		params.Add(jsonutils.NewString(string(blob)), "blob")

		result, err := modules.Policies.Create(s, params)
		if err != nil {
			return err
		}

		printObject(result)

		return nil
	})

	type PolicyPatchOptions struct {
		ID   string `help:"ID of policy"`
		FILE string `help:"path to policy file"`
	}
	R(&PolicyPatchOptions{}, "policy-patch", "Patch policy", func(s *mcclient.ClientSession, args *PolicyPatchOptions) error {
		blob, err := ioutil.ReadFile(args.FILE)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString("application/json"), "type")
		params.Add(jsonutils.NewString(string(blob)), "blob")

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
