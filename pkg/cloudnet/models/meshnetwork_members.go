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
	"strings"

	"yunion.io/x/log"
	yerrors "yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SMeshNetworkMember struct {
	db.SStandaloneResourceBase

	MeshNetworkId    string
	RouterId         string
	AdvertiseSubnets string
}

type SMeshNetworkMemberManager struct {
	db.SStandaloneResourceBaseManager
}

var MeshNetworkMemberManager *SMeshNetworkMemberManager

func init() {
	MeshNetworkMemberManager = &SMeshNetworkMemberManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SMeshNetworkMember{},
			"meshnetwork_members_tbl",
			"meshnetwork_member",
			"meshnetwork_members",
		),
	}
	MeshNetworkMemberManager.SetVirtualObject(MeshNetworkMemberManager)
}

func (member *SMeshNetworkMember) subnetsStrList() []string {
	return strings.Split(member.AdvertiseSubnets, ",")
}

func (member *SMeshNetworkMember) subnetsParsed() Subnets {
	parts := member.subnetsStrList()
	r := make([]*netutils.IPV4Prefix, 0, len(parts))
	for _, part := range parts {
		p, err := netutils.NewIPV4Prefix(part)
		if err != nil {
			log.Errorf("%s: invalid subnet sneaked in: %s", member.Id, part)
			return nil
		}
		r = append(r, &p)
	}
	return Subnets(r)
}

func (member *SMeshNetworkMember) getRouter() (*SRouter, error) {
	obj, err := db.FetchById(RouterManager, member.RouterId)
	if err != nil {
		return nil, err
	}
	router := obj.(*SRouter)
	return router, nil
}

func (man *SMeshNetworkMemberManager) getByFilter(filter map[string]string) ([]SMeshNetworkMember, error) {
	members := []SMeshNetworkMember{}
	q := man.Query()
	for key, val := range filter {
		q = q.Equals(key, val)
	}
	if err := db.FetchModelObjects(man, q, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (man *SMeshNetworkMemberManager) removeByFilter(ctx context.Context, userCred mcclient.TokenCredential, filter map[string]string) error {
	members, err := man.getByFilter(filter)
	if err != nil {
		return err
	}
	var errs []error
	for i := range members {
		if err := members[i].Delete(ctx, userCred); err != nil {
			errs = append(errs, err)
		}
	}
	return yerrors.NewAggregate(errs)
}

func (man *SMeshNetworkMemberManager) removeByRouter(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter) error {
	err := man.removeByFilter(ctx, userCred, map[string]string{
		"router_id": router.Id,
	})
	return err
}

func (man *SMeshNetworkMemberManager) removeByMeshNetwork(ctx context.Context, userCred mcclient.TokenCredential, mn *SMeshNetwork) error {
	err := man.removeByFilter(ctx, userCred, map[string]string{
		"mesh_network_id": mn.Id,
	})
	return err
}

func (man *SMeshNetworkMemberManager) removeByMeshNetworkRouter(ctx context.Context, userCred mcclient.TokenCredential,
	mn *SMeshNetwork, router *SRouter) error {
	err := man.removeByFilter(ctx, userCred, map[string]string{
		"mesh_network_id": mn.Id,
		"router_id":       router.Id,
	})
	return err
}

func (man *SMeshNetworkMemberManager) getMemebersByMeshNetwork(ctx context.Context, userCred mcclient.TokenCredential, mn *SMeshNetwork) ([]SMeshNetworkMember, error) {
	members := []SMeshNetworkMember{}
	q := man.Query().Equals("mesh_network_id", mn.Id)
	err := db.FetchModelObjects(man, q, &members)
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (man *SMeshNetworkMemberManager) addMember(ctx context.Context, userCred mcclient.TokenCredential, mn *SMeshNetwork, router *SRouter, nets Subnets) (*SMeshNetworkMember, error) {
	member := &SMeshNetworkMember{
		MeshNetworkId:    mn.Id,
		RouterId:         router.Id,
		AdvertiseSubnets: nets.String(),
	}
	member.SetModelManager(man, member)
	member.Name = fmt.Sprintf("%s-%s", mn.Name, router.Name)
	man.TableSpec().Insert(member)
	return member, nil
}
