package rbacutils

import (
	"testing"

	"yunion.io/x/jsonutils"
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

func TestSRbacPolicy_Allow(t *testing.T) {
	userCredStr := `{"domain":"Default","domain_id":"default","expires":"2018-10-28T05:33:54.000000Z","roles":"admin,teamleader","tenant":"system","tenant_id":"5d65667d112e47249ae66dbd7bc07030","token":"gAAAAABb0_jC2Qz2PpB00-pieLi4exKXq4O3QrvoqerpqoSxbp9pOLLdNWaAg0cPcd8eAjkiPhSo7VWQAVodoxnad95LdNbUf_1It8R_wXVDtO20caB7oLcas1oQt8b1cG0a7qagauP0iWVSW_dq_e92rD5Hd3SHn3Lw6ycrp_eHLskz_8EbPiI","user":"sysadmin","user_id":"dddf386b6ff24572b2e6a771d768495e"}`

	userCred, err := jsonutils.ParseString(userCredStr)
	if err != nil {
		t.Errorf("parse json fail %s", err)
		return
	}

	cases := []struct {
		policy string
		ops    []string
		want   TRbacResult
	}{
		{
			`{
    "condition": "tenant==\"system\" && roles.contains(\"admin\")",
    "is_admin": true,
    "policy": {
        "*": "allow"
    }
}`,
			[]string{"compute", "servers", "list"},
			Allow,
		},
		{
			`{"is_admin":"false","policy":{"*":{"*":{"*":"allow","delete":"deny"}}}}`,
			[]string{"compute", "servers", "delete"},
			Deny,
		},
	}

	for _, c := range cases {
		policyJson, err := jsonutils.ParseString(c.policy)
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

		isAllow := policy.Allow(userCred, c.ops[0], c.ops[1], c.ops[2])
		if isAllow != c.want {
			t.Errorf("%s %#v expect %#v, but get %#v", policyJson.String(), c.ops, c.want, isAllow)
			return
		}
	}
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

func TestTRbacResult_IsHigherPrivilege(t *testing.T) {
	cases := []struct {
		t1 TRbacResult
		t2 TRbacResult
		want bool
	} {
		{Allow, Allow, false},
		{Allow, OwnerAllow, true},
		{Deny, Deny, false},
		{Deny, OwnerAllow, false},
		{Deny, Allow, false},
		{OwnerAllow, Allow, false},
	}

	for _, c := range cases {
		got := c.t1.IsHigherPrivilege(c.t2)
		if got != c.want {
			t.Errorf("%s IsHigherPrivilege %s want %v got %v", c.t1, c.t2, c.want, got)
		}
	}
}