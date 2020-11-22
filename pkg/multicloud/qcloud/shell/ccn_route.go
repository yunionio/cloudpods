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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CcnRouteListOption struct {
		CCNID string
	}
	shellutils.R(&CcnRouteListOption{}, "ccn-route-list", "List cloud connect network route", func(cli *qcloud.SRegion, args *CcnRouteListOption) error {
		routes, err := cli.GetAllCcnRouteSets(args.CCNID)
		if err != nil {
			return err
		}
		printList(routes, len(routes), 0, len(routes), []string{})
		return nil
	})

	type CcnRouteEnableOption struct {
		CCNID   string
		ROUTEID string
	}
	shellutils.R(&CcnRouteEnableOption{}, "ccn-route-enable", "enable cloud connect network route", func(cli *qcloud.SRegion, args *CcnRouteEnableOption) error {
		err := cli.EnableCcnRoutes(args.CCNID, []string{args.ROUTEID})
		if err != nil {
			return err
		}
		return nil
	})

	type CcnRouteDisableOption struct {
		CCNID   string
		ROUTEID string
	}
	shellutils.R(&CcnRouteDisableOption{}, "ccn-route-disable", "disable cloud connect network route", func(cli *qcloud.SRegion, args *CcnRouteDisableOption) error {
		err := cli.DisableCcnRoutes(args.CCNID, []string{args.ROUTEID})
		if err != nil {
			return err
		}
		return nil
	})
}
