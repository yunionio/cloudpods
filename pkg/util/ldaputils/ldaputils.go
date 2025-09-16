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

package ldaputils

import (
	"encoding/hex"
	"fmt"
	"strings"

	"gopkg.in/ldap.v3"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

var (
	ErrUserNotFound      = errors.Error("not found")
	ErrUserDuplicate     = errors.Error("user id duplicate")
	ErrUserBadCredential = errors.Error("bad credential")

	binaryAttributes = []string{
		"objectGUID",
		"objectSid",
	}
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
		return errors.Wrap(err, "DiaURL")
	}
	cli.conn = conn

	return cli.bind()
}

func (cli *SLDAPClient) bind() error {
	if len(cli.account) > 0 {
		err := cli.conn.Bind(cli.account, cli.password)
		if err != nil {
			return errors.Wrap(err, "Bind")
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

func (cli *SLDAPClient) Authenticate(baseDN string, objClass string, uidAttr string, uname string, passwd string, filter string, fields []string, queryScope int) (*ldap.Entry, error) {
	attrMap := make(map[string]string)
	attrMap[uidAttr] = uname
	var retEntry *ldap.Entry
	total, err := cli.Search(
		baseDN, objClass, attrMap, filter, fields, queryScope,
		2, 2,
		func(offset uint32, entry *ldap.Entry) error {
			retEntry = entry
			return nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "Search")
	}
	if total == 0 {
		return nil, ErrUserNotFound
	}
	if total > 1 {
		return nil, ErrUserDuplicate
	}
	defer cli.bind()
	err = cli.conn.Bind(retEntry.DN, passwd)
	if err != nil {
		return nil, ErrUserBadCredential
	}
	return retEntry, nil
}

const (
	StopSearch = errors.Error("stop dap search")
)

// support pagination
// reference: https://zerokspot.com/weblog/2018/03/07/paging-in-gopkg-ldap/
func (cli *SLDAPClient) Search(
	base string, objClass string,
	condition map[string]string,
	filter string,
	fields []string,
	queryScope int,
	pageSize uint32, limit uint32,
	entryFunc func(offset uint32, entry *ldap.Entry) error,
) (uint32, error) {
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
		if isBinaryAttr(k) {
			v = toBinarySearchString(v)
		}
		searches.WriteString(v)
		searches.WriteString(")")
	}
	if len(filter) > 0 && strings.HasPrefix(filter, "(") && strings.HasSuffix(filter, ")") {
		searches.WriteString(filter)
	}
	searchStr := fmt.Sprintf("(&%s)", searches.String())

	if len(base) == 0 {
		base = cli.baseDN
	}

	if queryScope != ldap.ScopeWholeSubtree && queryScope != ldap.ScopeSingleLevel && queryScope != ldap.ScopeBaseObject {
		queryScope = ldap.ScopeWholeSubtree
	}

	log.Debugf("ldapSearch: %s", searchStr)

	offset := uint32(0)
	paging := ldap.NewControlPaging(pageSize)
	for {
		searchRequest := ldap.NewSearchRequest(
			base, // The base dn to search
			queryScope, ldap.NeverDerefAliases, 0, 0, false,
			searchStr,
			fields, // A list attributes to retrieve
			[]ldap.Control{paging},
		)
		searchResult, err := cli.conn.Search(searchRequest)
		if err != nil {
			return offset, errors.Wrap(err, "Seearch")
		}
		pageTotal := len(searchResult.Entries)
		for i := 0; i < pageTotal; i++ {
			entry := searchResult.Entries[i]
			err := entryFunc(offset, entry)
			if err != nil && errors.Cause(err) != StopSearch {
				return offset, errors.Wrapf(err, "process entry fail at %d-%d-%d", pageTotal, i, offset)
			}
			offset += 1
			if (err != nil && errors.Cause(err) == StopSearch) || (limit > 0 && offset >= limit) {
				// stop, offset exceeds limit or receive StopSearch error
				return offset, nil
			}
		}

		resultCtrl := ldap.FindControl(searchResult.Controls, paging.GetControlType())
		if resultCtrl == nil {
			break
		}
		if pagingCtrl, ok := resultCtrl.(*ldap.ControlPaging); ok {
			if len(pagingCtrl.Cookie) == 0 {
				break
			}
			paging.SetCookie(pagingCtrl.Cookie)
		}
	}

	return offset, nil
}

func isBinaryAttr(attrName string) bool {
	for _, attr := range binaryAttributes {
		if strings.EqualFold(attr, attrName) {
			return true
		}
	}
	return false
}

func toBinary(val string) string {
	ret, err := hex.DecodeString(val)
	if err != nil {
		return val
	} else {
		return string(ret)
	}
}

func toBinarySearchString(val string) string {
	ret := strings.Builder{}
	for i := 0; i < len(val); i += 2 {
		ret.WriteString(`\`)
		ret.WriteString(val[i : i+2])
	}
	return ret.String()
}

func toHex(val string) string {
	return hex.EncodeToString([]byte(val))
}

func GetAttributeValue(e *ldap.Entry, key string) string {
	val := e.GetAttributeValue(key)
	if isBinaryAttr(key) {
		val = toHex(val)
	}
	return val
}

func GetAttributeValues(e *ldap.Entry, key string) []string {
	vals := e.GetAttributeValues(key)
	if isBinaryAttr(key) {
		for i := range vals {
			vals[i] = toHex(vals[i])
		}
	}
	return vals
}
