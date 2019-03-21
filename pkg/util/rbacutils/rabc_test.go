package rbacutils

import (
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestSRabcRule_Match(t *testing.T) {
	all := SRbacRule{Service: "*", Resource: "*", Action: "*"}
	compute := SRbacRule{Service: "compute", Resource: "*", Action: "*"}
	getOnly := SRbacRule{Service: "*", Resource: "*", Action: "get"}
	listOnly := SRbacRule{Service: "*", Resource: "*", Action: "list"}
	serverList := SRbacRule{Service: "compute", Resource: "server", Action: "list"}
	serverPerform := SRbacRule{Service: "compute", Resource: "server", Action: "perform", Extra: []string{"*"}}

	rule_server_list := []string{"compute", "server", "list"}
	rule_server_perform_start := []string{"compute", "server", "perform", "start"}
	rule_server_create := []string{"compute", "server", "create"}

	cases := []struct {
		inRule  SRbacRule
		inMatch []string
		want    bool
		count   int
	}{
		{all, rule_server_list, true, 0},
		{all, rule_server_perform_start, true, 0},
		{all, rule_server_create, true, 0},
		{compute, rule_server_list, true, 1},
		{compute, rule_server_perform_start, true, 1},
		{compute, rule_server_create, true, 1},
		{getOnly, rule_server_list, false, 0},
		{getOnly, rule_server_perform_start, false, 0},
		{getOnly, rule_server_create, false, 0},
		{listOnly, rule_server_list, true, 1},
		{listOnly, rule_server_perform_start, false, 0},
		{listOnly, rule_server_create, false, 0},
		{serverList, rule_server_list, true, 3},
		{serverList, rule_server_perform_start, false, 0},
		{serverList, rule_server_create, false, 0},
		{serverPerform, rule_server_list, false, 0},
		{serverPerform, rule_server_perform_start, true, 3},
		{serverPerform, rule_server_create, false, 0},
	}

	for _, c := range cases {
		got, cnt, _ := c.inRule.match(c.inMatch[0], c.inMatch[1], c.inMatch[2], c.inMatch[3:]...)
		if got != c.want {
			t.Errorf("%#v %#v want %#v got %#v", c.inRule, c.inMatch, c.want, got)
		}
		if cnt != c.count {
			t.Errorf("%#v %#v want %#v got %#v", c.inRule, c.inMatch, c.count, cnt)
		}
	}
}

func TestContains(t *testing.T) {
	cases := []struct {
		left     SRbacRule
		right    SRbacRule
		contains bool
	}{
		{
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			true,
		},
		{
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			SRbacRule{Service: "compute", Resource: "*", Action: "*", Result: Allow},
			true,
		},
		{
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			SRbacRule{Service: "compute", Resource: "server", Action: "*", Result: Allow},
			true,
		},
		{
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			SRbacRule{Service: "compute", Resource: "server", Action: "list", Result: Allow},
			true,
		},
		{
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			SRbacRule{Service: "compute", Resource: "server", Action: "get", Extra: []string{"vnc"}, Result: Allow},
			true,
		},
		{
			SRbacRule{Service: "compute", Resource: "*", Action: "*", Result: Allow},
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			false,
		},
		{
			SRbacRule{Service: "compute", Resource: "server", Action: "*", Result: Allow},
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			false,
		},
		{
			SRbacRule{Service: "compute", Resource: "server", Action: "list", Result: Allow},
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			false,
		},
		{
			SRbacRule{Service: "compute", Resource: "server", Action: "get", Extra: []string{"vnc"}, Result: Allow},
			SRbacRule{Service: "*", Resource: "*", Action: "*", Result: Allow},
			false,
		},
	}

	for _, c := range cases {
		got := c.left.contains(&c.right)
		if got != c.contains {
			t.Errorf("%s contains %s want %#v got %#v", c.left, c.right, c.contains, got)
		}
	}
}

func TestSRabcPolicy_Encode(t *testing.T) {
	policyStr := `{
        "condition": "usercred.project != \"system\" && usercred.roles==\"projectowner\"",
        "is_admin": false,
        "policy": {
            "compute": {
				 "keypair": "allow",
				 "server": "deny",
                 "*": {
                      "*": "allow",
                      "create": "deny"
                 }
            },
			"meter": {
				 "*": "allow"
			}
        }
    }`
	policyJson, err := jsonutils.ParseString(policyStr)
	if err != nil {
		t.Errorf("fail to parse json string %s", err)
		return
	}

	policy := SRbacPolicy{}

	err = policy.Decode(policyJson)
	if err != nil {
		t.Errorf("decode error %s", err)
		return
	}

	policyJson1, err := policy.Encode()
	if err != nil {
		t.Errorf("encode error %s", err)
		return
	}

	policy2 := SRbacPolicy{}

	err = policy2.Decode(policyJson1)
	if err != nil {
		t.Errorf("decode error 2 %s", err)
		return
	}

	policyJson2, err := policy2.Encode()
	if err != nil {
		t.Errorf("encode error 2 %s", err)
		return
	}

	policyStr1 := policyJson1.PrettyString()
	policyStr2 := policyJson2.PrettyString()

	if policyStr1 != policyStr2 {
		t.Errorf("%s != %s", policyStr1, policyStr2)
		return
	}

	t.Logf("%s", policyStr1)
}

func TestSRabcPolicy_Explain(t *testing.T) {
	policyStr := `{
        "condition": "usercred.project != \"system\" && usercred.roles==\"projectowner\"",
        "is_admin": false,
        "policy": {
            "compute": {
				 "keypair": "allow",
				 "server": "deny",
                 "*": {
                      "*": "allow",
                      "create": "deny"
                 }
            },
			"meter": {
				 "*": "allow"
			},
			"k8s": "allow"
        }
    }`
	policyJson, err := jsonutils.ParseString(policyStr)
	if err != nil {
		t.Errorf("fail to parse json string %s", err)
		return
	}

	policy := SRbacPolicy{}

	err = policy.Decode(policyJson)
	if err != nil {
		t.Errorf("decode error %s", err)
		return
	}

	request := [][]string{
		{"compute", "keypair", "list"},
		{"compute", "server", "list"},
		{"compute", "server", "get", "vnc"},
		{"compute", "keypair", "create"},
		{"meter", "price", "list"},
		{"image", "image", "list"},
		{"k8s", "pod", "list"},
	}

	output := policy.Explain(request)

	t.Logf("%#v", output)
}

func TestConditionParser(t *testing.T) {
	condition := `tenant=="system" && roles.contains("admin")`
	tenants := searchMatchTenants(condition)
	t.Logf("%s", tenants)
	roles := searchMatchRoles(condition)
	t.Logf("%s", roles)
}

func TestSRbacPolicyMatch(t *testing.T) {
	cases := []struct {
		policy   SRbacPolicy
		userCred mcclient.TokenCredential
		want     bool
	}{
		{
			SRbacPolicy{},
			&mcclient.SSimpleToken{},
			true,
		},
		{
			SRbacPolicy{},
			nil,
			true,
		},
		{
			SRbacPolicy{
				Projects: []string{"system"},
			},
			&mcclient.SSimpleToken{
				Project: "system",
			},
			true,
		},
		{
			SRbacPolicy{
				Projects: []string{"system"},
			},
			&mcclient.SSimpleToken{
				Project: "demo",
			},
			false,
		},
		{
			SRbacPolicy{
				Projects: []string{"system"},
				Roles:    []string{"admin"},
			},
			&mcclient.SSimpleToken{
				Project: "system",
				Roles:   "admin",
			},
			true,
		},
		{
			SRbacPolicy{
				Projects: []string{"system"},
				Roles:    []string{"admin"},
			},
			&mcclient.SSimpleToken{
				Project: "system",
				Roles:   "admin,_member_",
			},
			true,
		},
		{
			SRbacPolicy{
				Projects: []string{"system"},
				Roles:    []string{"admin"},
			},
			&mcclient.SSimpleToken{
				Project: "system",
				Roles:   "_member_",
			},
			false,
		},
		{
			SRbacPolicy{
				Projects: []string{"system"},
				Roles:    []string{"admin"},
			},
			nil,
			false,
		},
	}
	for _, c := range cases {
		got := c.policy.Match(c.userCred)
		if got != c.want {
			t.Errorf("%#v %#v got %v want %v", c.policy, c.userCred, got, c.want)
		}
	}
}
