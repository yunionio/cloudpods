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
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	yerrors "yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	cnutils "yunion.io/x/onecloud/pkg/cloudnet/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var regexpIfname = regexp.MustCompile(`[A-Za-z][A-Za-z0-9]{0,14}`)

type SIface struct {
	db.SStandaloneResourceBase

	RouterId  string `length:"32" nullable:"false"`
	NetworkId string `length:"32" nullable:"false"`

	Ifname string `length:"32" nullable:"false"`

	PrivateKey string
	PublicKey  string
	ListenPort int `nullable:"false"`

	IsSystem bool `nullable:"false"`
}

type SIfaceManager struct {
	db.SStandaloneResourceBaseManager
}

var IfaceManager *SIfaceManager

func init() {
	IfaceManager = &SIfaceManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SIface{},
			"ifaces_tbl",
			"iface",
			"ifaces",
		),
	}
	IfaceManager.SetVirtualObject(IfaceManager)
}

func (man *SIfaceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// router existence
	// PrivateKey validation , or generation
	// ListenPort validation, uniqueness
	// ListenPort generation
	return nil, errors.New("manually adding interface is currently not supported")
}

func (man *SIfaceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "router", ModelKeyword: "router", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (iface *SIface) ValidateUpdateCondition(ctx context.Context) error {
	// same as create but no generation
	// if privatekey updated
	// 	update peers whose peerifaceid == self.id
	return nil
}

func (iface *SIface) ValidateDeleteCondition(ctx context.Context) error {
	// if networkid != "" {
	// 	return errors.New("part of network, remove it by remove network memeber")
	// }
	return nil
}

func (iface *SIface) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// remove ifacepeer whose peerifaceId is self.id
	return nil
}

func (iface *SIface) addOrUpdatePeer(ctx context.Context, userCred mcclient.TokenCredential,
	peerIface *SIface, allowedNets Subnets, peerRouter *SRouter) error {
	// XXX lock
	endpoint := ""
	endpointIP := peerRouter.endpointIP()
	if endpointIP != "" && peerIface.ListenPort > 0 {
		endpoint = fmt.Sprintf("%s:%d", endpointIP, peerIface.ListenPort)
	}
	persistentKeepalive := 0
	if endpointIP != "" {
		// persistent keepalive from private addr to exit addr
		router, err := iface.getRouter()
		if err != nil {
			return errors.WithMessagef(err, "get iface router %s", iface.RouterId)
		}
		myIP := router.endpointIP()
		if myIP != "" {
			myIPAddr, err := netutils.NewIPV4Addr(myIP)
			if err != nil {
				return err
			}
			if netutils.IsPrivate(myIPAddr) {
				peerIPAddr, err := netutils.NewIPV4Addr(endpointIP)
				if err != nil {
					return err
				}
				if netutils.IsExitAddress(peerIPAddr) {
					persistentKeepalive = 10
				}
			}
		}
	}
	ifacePeer, err := IfacePeerManager.getByIfacePublicKey(iface, peerIface.PublicKey)
	if err != nil && !IsNotFound(err) {
		return err
	}
	if err := IfacePeerManager.checkAllowedIPs(iface, ifacePeer, allowedNets); err != nil {
		return err
	}
	if ifacePeer == nil {
		ifacePeer := &SIfacePeer{
			RouterId: iface.RouterId,
			IfaceId:  iface.Id,

			PeerIfaceId:         peerIface.Id,
			PeerRouterId:        peerIface.RouterId,
			PublicKey:           peerIface.PublicKey,
			AllowedIPs:          allowedNets.String(),
			Endpoint:            endpoint,
			PersistentKeepalive: persistentKeepalive,
		}
		ifacePeer.Name = fmt.Sprintf("%s-%s", iface.Name, peerIface.Name)
		err := IfacePeerManager.TableSpec().Insert(ifacePeer)
		return err
	}
	_, err = db.Update(ifacePeer, func() error {
		ifacePeer.PeerIfaceId = peerIface.Id
		ifacePeer.PeerRouterId = peerIface.RouterId
		ifacePeer.Endpoint = endpoint
		ifacePeer.AllowedIPs = allowedNets.String()
		ifacePeer.PersistentKeepalive = persistentKeepalive
		return nil
	})
	return err
}

func (iface *SIface) clearPeers(ctx context.Context, userCred mcclient.TokenCredential) error {
	return IfacePeerManager.removeByIface(ctx, userCred, iface)
}

func (iface *SIface) clearPeerRefs(ctx context.Context, userCred mcclient.TokenCredential) error {
	return IfacePeerManager.removeByPeerIface(ctx, userCred, iface)
}

func (iface *SIface) clearRoutes(ctx context.Context, userCred mcclient.TokenCredential) error {
	return RouteManager.removeByIface(ctx, userCred, iface)
}

func (iface *SIface) remove(ctx context.Context, userCred mcclient.TokenCredential) error {
	var errs []error
	if err := iface.clearRoutes(ctx, userCred); err != nil {
		errs = append(errs, err)
	}
	if err := iface.clearPeers(ctx, userCred); err != nil {
		errs = append(errs, err)
	}
	if err := iface.clearPeerRefs(ctx, userCred); err != nil {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		if err := iface.Delete(ctx, userCred); err != nil {
			errs = append(errs, err)
		}
	}
	return yerrors.NewAggregate(errs)
}

func (iface *SIface) isTypeWireguard() bool {
	r := iface.ListenPort > 0 && iface.PrivateKey != "" && iface.PublicKey != ""
	return r
}

func (iface *SIface) getRouter() (*SRouter, error) {
	obj, err := db.FetchById(RouterManager, iface.RouterId)
	if err != nil {
		return nil, err
	}
	router := obj.(*SRouter)
	return router, nil
}

func (man *SIfaceManager) removeByFilter(ctx context.Context, userCred mcclient.TokenCredential, filter map[string]string) error {
	ifaces, err := man.getByFilter(filter)
	if err != nil {
		return err
	}
	var errs []error
	for i := range ifaces {
		iface := &ifaces[i]
		if err := iface.remove(ctx, userCred); err != nil {
			errs = append(errs, err)
		}
	}
	return yerrors.NewAggregate(errs)
}

func (man *SIfaceManager) removeByRouter(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter) error {
	err := man.removeByFilter(ctx, userCred, map[string]string{
		"router_id": router.Id,
	})
	return err
}

func (man *SIfaceManager) removeByMeshNetwork(ctx context.Context, userCred mcclient.TokenCredential, mn *SMeshNetwork) error {
	err := man.removeByFilter(ctx, userCred, map[string]string{
		"network_id": mn.Id,
	})
	return err
}

func (man *SIfaceManager) removeByMeshNetworkRouter(ctx context.Context, userCred mcclient.TokenCredential, mn *SMeshNetwork, router *SRouter) error {
	err := man.removeByFilter(ctx, userCred, map[string]string{
		"network_id": mn.Id,
		"router_id":  router.Id,
	})
	return err
}
func (man *SIfaceManager) getByRouter(router *SRouter) ([]SIface, error) {
	return man.getByFilter(map[string]string{
		"router_id": router.Id,
	})
}

func (man *SIfaceManager) getByRouterIfname(router *SRouter, ifname string) (*SIface, error) {
	return man.getOneByFilter(map[string]string{
		"router_id": router.Id,
		"ifname":    ifname,
	})
}

func (man *SIfaceManager) getByFilter(filter map[string]string) ([]SIface, error) {
	ifaces := []SIface{}
	q := man.Query()
	for key, val := range filter {
		q = q.Equals(key, val)
	}
	if err := db.FetchModelObjects(IfaceManager, q, &ifaces); err != nil {
		return nil, err
	}
	return ifaces, nil
}

func (man *SIfaceManager) getOneByFilter(filter map[string]string) (*SIface, error) {
	ifaces, err := man.getByFilter(filter)
	if err != nil {
		return nil, err
	}
	if len(ifaces) == 0 {
		return nil, fmt.Errorf("cannot find iface for condition: %#v", filter)
	}
	if len(ifaces) > 1 {
		return nil, fmt.Errorf("found more than 1 ifaces for condition: %#v", filter)
	}
	return &ifaces[0], nil
}

func (man *SIfaceManager) checkExistenceByFilter(filter map[string]string) error {
	ifaces, err := man.getByFilter(filter)
	if err != nil {
		return err
	}
	if len(ifaces) > 0 {
		return fmt.Errorf("iface exist: %s(%s)", ifaces[0].Name, ifaces[0].Id)
	}
	return nil
}

func (man *SIfaceManager) getByRouterNetwork(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter, mn *SMeshNetwork) ([]SIface, error) {
	ifaces, err := man.getByFilter(map[string]string{
		"router_id":  router.Id,
		"network_id": mn.Id,
	})
	return ifaces, err
}

func (man *SIfaceManager) getOneByRouterNetwork(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter, mn *SMeshNetwork) (*SIface, error) {
	iface, err := man.getOneByFilter(map[string]string{
		"router_id":  router.Id,
		"network_id": mn.Id,
	})
	return iface, err
}

func (man *SIfaceManager) getByMeshNetworkMember(member *SMeshNetworkMember) (*SIface, error) {
	iface, err := man.getOneByFilter(map[string]string{
		"router_id":  member.RouterId,
		"network_id": member.MeshNetworkId,
	})
	return iface, err
}

func (man *SIfaceManager) getNextName(filter map[string]string, base string) (string, error) {
	ifaces, err := man.getByFilter(filter)
	if err != nil {
		return "", err
	}
	occupied := map[int]struct{}{}
	for i := range ifaces {
		iface := &ifaces[i]
		if strings.HasPrefix(iface.Ifname, base) {
			istr := iface.Ifname[len(base):]
			i, err := strconv.ParseUint(istr, 10, 16)
			if err != nil {
				continue
			}
			occupied[int(i)] = struct{}{}
		}
	}
	for i := 0; i < 65536; i++ {
		if _, ok := occupied[i]; !ok {
			return fmt.Sprintf("%s%d", base, i), nil
		}
	}
	return "", fmt.Errorf("all names occupied")
}

func (man *SIfaceManager) addWireguardIface(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter, mn *SMeshNetwork) (*SIface, error) {
	k := cnutils.MustNewKey()
	port := router.mustFindFreePort(ctx)
	iface := &SIface{
		RouterId:   router.Id,
		PrivateKey: k.String(),
		PublicKey:  k.PublicKey().String(),
		ListenPort: port,
	}
	iface.IsSystem = true

	{ // ifname
		if name, err := man.getNextName(map[string]string{
			"router_id": router.Id,
		}, "wg"); err != nil {
			return nil, err
		} else {
			iface.Ifname = name
		}
	}

	{ // obj name
		name := router.Name + "-"
		if mn != nil {
			iface.NetworkId = mn.Id
			name += mn.Name + "-"
		}
		name += fmt.Sprintf("%d", port)
		iface.Name = name
	}

	iface.SetModelManager(man, iface)
	err := man.TableSpec().Insert(iface)
	if err != nil {
		return nil, err
	}
	if err := RuleManager.addWireguardIfaceRules(ctx, userCred, iface); err != nil {
		iface.Delete(ctx, userCred)
		return nil, err
	}
	return iface, nil
}

func (man *SIfaceManager) addIface(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter, ifname string) (*SIface, error) {
	if err := man.checkExistenceByFilter(map[string]string{
		"router_id": router.Id,
		"ifname":    ifname,
	}); err != nil {
		return nil, err
	}
	iface := &SIface{
		RouterId: router.Id,
		Ifname:   ifname,
	}
	iface.SetModelManager(man, iface)
	err := man.TableSpec().Insert(iface)
	if err != nil {
		return nil, err
	}
	return iface, nil
}
