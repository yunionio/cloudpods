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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type wireIdChangeArgs struct {
	oldWire *SWire
	newWire *SWire
}

type wireIdChangeHandler interface {
	handleWireIdChange(ctx context.Context, args *wireIdChangeArgs) error
}

func (manager *SHostwireManager) handleWireIdChange(ctx context.Context, args *wireIdChangeArgs) error {
	hws := make([]SHostwire, 0, 8)
	err := db.FetchModelObjects(manager, manager.Query().Equals("wire_id", args.oldWire.Id), &hws)
	if err != nil {
		return err
	}
	for i := range hws {
		hw := &hws[i]
		_, err := db.Update(hw, func() error {
			hw.WireId = args.newWire.Id
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "unable to update hostwire host %q wire %q", hw.HostId, hw.WireId)
		}

	}
	return nil
}

func (manager *SLoadbalancerClusterManager) handleWireIdChange(ctx context.Context, args *wireIdChangeArgs) error {
	lcs := make([]SLoadbalancerCluster, 0, 8)
	err := db.FetchModelObjects(manager, manager.Query().Equals("wire_id", args.oldWire.Id), &lcs)
	if err != nil {
		return err
	}
	for i := range lcs {
		lc := &lcs[i]
		_, err := db.Update(lc, func() error {
			lc.WireId = args.newWire.Id
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "unable to update loadbalancercluster %q", lc.GetId())
		}
	}
	return nil
}

func (manager *SNetworkManager) handleWireIdChange(ctx context.Context, args *wireIdChangeArgs) error {
	ns := make([]SNetwork, 0, 8)
	err := db.FetchModelObjects(manager, manager.Query().Equals("wire_id", args.oldWire.Id), &ns)
	if err != nil {
		return err
	}
	for i := range ns {
		n := &ns[i]
		_, err := db.Update(n, func() error {
			n.WireId = args.newWire.Id
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "unable to update network %q", n.GetId())
		}
	}
	return nil
}
