package ldaputils

import (
	"fmt"

	"gopkg.in/ldap.v3"

	"github.com/pkg/errors"
	"strings"
)

var (
	ErrUserNotFound      = errors.New("not found")
	ErrUserDuplicate     = errors.New("user id duplicate")
	ErrUserBadCredential = errors.New("bad credential")
)

type SLDAPClient struct {
	url      string
	account  string
	password string
	baseDN   string

	isDebug bool

	conn *ldap.Conn
}

func NewLDAPClient(url, account, password string, baseDN string, isDebug bool) *SLDAPClient {
	return &SLDAPClient{
		url:      url,
		account:  account,
		password: password,
		baseDN:   baseDN,

		isDebug: isDebug,
	}
}

func (cli *SLDAPClient) Connect() error {
	conn, err := ldap.DialURL(cli.url)
	if err != nil {
		return errors.WithMessage(err, "DiaURL")
	}
	cli.conn = conn

	return cli.bind()
}

func (cli *SLDAPClient) bind() error {
	if len(cli.account) > 0 {
		err := cli.conn.Bind(cli.account, cli.password)
		if err != nil {
			return errors.WithMessage(err, "Bind")
		}
	}
	return nil
}

func (cli *SLDAPClient) Close() {
	if cli.conn != nil {
		cli.conn.Close()
		cli.conn = nil
	}
}

func (cli *SLDAPClient) Authenticate(baseDN string, objClass string, uidAttr string, uname string, passwd string, fields []string) (*ldap.Entry, error) {
	attrMap := make(map[string]string)
	attrMap[uidAttr] = uname
	entries, err := cli.Search(baseDN, objClass, attrMap, fields)
	if err != nil {
		return nil, errors.WithMessage(err, "Search")
	}
	if len(entries) == 0 {
		return nil, ErrUserNotFound
	}
	if len(entries) > 1 {
		return nil, ErrUserDuplicate
	}
	defer cli.bind()
	entry := entries[0]
	err = cli.conn.Bind(entry.DN, passwd)
	if err != nil {
		return nil, ErrUserBadCredential
	}
	return entry, nil
}

func (cli *SLDAPClient) Search(base string, objClass string, condition map[string]string, fields []string) ([]*ldap.Entry, error) {
	searches := strings.Builder{}
	if len(condition) == 0 && len(objClass) == 0 {
		searches.WriteString("(objectClass=*)")
	}
	if len(objClass) > 0 {
		searches.WriteString("(objectClass=")
		searches.WriteString(objClass)
		searches.WriteString(")")
	}
	for k, v := range condition {
		searches.WriteString("(")
		searches.WriteString(k)
		searches.WriteString("=")
		searches.WriteString(v)
		searches.WriteString(")")
	}
	searchStr := fmt.Sprintf("(&%s)", searches.String())

	if len(base) == 0 {
		base = cli.baseDN
	}

	searchRequest := ldap.NewSearchRequest(
		base, // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		searchStr,
		fields, // A list attributes to retrieve
		nil,
	)
	sr, err := cli.conn.Search(searchRequest)
	if err != nil {
		return nil, errors.Wrap(err, "Search")
	}

	return sr.Entries, nil
}
