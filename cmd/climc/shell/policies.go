package shell

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

func getPolicy(s *mcclient.ClientSession, id string) (jsonutils.JSONObject, error) {
	result, err := modules.Policies.GetById(s, id, nil)
	if err == nil {
		return result, nil
	}
	jsonErr := err.(*httputils.JSONClientError)
	if jsonErr.Code != 404 {
		return nil, err
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(id), "type")
	listResult, err := modules.Policies.List(s, params)
	if err != nil {
		return nil, err
	}
	if listResult.Total == 1 {
		return listResult.Data[0], nil
	} else if listResult.Total == 0 {
		return nil, &httputils.JSONClientError{Code: 404, Class: "NotFound",
			Details: fmt.Sprintf("%d not found", id)}
	} else {
		return nil, &httputils.JSONClientError{Code: 409, Class: "Conflict",
			Details: fmt.Sprintf("multiple %d found", id)}
	}
}

func getPolicyId(s *mcclient.ClientSession, name string) (string, error) {
	policyJson, err := getPolicy(s, name)
	if err != nil {
		return "", err
	}
	return policyJson.GetString("id")
}

func getPolicyYaml(s *mcclient.ClientSession, id string) (string, error) {
	result, err := modules.Policies.GetById(s, id, nil)
	if err != nil {
		return "", err
	}
	blobStr, err := result.GetString("blob")
	if err != nil {
		return "", err
	}
	blobJson, err := jsonutils.ParseString(blobStr)
	if err != nil {
		return "", err
	}
	return blobJson.YAMLString(), nil
}

func patchPolicyYaml(s *mcclient.ClientSession, policyId string, typeStr string, fileName string) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(typeStr) > 0 {
		params.Add(jsonutils.NewString(typeStr), "type")
	}
	if len(fileName) > 0 {
		blob, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		jsonBlob, err := jsonutils.ParseYAML(string(blob))
		if err != nil {
			return nil, err
		}
		params.Add(jsonutils.NewString(jsonBlob.String()), "blob")
	}
	if params.Size() == 0 {
		return nil, InvalidUpdateError()
	}
	return modules.Policies.Patch(s, policyId, params)
}

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
		policyId, err := getPolicyId(s, args.ID)
		if err != nil {
			return err
		}
		result, err := patchPolicyYaml(s, policyId, args.Type, args.File)
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
		policyId, err := getPolicyId(s, args.ID)
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
		ID string `help:"ID of policy"`
	}
	R(&PolicyShowOptions{}, "policy-show", "Show policy", func(s *mcclient.ClientSession, args *PolicyShowOptions) error {
		policyId, err := getPolicyId(s, args.ID)
		if err != nil {
			return err
		}
		yaml, err := getPolicyYaml(s, policyId)
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
		policyId, err := getPolicyId(s, args.ID)
		if err != nil {
			return err
		}
		yaml, err := getPolicyYaml(s, policyId)
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

		result, err := patchPolicyYaml(s, policyId, "", tmpfile.Name())
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

}
