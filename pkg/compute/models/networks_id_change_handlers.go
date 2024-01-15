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

package models

import (
	"context"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type networkIdChangeArgs struct {
	action   string
	oldNet   *SNetwork
	newNet   *SNetwork
	userCred mcclient.TokenCredential
}

type networkIdChangeHandler interface {
	handleNetworkIdChange(ctx context.Context, args *networkIdChangeArgs) error
}

func (manager *SGuestnetworkManager) handleNetworkIdChange(ctx context.Context, args *networkIdChangeArgs) error {
	gns := make([]SGuestnetwork, 0)
	err := db.FetchModelObjects(manager, manager.Query().Equals("network_id", args.oldNet.Id), &gns)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, args.oldNet, args.action, err.Error(), args.userCred, false)
		return err
	}
	for _, gn := range gns {
		addr, _ := netutils.NewIPV4Addr(gn.IpAddr)
		if args.newNet.IsAddressInRange(addr) {
			_, err = db.Update(&gn, func() error {
				gn.NetworkId = args.newNet.Id
				return nil
			})
			if err != nil {
				log.Errorln(err)
			}
		}
	}
	return nil
}

func (manager *SHostnetworkManager) handleNetworkIdChange(ctx context.Context, args *networkIdChangeArgs) error {
	hns := make([]SHostnetwork, 0)
	err := db.FetchModelObjects(manager, manager.Query().Equals("network_id", args.oldNet.Id), &hns)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, args.oldNet, args.action, err.Error(), args.userCred, false)
		return err
	}
	for _, hn := range hns {
		addr, _ := netutils.NewIPV4Addr(hn.IpAddr)
		if args.newNet.IsAddressInRange(addr) {
			_, err = db.Update(&hn, func() error {
				hn.NetworkId = args.newNet.Id
				return nil
			})
			if err != nil {
				log.Errorln(err)
			}
		}
	}
	return nil
}

func (manager *SReservedipManager) handleNetworkIdChange(ctx context.Context, args *networkIdChangeArgs) error {
	ris := make([]SReservedip, 0)
	err := db.FetchModelObjects(manager, manager.Query().Equals("network_id", args.oldNet.Id), &ris)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, args.oldNet, args.action, err.Error(), args.userCred, false)
		return err
	}
	for _, ri := range ris {
		if len(ri.IpAddr) == 0 {
			continue
		}
		addr, _ := netutils.NewIPV4Addr(ri.IpAddr)
		if args.newNet.IsAddressInRange(addr) {
			_, err = db.Update(&ri, func() error {
				ri.NetworkId = args.newNet.Id
				return nil
			})
			if err != nil {
				log.Errorln(err)
			}
		}
	}
	return nil
}

func (manager *SGroupnetworkManager) handleNetworkIdChange(ctx context.Context, args *networkIdChangeArgs) error {
	gns := make([]SGroupnetwork, 0)
	err := db.FetchModelObjects(manager, manager.Query().Equals("network_id", args.oldNet.Id), &gns)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, args.oldNet, args.action, err.Error(), args.userCred, false)
		return err
	}
	for _, gn := range gns {
		if len(gn.IpAddr) == 0 {
			continue
		}
		addr, _ := netutils.NewIPV4Addr(gn.IpAddr)
		if args.newNet.IsAddressInRange(addr) {
			_, err = db.Update(&gn, func() error {
				gn.NetworkId = args.newNet.Id
				return nil
			})
			if err != nil {
				log.Errorln(err)
			}
		}
	}
	return nil
}

func (manager *SLoadbalancernetworkManager) handleNetworkIdChange(ctx context.Context, args *networkIdChangeArgs) error {
	lbns := make([]SLoadbalancerNetwork, 0)
	err := db.FetchModelObjects(manager, manager.Query().Equals("network_id", args.oldNet.Id), &lbns)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, args.oldNet, args.action, err.Error(), args.userCred, false)
		return err
	}
	for _, lbn := range lbns {
		addr, _ := netutils.NewIPV4Addr(lbn.IpAddr)
		if args.newNet.IsAddressInRange(addr) {
			_, err = db.Update(&lbn, func() error {
				lbn.NetworkId = args.newNet.Id
				return nil
			})
			if err != nil {
				log.Errorln(err)
			}
		}
	}
	return nil
}

func (manager *SLoadbalancerManager) handleNetworkIdChange(ctx context.Context, args *networkIdChangeArgs) error {
	lbs := make([]SLoadbalancer, 0)
	err := db.FetchModelObjects(manager, manager.Query().Equals("network_id", args.oldNet.Id), &lbs)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, args.oldNet, args.action, err.Error(), args.userCred, false)
		return err
	}
	for _, lb := range lbs {
		addr, _ := netutils.NewIPV4Addr(lb.Address)
		if args.newNet.IsAddressInRange(addr) {
			_, err = db.Update(&lb, func() error {
				lb.NetworkId = args.newNet.Id
				return nil
			})
			if err != nil {
				log.Errorln(err)
			}
		}
	}
	return nil
}
