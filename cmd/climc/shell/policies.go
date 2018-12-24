package shell

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type PolicyListOptions struct {
		Limit  int64  `help:"Limit, default 0, i.e. no limit"`
		Offset int64  `help:"Offset, default 0, i.e. no offset"`
		Search string `help:"Search by name"`
		Type   string `help:"filter by type"`
		Format string `help:"policy format, default to yaml" default:"yaml" choices:"yaml|json"`
	}
	R(&PolicyListOptions{}, "policy-list", "List all policies", func(s *mcclient.ClientSession, args *PolicyListOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.Format), "format")
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
		policyBytes, err := ioutil.ReadFile(args.FILE)
		if err != nil {
			return err
		}

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TYPE), "type")
		params.Add(jsonutils.NewString(string(policyBytes)), "policy")

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
		policyId, err := modules.Policies.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "type")
		}
		if len(args.File) > 0 {
			policyBytes, err := ioutil.ReadFile(args.File)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(string(policyBytes)), "policy")
		}
		result, err := modules.Policies.Patch(s, policyId, params)
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
		policyId, err := modules.Policies.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := modules.Policies.Delete(s, policyId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type PolicyShowOptions struct {
		ID     string `help:"ID of policy"`
		Format string `help:"policy format, default yaml" default:"yaml" choices:"yaml|json"`
	}
	R(&PolicyShowOptions{}, "policy-show", "Show policy", func(s *mcclient.ClientSession, args *PolicyShowOptions) error {
		query := jsonutils.NewDict()
		query.Add(jsonutils.NewString(args.Format), "format")
		result, err := modules.Policies.Get(s, args.ID, query)
		if err != nil {
			return err
		}
		yaml, err := result.GetString("policy")
		if err != nil {
			return err
		}
		fmt.Println(yaml)
		return nil
	})

	type PolicyEditOptions struct {
		ID string `help:"ID of policy"`
	}
	R(&PolicyEditOptions{}, "policy-edit", "Edit and update policy", func(s *mcclient.ClientSession, args *PolicyEditOptions) error {
		query := jsonutils.NewDict()
		query.Add(jsonutils.NewString("yaml"), "format")
		result, err := modules.Policies.Get(s, args.ID, query)
		if err != nil {
			return err
		}
		policyId, err := result.GetString("id")
		if err != nil {
			return err
		}
		yaml, err := result.GetString("policy")
		if err != nil {
			return err
		}

		tmpfile, err := ioutil.TempFile("", "policy-blob")
		if err != nil {
			return err
		}
		defer os.Remove(tmpfile.Name()) // clean up

		if _, err := tmpfile.Write([]byte(yaml)); err != nil {
			return err
		}
		if err := tmpfile.Close(); err != nil {
			return err
		}

		cmd := exec.Command("vim", tmpfile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			return err
		}

		params := jsonutils.NewDict()
		policyBytes, err := ioutil.ReadFile(tmpfile.Name())
		if err != nil {
			return err
		}
		params.Add(jsonutils.NewString(string(policyBytes)), "policy")

		result, err = modules.Policies.Patch(s, policyId, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	type PolicyAdminCapableOptions struct {
		User       string   `help:"For user"`
		UserDomain string   `help:"Domain for user"`
		Project    string   `help:"Role assignments for project"`
		Role       []string `help:"Roles"`
	}
	R(&PolicyAdminCapableOptions{}, "policy-admin-capable", "Check admin capable", func(s *mcclient.ClientSession, args *PolicyAdminCapableOptions) error {
		auth.InitFromClientSession(s)
		policy.EnableGlobalRbac(15*time.Second, 15*time.Second, false)

		var token mcclient.TokenCredential
		if len(args.User) > 0 {
			token = &mcclient.SSimpleToken{
				Domain:  args.UserDomain,
				User:    args.User,
				Project: args.Project,
				Roles:   strings.Join(args.Role, ","),
			}
		} else {
			token = s.GetToken()
		}

		capable := policy.PolicyManager.IsAdminCapable(token)
		fmt.Printf("%v\n", capable)

		return nil
	})

	type PolicyExplainOptions struct {
		User       string   `help:"For user"`
		UserDomain string   `help:"Domain for user"`
		Project    string   `help:"Role assignments for project"`
		Role       []string `help:"Roles"`
		Request    []string `help:"explain request, in format of key:is_admin:service:resource:action:extra"`
		Name       string   `help:"policy name"`
	}
	R(&PolicyExplainOptions{}, "policy-explain", "Explain policy result", func(s *mcclient.ClientSession, args *PolicyExplainOptions) error {
		auth.InitFromClientSession(s)
		policy.EnableGlobalRbac(15*time.Second, 15*time.Second, false)

		req := jsonutils.NewDict()
		for i := 0; i < len(args.Request); i += 1 {
			parts := strings.Split(args.Request[i], ":")
			if len(parts) < 3 {
				return fmt.Errorf("invalid request, should be in the form of key:is_admin:service[:resource:action:extra]")
			}
			key := parts[0]
			data := make([]jsonutils.JSONObject, 1)
			if parts[1] == "true" {
				data[0] = jsonutils.JSONTrue
			} else if parts[1] == "false" {
				data[0] = jsonutils.JSONFalse
			} else {
				return fmt.Errorf("invalid request, is_admin should be true|false")
			}
			for i := 2; i < len(parts); i += 1 {
				data = append(data, jsonutils.NewString(parts[i]))
			}
			req.Add(jsonutils.NewArray(data...), key)
		}
		fmt.Println(req.String())

		var token mcclient.TokenCredential
		if len(args.User) > 0 {
			token = &mcclient.SSimpleToken{
				Domain:  args.UserDomain,
				User:    args.User,
				Project: args.Project,
				Roles:   strings.Join(args.Role, ","),
			}
		} else {
			token = s.GetToken()
		}

		result, err := policy.PolicyManager.ExplainRpc(token, req, args.Name)
		if err != nil {
			return err
		}
		printObject(result)

		return nil
	})
}
