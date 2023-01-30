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

package compute

import (
	"crypto/md5"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type LoadbalancerAclListInput struct {
	apis.SharableVirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput

	//
	Fingerprint string `json:"fingerprint"`
}

type LoadbalancerAclDetails struct {
	apis.SharableVirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	SLoadbalancerAcl

	LbListenerCount int `json:"lb_listener_count"`
}

type LoadbalancerAclResourceInfo struct {
	// 负载均衡ACL名称
	Acl string `json:"acl"`
}

type LoadbalancerAclResourceInput struct {
	// ACL名称或ID
	AclId string `json:"acl_id"`

	// swagger:ignore
	// Deprecated
	Acl string `json:"acl" yunion-deprecated-by:"acl_id"`
}

type LoadbalancerAclFilterListInput struct {
	LoadbalancerAclResourceInput

	// 以ACL名称排序
	OrderByAcl string `json:"order_by_acl"`
}

type SAclEntry struct {
	Cidr    string
	Comment string
}

type SAclEntries []SAclEntry

func (self SAclEntries) GetFingerprint() string {
	cidrs := []string{}
	for _, acl := range self {
		cidrs = append(cidrs, acl.Cidr)
	}
	sort.Strings(cidrs)
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(cidrs, ""))))
}

func (self SAclEntries) String() string {
	return jsonutils.Marshal(self).String()
}

func (self SAclEntries) IsZero() bool {
	return len(self) == 0
}

func (aclEntry *SAclEntry) Validate() error {
	if strings.Index(aclEntry.Cidr, "/") > 0 {
		_, ipNet, err := net.ParseCIDR(aclEntry.Cidr)
		if err != nil {
			return err
		}
		// normalize from 192.168.1.3/24 to 192.168.1.0/24
		aclEntry.Cidr = ipNet.String()
	} else {
		ip := net.ParseIP(aclEntry.Cidr).To4()
		if ip == nil {
			return httperrors.NewInputParameterError("invalid addr %s", aclEntry.Cidr)
		}
	}
	if commentLimit := 128; len(aclEntry.Comment) > commentLimit {
		return httperrors.NewInputParameterError("comment too long (%d>=%d)",
			len(aclEntry.Comment), commentLimit)
	}
	for _, r := range aclEntry.Comment {
		if !unicode.IsPrint(r) {
			return httperrors.NewInputParameterError("comment contains non-printable char: %v", r)
		}
	}
	return nil
}

type LoadbalancerAclCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	AclEntries SAclEntries `json:"acl_entries"`
	// swagger: ignore
	Fingerprint string
}

func (self *LoadbalancerAclCreateInput) Validate() error {
	if len(self.AclEntries) == 0 {
		return httperrors.NewMissingParameterError("acl_entries")
	}
	found := map[string]bool{}
	for _, acl := range self.AclEntries {
		err := acl.Validate()
		if err != nil {
			return err
		}
		if _, ok := found[acl.Cidr]; ok {
			// error so that the user has a chance to deal with comments
			return httperrors.NewInputParameterError("acl cidr duplicate %s", acl.Cidr)
		}
		found[acl.Cidr] = true
	}
	self.Fingerprint = self.AclEntries.GetFingerprint()
	return nil
}

type LoadbalancerAclUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput

	AclEntries SAclEntries `json:"acl_entries"`
	// swagger: ignore
	Fingerprint string
}

func (self *LoadbalancerAclUpdateInput) Validate() error {
	if len(self.AclEntries) > 0 {
		found := map[string]bool{}
		for _, acl := range self.AclEntries {
			err := acl.Validate()
			if err != nil {
				return err
			}
			if _, ok := found[acl.Cidr]; ok {
				// error so that the user has a chance to deal with comments
				return httperrors.NewInputParameterError("acl cidr duplicate %s", acl.Cidr)
			}
			found[acl.Cidr] = true
		}
		self.Fingerprint = self.AclEntries.GetFingerprint()
	}
	return nil
}

type LoadbalancerAclPatchInput struct {
	Adds []SAclEntry `json:"adds"`
	Dels []SAclEntry `json:"dels"`
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SAclEntries{}), func() gotypes.ISerializable {
		return &SAclEntries{}
	})
}
