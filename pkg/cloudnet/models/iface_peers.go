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
	"fmt"
	"net"
	"strings"

	"yunion.io/x/log"
	yerrors "yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SIfacePeer struct {
	db.SStandaloneResourceBase

	RouterId string
	IfaceId  string

	PeerIfaceId  string
	PeerRouterId string

	PublicKey           string
	AllowedIPs          string
	Endpoint            string
	PersistentKeepalive int
}

type SIfacePeerManager struct {
	db.SStandaloneResourceBaseManager
}

var IfacePeerManager *SIfacePeerManager

func init() {
	IfacePeerManager = &SIfacePeerManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SIfacePeer{},
			"ifacepeers_tbl",
			"ifacepeer",
			"ifacepeers",
		),
	}
	IfacePeerManager.SetVirtualObject(IfacePeerManager)
}

func (ifacePeer *SIfacePeer) subnetsStrList() []string {
	return strings.Split(ifacePeer.AllowedIPs, ",")
}

func (ifacePeer *SIfacePeer) subnetsParsed() Subnets {
	parts := ifacePeer.subnetsStrList()
	r := make([]*netutils.IPV4Prefix, 0, len(parts))
	for _, part := range parts {
		p, err := netutils.NewIPV4Prefix(part)
		if err != nil {
			log.Errorf("%s: invalid subnet sneaked in: %s", ifacePeer.Id, part)
			return nil
		}
		r = append(r, &p)
	}
	return Subnets(r)
}

func (man *SIfacePeerManager) removeByPeerIface(ctx context.Context, userCred mcclient.TokenCredential, iface *SIface) error {
	peers := []SIfacePeer{}
	q := man.Query().Equals("peer_iface_id", iface.Id)
	if err := db.FetchModelObjects(IfacePeerManager, q, &peers); err != nil {
		return err
	}
	var errs []error
	for j := range peers {
		if err := peers[j].Delete(ctx, userCred); err != nil {
			errs = append(errs, err)
		}
	}
	return yerrors.NewAggregate(errs)
}

func (man *SIfacePeerManager) removeByIface(ctx context.Context, userCred mcclient.TokenCredential, iface *SIface) error {
	peers := []SIfacePeer{}
	q := man.Query().Equals("iface_id", iface.Id)
	if err := db.FetchModelObjects(IfacePeerManager, q, &peers); err != nil {
		return err
	}
	var errs []error
	for j := range peers {
		if err := peers[j].Delete(ctx, userCred); err != nil {
			errs = append(errs, err)
		}
	}
	return yerrors.NewAggregate(errs)
}

func (man *SIfacePeerManager) getByFilter(filter map[string]string) ([]SIfacePeer, error) {
	ifacePeers := []SIfacePeer{}
	q := man.Query()
	for key, val := range filter {
		q = q.Equals(key, val)
	}
	if err := db.FetchModelObjects(IfacePeerManager, q, &ifacePeers); err != nil {
		return nil, err
	}
	return ifacePeers, nil
}

func (man *SIfacePeerManager) getOneByFilter(filter map[string]string) (*SIfacePeer, error) {
	ifacePeers, err := man.getByFilter(filter)
	if err != nil {
		return nil, err
	}
	if len(ifacePeers) == 0 {
		return nil, errNotFound(fmt.Errorf("cannot find iface peer: %#v", filter))
	}
	if len(ifacePeers) > 1 {
		return nil, errMoreThanOne(fmt.Errorf("found more than 1 iface peers: %#v", filter))
	}
	return &ifacePeers[0], nil
}

func (man *SIfacePeerManager) getByIface(iface *SIface) ([]SIfacePeer, error) {
	filter := map[string]string{
		"router_id": iface.RouterId,
		"iface_id":  iface.Id,
	}
	ifacePeers, err := man.getByFilter(filter)
	if err != nil {
		return nil, err
	}
	return ifacePeers, nil
}

func (man *SIfacePeerManager) getByIfacePublicKey(iface *SIface, pubkey string) (*SIfacePeer, error) {
	filter := map[string]string{
		"router_id":  iface.RouterId,
		"iface_id":   iface.Id,
		"public_key": pubkey,
	}
	ifacePeer, err := man.getOneByFilter(filter)
	if err != nil {
		return nil, err
	}
	return ifacePeer, nil
}

func (man *SIfacePeerManager) checkAllowedIPs(iface *SIface, oldPeer *SIfacePeer, allowedNets Subnets) error {
	ifacePeers, err := man.getByIface(iface)
	if err != nil {
		return err
	}
	for i := range ifacePeers {
		ifacePeer := &ifacePeers[i]
		if oldPeer != nil && oldPeer.Id == ifacePeer.Id {
			continue
		}
		existingNets := ifacePeer.subnetsParsed()
		if _, net := existingNets.ContainsAnyEx(allowedNets); net != nil {
			return fmt.Errorf("subnet %s is already occupied by peer %s(%s)",
				net, ifacePeer.Name, ifacePeer.Id)
		}
	}
	return nil
}

func (man *SIfacePeerManager) updateEndpointIPByPeerRouter(ctx context.Context, endpointIP string, router *SRouter) error {
	filter := map[string]string{
		"peer_router_id": router.Id,
	}
	ifacePeers, err := man.getByFilter(filter)
	if err != nil {
		return err
	}

	var errs []error
	for i := range ifacePeers {
		ifacePeer := &ifacePeers[i]
		host, port, err := net.SplitHostPort(ifacePeer.Endpoint)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if host == endpointIP {
			continue
		}
		_, err = db.Update(ifacePeer, func() error {
			endpoint := net.JoinHostPort(endpointIP, port)
			ifacePeer.Endpoint = endpoint
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return yerrors.NewAggregate(errs)
}
