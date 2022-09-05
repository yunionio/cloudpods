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
	"strings"

	"gopkg.in/ldap.v3"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/ldaputils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func queryScope(scope string) int {
	if scope == api.QueryScopeOne {
		return ldap.ScopeSingleLevel
	} else {
		return ldap.ScopeWholeSubtree
	}
}

func init() {
	type LdapSearchOptions struct {
		Base        string   `help:"base DN, e.g. OU=tech,DC=example,DC=com"`
		Objectclass string   `help:"objectclass, e.g. organizationalPerson"`
		Search      []string `help:"search conditions, in format of field:value"`
		Field       []string `help:"retrieve field info"`
		Scope       string   `help:"query scope" choices:"one|sub" default:"sub"`
		PageLimit   uint32   `help:"page size" default:"100"`
		Limit       uint32   `help:"maximal output items"`
	}
	shellutils.R(&LdapSearchOptions{}, "search", "search ldap", func(cli *ldaputils.SLDAPClient, args *LdapSearchOptions) error {
		search := make(map[string]string)
		for _, s := range args.Search {
			colonPos := strings.IndexByte(s, ':')
			if colonPos <= 0 {
				return fmt.Errorf("invalid search condition %s", s)
			}
			search[s[:colonPos]] = s[(colonPos + 1):]
		}
		total, err := cli.Search(args.Base, args.Objectclass, search, "", args.Field, queryScope(args.Scope), args.PageLimit, args.Limit, func(offset uint32, entry *ldap.Entry) error {
			entry.PrettyPrint(2)
			return nil
		})
		if err != nil {
			return err
		}
		fmt.Println("Total:", total)
		return nil
	})

	type LdapAuthOptions struct {
		Base        string   `help:"base DN, e.g. OU=tech,DC=example,DC=com"`
		Objectclass string   `help:"objectclass, e.g. organizationalPerson"`
		ATTR        string   `help:"account attribute name"`
		ACCOUNT     string   `help:"account name to auth"`
		PASSWORD    string   `help:"Password to auth"`
		Field       []string `help:"retrieve field info"`
		Scope       string   `help:"query scope" choices:"one|sub" default:"sub"`
	}
	shellutils.R(&LdapAuthOptions{}, "auth", "authenticate against ldap", func(cli *ldaputils.SLDAPClient, args *LdapAuthOptions) error {
		entry, err := cli.Authenticate(args.Base, args.Objectclass, args.ATTR, args.ACCOUNT, args.PASSWORD, "", args.Field, queryScope(args.Scope))
		if err != nil {
			return err
		}
		entry.PrettyPrint(2)
		return nil
	})
}
