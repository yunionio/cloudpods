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
	"yunion.io/x/onecloud/pkg/util/ldaputils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LdapSearchOptions struct {
		Base        string   `help:"base DN, e.g. OU=tech,DC=example,DC=com"`
		Objectclass string   `help:"objectclass, e.g. organizationalPerson"`
		Search      []string `help:"search conditions, in format of field:value"`
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
		entries, err := cli.Search(args.Base, args.Objectclass, search, nil)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			entry.PrettyPrint(2)
		}
		return nil
	})
}
