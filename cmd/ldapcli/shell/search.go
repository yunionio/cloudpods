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
