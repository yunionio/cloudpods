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
// PageSizeations under the License.

package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CdnDomainList struct {
		Origin string
	}
	shellutils.R(&CdnDomainList{}, "cdn-domain-list", "List cdn domain", func(cli *aliyun.SRegion, args *CdnDomainList) error {
		domainlist, e := cli.GetClient().DescribeDomainsBySource(args.Origin)
		if e != nil {
			return e
		}
		printList(domainlist.DomainsData, len(domainlist.DomainsData), 1, len(domainlist.DomainsData), []string{})
		return nil
	})

	type CdnDomainShow struct {
		DOMAIN string
	}
	shellutils.R(&CdnDomainShow{}, "cdn-domain-show", "show cdn domain", func(cli *aliyun.SRegion, args *CdnDomainShow) error {
		domainlist, e := cli.GetClient().DescribeUserDomains(args.DOMAIN)
		if e != nil {
			return e
		}
		printList(domainlist.PageData, len(domainlist.PageData), 1, len(domainlist.PageData), []string{})
		return nil
	})
}
