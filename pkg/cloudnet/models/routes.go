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

	"yunion.io/x/jsonutils"
	yerrors "yunion.io/x/pkg/util/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rand"
)

type SRoute struct {
	db.SStandaloneResourceBase

	IfaceId string `length:"32" nullable:"false" list:"user" create:"required"`
	Ifname  string `length:"32" nullable:"false" list:"user" create:"optional"`

	Network string `length:"32" nullable:"false" list:"user" update:"user" create:"required"`
	Gateway string `length:"32" nullable:"false" list:"user" update:"user" create:"optional"`

	RouterId string `length:"32" nullable:"false" list:"user" create:"optional"`
}

type SRouteManager struct {
	db.SStandaloneResourceBaseManager
}

var RouteManager *SRouteManager

func init() {
	RouteManager = &SRouteManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SRoute{},
			"routes_tbl",
			"route",
			"routes",
		),
	}
	RouteManager.SetVirtualObject(RouteManager)
}

func (man *SRouteManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if _, err := man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}

	ifaceV := validators.NewModelIdOrNameValidator("iface", "iface", ownerId)
	networkV := validators.NewIPv4PrefixValidator("network")
	gatewayV := validators.NewIPv4AddrValidator("gateway")
	vs := []validators.IValidator{
		networkV,
		gatewayV.Optional(true),
		ifaceV,
	}
	for _, v := range vs {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	iface := ifaceV.Model.(*SIface)
	network := networkV.Value.String()
	routerId := iface.RouterId
	{
		if routes, err := man.getByFilter(map[string]string{
			"router_id": routerId,
			"network":   network,
		}); err != nil {
			return nil, httperrors.NewConflictError("query existing route to network %s: %v", network, err)
		} else if len(routes) > 0 {
			return nil, httperrors.NewConflictError("route to %s already exist: %s(%s)", network, routes[0].Name, routes[0].Id)
		}
	}

	data.Set("router_id", jsonutils.NewString(routerId))
	data.Set("ifname", jsonutils.NewString(iface.Ifname))
	routerV := validators.NewModelIdOrNameValidator("router", "router", ownerId)
	if err := routerV.Validate(data); err != nil {
		return nil, err
	}
	if !data.Contains("name") {
		router := routerV.Model.(*SRouter)
		data.Set("name", jsonutils.NewString(
			router.Name+"-"+iface.Name+"-"+rand.String(4)),
		)
	}
	return nil, nil
}

func (man *SRouteManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "router", ModelKeyword: "router", OwnerId: userCred},
		{Key: "iface", ModelKeyword: "iface", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (route *SRoute) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if _, err := route.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data); err != nil {
		return nil, err
	}
	vs := []validators.IValidator{
		validators.NewIPv4PrefixValidator("network"),
		validators.NewIPv4AddrValidator("gateway"),
	}
	for _, v := range vs {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (man *SRouteManager) removeByIface(ctx context.Context, userCred mcclient.TokenCredential, iface *SIface) error {
	routes := []SRoute{}
	q := man.Query().Equals("iface_id", iface.Id)
	if err := db.FetchModelObjects(RouteManager, q, &routes); err != nil {
		return err
	}
	var errs []error
	for j := range routes {
		if err := routes[j].Delete(ctx, userCred); err != nil {
			errs = append(errs, err)
		}
	}
	return yerrors.NewAggregate(errs)
}

func (man *SRouteManager) getByFilter(filter map[string]string) ([]SRoute, error) {
	routes := []SRoute{}
	q := man.Query()
	for key, val := range filter {
		q = q.Equals(key, val)
	}
	if err := db.FetchModelObjects(RouteManager, q, &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

func (man *SRouteManager) getOneByFilter(filter map[string]string) (*SRoute, error) {
	routes, err := man.getByFilter(filter)
	if err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, errNotFound(fmt.Errorf("cannot find iface route: %#v", filter))
	}
	if len(routes) > 1 {
		return nil, errMoreThanOne(fmt.Errorf("found more than 1 iface routes: %#v", filter))
	}
	return &routes[0], nil
}

func (man *SRouteManager) checkExistenceByFilter(filter map[string]string) error {
	ifaces, err := man.getByFilter(filter)
	if err != nil {
		return err
	}
	if len(ifaces) > 0 {
		return fmt.Errorf("iface exist: %s(%s)", ifaces[0].Name, ifaces[0].Id)
	}
	return nil
}

func (man *SRouteManager) getByIface(iface *SIface) ([]SRoute, error) {
	filter := map[string]string{
		"router_id": iface.RouterId,
		"iface_id":  iface.Id,
	}
	routes, err := man.getByFilter(filter)
	if err != nil {
		return nil, err
	}
	return routes, nil
}

func (man *SRouteManager) getByRouter(router *SRouter) ([]SRoute, error) {
	filter := map[string]string{
		"router_id": router.Id,
	}
	routes, err := man.getByFilter(filter)
	if err != nil {
		return nil, err
	}
	return routes, nil
}

func (route *SRoute) routeLine() string {
	line := route.Network
	if route.Gateway != "" {
		line += " via " + route.Gateway
	}
	if route.Ifname != "" {
		line += " dev " + route.Ifname
	}
	return line
}

func (man *SRouteManager) routeLinesByIface(iface *SIface) ([]string, error) {
	routes, err := man.getByIface(iface)
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for i := range routes {
		route := &routes[i]
		line := route.routeLine()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func (man *SRouteManager) routeLinesRouter(router *SRouter) (map[string][]string, error) {
	routes, err := man.getByRouter(router)
	if err != nil {
		return nil, err
	}
	r := map[string][]string{}
	for i := range routes {
		route := &routes[i]
		line := route.routeLine()
		if line != "" {
			lines := r[route.Ifname]
			lines = append(lines, line)
			r[route.Ifname] = lines
		}
	}
	return r, nil
}
