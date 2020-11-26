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

package options

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
)

type AclEntry struct {
	Cidr    string
	Comment string
}

type AclEntries []*AclEntry

func NewAclEntry(s string) *AclEntry {
	tu := strings.SplitN(s, "#", 2)
	cidr := strings.TrimSpace(tu[0])
	comment := ""
	if len(tu) > 1 {
		comment = tu[1]
	}
	aclEntry := &AclEntry{
		Cidr:    cidr,
		Comment: comment,
	}
	return aclEntry
}

func NewAclEntries(ss []string) AclEntries {
	aclEntries := AclEntries{}
	for _, s := range ss {
		aclEntry := NewAclEntry(s)
		aclEntries = append(aclEntries, aclEntry)
	}
	return aclEntries
}

func (entry *AclEntry) String() string {
	cidr := entry.Cidr
	comment := entry.Comment
	if comment != "" {
		comment = " # " + comment
	}
	return fmt.Sprintf("%-15s%s", cidr, comment)
}

func (entries AclEntries) String() string {
	ss := []string{}
	for _, entry := range entries {
		ss = append(ss, entry.String())
	}
	lines := strings.Join(ss, "\n")
	return lines
}

type LoadbalancerAclCreateOptions struct {
	SharableProjectizedResourceBaseCreateInput

	NAME     string
	AclEntry []string `help:"acl entry with cidr and comment separated by #, e.g. 10.9.0.0/16#no comment" json:"-"`
	Manager  string   `json:"manager_id"`
	Region   string   `json:"cloudregion"`
	Listener string   `help:"Huawei listener id"`
}

type LoadbalancerAclGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerAclListOptions struct {
	BaseListOptions
	Cloudregion string
}

type LoadbalancerAclUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	AclEntry []string `help:"acl entry with cidr and comment separated by #, e.g. 10.9.0.0/16#no comment" json:"-"`
}

type LoadbalancerAclDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerAclActionPatchOptions struct {
	ID  string   `json:"-"`
	Add []string `help:"acl entry with cidr and comment separated by #, e.g. 10.9.0.0/16#no comment" json:"-"`
	Del []string `help:"acl entry with cidr and comment separated by #, e.g. 10.9.0.0/16#no comment" json:"-"`
}

type LoadbalancerAclPublicOptions struct {
	SharableResourcePublicBaseOptions

	ID string `json:"-"`
}

type LoadbalancerAclPrivateOptions struct {
	ID string `json:"-"`
}

func (opts *LoadbalancerAclCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}

	sp, err := opts.SharableProjectizedResourceBaseCreateInput.Params()
	if err != nil {
		return nil, err
	}

	params.Update(sp)
	aclEntries := NewAclEntries(opts.AclEntry)
	aclEntriesJson := jsonutils.Marshal(aclEntries)
	params.Set("acl_entries", aclEntriesJson)
	return params, nil
}

func (opts *LoadbalancerAclUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}

	// - when it's nil, we leave it alone without updating
	// - when it's non-nil, we update it as a whole
	if opts.AclEntry != nil {
		aclEntries := NewAclEntries(opts.AclEntry)
		aclEntriesJson := jsonutils.Marshal(aclEntries)
		params.Set("acl_entries", aclEntriesJson)
	}
	return params, nil
}

func (opts *LoadbalancerAclActionPatchOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	m := map[string][]string{
		"adds": opts.Add,
		"dels": opts.Del,
	}
	for k, ss := range m {
		aclEntries := NewAclEntries(ss)
		aclEntriesJson := jsonutils.Marshal(aclEntries)
		params.Set(k, aclEntriesJson)
	}
	return params, nil
}
