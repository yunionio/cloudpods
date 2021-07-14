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
	type CenRouteListOptions struct {
		CENID               string
		CHILDINSTANCEID     string
		CHILDINSTANCEREGION string
		CHILDINSTANCETYPE   string
	}
	shellutils.R(&CenRouteListOptions{}, "cen-route-list", "List cloud enterprise network route", func(cli *aliyun.SRegion, args *CenRouteListOptions) error {
		routes, e := cli.GetClient().GetAllCenChildInstanceRouteEntries(args.CENID, args.CHILDINSTANCEID, args.CHILDINSTANCEREGION, args.CHILDINSTANCETYPE)
		if e != nil {
			return e
		}
		printList(routes, len(routes), 1, len(routes), []string{})
		return nil
	})

	type PublishCenRouteOptions struct {
		CENID               string
		CHILDINSTANCEID     string
		CHILDINSTANCEREGION string
		CHILDINSTANCETYPE   string
		ROUTETABLEID        string
		CIDR                string
	}
	shellutils.R(&PublishCenRouteOptions{}, "cen-route-publish", "publish cloud enterprise network route", func(cli *aliyun.SRegion, args *PublishCenRouteOptions) error {
		e := cli.GetClient().PublishRouteEntries(args.CENID, args.CHILDINSTANCEID, args.ROUTETABLEID, args.CHILDINSTANCEREGION, args.CHILDINSTANCETYPE, args.CIDR)
		if e != nil {
			return e
		}
		return nil
	})

	type WithDrawCenRouteOptions struct {
		CENID               string
		CHILDINSTANCEID     string
		CHILDINSTANCEREGION string
		CHILDINSTANCETYPE   string
		ROUTETABLEID        string
		CIDR                string
	}
	shellutils.R(&PublishCenRouteOptions{}, "cen-route-withdraw", "withdraw cloud enterprise network route", func(cli *aliyun.SRegion, args *PublishCenRouteOptions) error {
		e := cli.GetClient().WithdrawPublishedRouteEntries(args.CENID, args.CHILDINSTANCEID, args.ROUTETABLEID, args.CHILDINSTANCEREGION, args.CHILDINSTANCETYPE, args.CIDR)
		if e != nil {
			return e
		}
		return nil
	})
}
