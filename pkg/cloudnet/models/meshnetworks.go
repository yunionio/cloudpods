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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	yerrors "yunion.io/x/pkg/util/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SMeshNetwork struct {
	db.SStandaloneResourceBase
}

type SMeshNetworkManager struct {
	db.SStandaloneResourceBaseManager
}

var MeshNetworkManager *SMeshNetworkManager

func init() {
	MeshNetworkManager = &SMeshNetworkManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SMeshNetwork{},
			"meshnetworks_tbl",
			"meshnetwork",
			"meshnetworks",
		),
	}
	MeshNetworkManager.SetVirtualObject(MeshNetworkManager)
}

func (mn *SMeshNetwork) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	var errs []error
	if err := MeshNetworkMemberManager.removeByMeshNetwork(ctx, userCred, mn); err != nil {
		errs = append(errs, err)
	}
	if err := IfaceManager.removeByMeshNetwork(ctx, userCred, mn); err != nil {
		errs = append(errs, err)
	}
	if err := mn.SStandaloneResourceBase.CustomizeDelete(ctx, userCred, query, data); err != nil {
		errs = append(errs, err)
	}
	return yerrors.NewAggregate(errs)
}

func (mn *SMeshNetwork) AllowPerformRealize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, mn, "realize")
}

func (mn *SMeshNetwork) PerformRealize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	members, err := MeshNetworkMemberManager.getMemebersByMeshNetwork(ctx, userCred, mn)
	if err != nil {
		return nil, httperrors.NewBadRequestError("fetch members: %v", err)
	}
	var errs []error
	for i := range members {
		member := &members[i]
		router, err := member.getRouter()
		if err != nil {
			errs = append(errs, errors.WithMessagef(err, "get router %s", member.RouterId))
			continue
		}
		if err := router.realize(ctx, userCred); err != nil {
			errs = append(errs, errors.WithMessagef(err, "realize router %s", router.Name))
		}
	}
	err = yerrors.NewAggregate(errs)
	if err != nil {
		err = httperrors.NewBadRequestError("some router realization failed: %s", err)
	}
	return nil, err
}

func (mn *SMeshNetwork) addRouter(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter, nets Subnets) error {
	// XXX lock
	members, err := MeshNetworkMemberManager.getMemebersByMeshNetwork(ctx, userCred, mn)
	if err != nil {
		return err
	}
	for i := range members {
		member := &members[i]
		if member.RouterId == router.Id {
			return fmt.Errorf("router %s is already a member of %s",
				router.Name, mn.Name)
		}
		memberSubnets := member.subnetsParsed()
		if _, p := memberSubnets.ContainsAnyEx(nets); p != nil {
			return fmt.Errorf("router %s subnet %s already advertised by member %s(%s)",
				router.Name, p.String(), member.Name, member.Id)
		}
	}
	_, err = MeshNetworkMemberManager.addMember(ctx, userCred, mn, router, nets)
	if err != nil {
		return err
	}
	newIface, err := IfaceManager.addWireguardIface(ctx, userCred, router, mn)
	if err != nil {
		return err
	}
	// XXX allocate an iface and populate iface peers
	var errs []error
	for i := range members {
		member := &members[i]
		memberIface, err := IfaceManager.getByMeshNetworkMember(member)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := memberIface.addOrUpdatePeer(ctx, userCred, newIface, nets, router); err != nil {
			errs = append(errs, err)
		}
		if memberHost, err := RouterManager.getById(member.RouterId); err != nil {
			errs = append(errs, err)
		} else {
			if err := newIface.addOrUpdatePeer(ctx, userCred, memberIface, member.subnetsParsed(), memberHost); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return yerrors.NewAggregate(errs)
}

func (mn *SMeshNetwork) removeRouter(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter) error {
	if err := MeshNetworkMemberManager.removeByMeshNetworkRouter(ctx, userCred, mn, router); err != nil {
		return err
	}
	if err := IfaceManager.removeByMeshNetworkRouter(ctx, userCred, mn, router); err != nil {
		return err
	}
	return nil
}

func (man *SMeshNetworkManager) removeRouter(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter) error {
	err := MeshNetworkMemberManager.removeByRouter(ctx, userCred, router)
	return err
}
