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
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	yerrors "yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// Add revision?
type SRouter struct {
	db.SStandaloneResourceBase

	User       string `nullable:"false" list:"user" update:"user" create:"optional"`
	Host       string `nullable:"false" list:"user" update:"user" create:"required"`
	Port       int    `nullable:"false" list:"user" update:"user" create:"optional"`
	PrivateKey string `nullable:"true" update:"user" create:"optional"` // do not allow get, list

	RealizeWgIfaces bool `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	RealizeRoutes   bool `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	RealizeRules    bool `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
}

type SRouterManager struct {
	db.SStandaloneResourceBaseManager
}

var RouterManager *SRouterManager

func init() {
	RouterManager = &SRouterManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SRouter{},
			"routers_tbl",
			"router",
			"routers",
		),
	}
	RouterManager.SetVirtualObject(RouterManager)
}

func (man *SRouterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if _, err := man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}

	vs := []validators.IValidator{
		validators.NewStringNonEmptyValidator("user").Default("cloudroot"),
		validators.NewStringNonEmptyValidator("host"),
		validators.NewPortValidator("port").Default(22),
		validators.NewSSHKeyValidator("private_key").Optional(true),
		validators.NewBoolValidator("realize_wg_ifaces").Default(true),
		validators.NewBoolValidator("realize_routes").Default(true),
		validators.NewBoolValidator("realize_rules").Default(true),
	}
	for _, v := range vs {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	// populate ssh credential through "cloudhost"
	//
	// if ! skip validation {
	// 	ssh credential validation
	// }
	return data, nil
}

func (router *SRouter) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	err := RuleManager.addRouterRules(ctx, userCred, router)
	if err != nil {
		log.Errorf("add router rule: %v", err)
	}
}

func (man *SRouterManager) getById(id string) (*SRouter, error) {
	m, err := db.FetchById(man, id)
	if err != nil {
		return nil, err
	}
	router := m.(*SRouter)
	return router, err
}

func (router *SRouter) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if _, err := router.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data); err != nil {
		return nil, err
	}
	vs := []validators.IValidator{
		validators.NewStringNonEmptyValidator("user"),
		validators.NewStringNonEmptyValidator("host"),
		validators.NewPortValidator("port"),
		validators.NewSSHKeyValidator("private_key").Optional(true),
		validators.NewBoolValidator("realize_wg_ifaces"),
		validators.NewBoolValidator("realize_routes"),
		validators.NewBoolValidator("realize_rules"),
	}
	for _, v := range vs {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	data.Set("_old_endpoint", jsonutils.NewString(router.endpointIP()))
	return data, nil
}

func (router *SRouter) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	endpointOld, _ := data.GetString("_old_endpoint")
	endpoint := router.endpointIP()
	if endpoint != endpointOld {
		err := IfacePeerManager.updateEndpointIPByPeerRouter(ctx, endpoint, router)
		if err != nil {
			log.Errorf("updating peer endpoint failed: %v", err)
		}
	}
}

func (router *SRouter) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	var errs []error
	if err := MeshNetworkManager.removeRouter(ctx, userCred, router); err != nil {
		errs = append(errs, err)
	}
	if err := IfaceManager.removeByRouter(ctx, userCred, router); err != nil {
		errs = append(errs, err)
	}
	if err := RuleManager.removeByRouter(ctx, userCred, router); err != nil {
		errs = append(errs, err)
	}
	if err := router.SStandaloneResourceBase.CustomizeDelete(ctx, userCred, query, data); err != nil {
		errs = append(errs, err)
	}
	return yerrors.NewAggregate(errs)
}

func (router *SRouter) AllowPerformJoinMeshNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, router, "join-mesh-network")
}

func (router *SRouter) PerformJoinMeshNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mnV := validators.NewModelIdOrNameValidator("mesh_network", "meshnetwork", userCred)
	advSubnetsV := validators.NewValidatorByActor("advertise_subnets",
		validators.NewActorJoinedBy(",", validators.NewActorIPv4Prefix()))
	{
		vs := []validators.IValidator{
			mnV,
			advSubnetsV,
		}
		jd, ok := data.(*jsonutils.JSONDict)
		if !ok {
			return nil, httperrors.NewBadRequestError("expecting json dict")
		}
		for _, v := range vs {
			if err := v.Validate(jd); err != nil {
				return nil, err
			}
		}
	}
	// TODO dedup
	nets := gotypes.ConvertSliceElemType(advSubnetsV.Value, (**netutils.IPV4Prefix)(nil)).([]*netutils.IPV4Prefix)
	if len(nets) == 0 {
		return nil, httperrors.NewBadRequestError("advertise_subnets must not be empty")
	}
	mn := mnV.Model.(*SMeshNetwork)
	if err := mn.addRouter(ctx, userCred, router, nets); err != nil {
		return nil, err
	}
	return data, nil
}

func (router *SRouter) AllowPerformLeaveMeshNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, router, "leave-mesh-network")
}

func (router *SRouter) PerformLeaveMeshNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	jd, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewBadRequestError("expecting json dict")
	}
	mnV := validators.NewModelIdOrNameValidator("mesh_network", "meshnetwork", userCred)
	if err := mnV.Validate(jd); err != nil {
		return nil, err
	}
	mn := mnV.Model.(*SMeshNetwork)
	if err := mn.removeRouter(ctx, userCred, router); err != nil {
		return nil, err
	}
	return nil, nil
}

func (router *SRouter) AllowPerformRegisterIfname(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, router, "register-ifname")
}

func (router *SRouter) PerformRegisterIfname(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	jd, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewBadRequestError("expecting json dict")
	}
	ifnameV := validators.NewRegexpValidator("ifname", regexpIfname)
	if err := ifnameV.Validate(jd); err != nil {
		return nil, err
	}
	_, err := IfaceManager.addIface(ctx, userCred, router, ifnameV.Value)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (router *SRouter) AllowPerformUnregisterIfname(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, router, "unregister-ifname")
}

func (router *SRouter) PerformUnregisterIfname(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	jd, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewBadRequestError("expecting json dict")
	}
	ifname, err := jd.GetString("ifname")
	if err != nil {
		return nil, httperrors.NewBadRequestError("get request ifname field: %v", err)
	}
	iface, err := IfaceManager.getByRouterIfname(router, ifname)
	if err != nil {
		return nil, httperrors.NewBadRequestError("get iface: %s", err)
	}
	if iface.NetworkId != "" {
		// XXX can_unregister
		return nil, httperrors.NewBadRequestError("please use leave network to unregister")
	}
	if err := iface.remove(ctx, userCred); err != nil {
		return nil, httperrors.NewBadRequestError("remove iface: %v", err)
	}
	return nil, nil
}

func (router *SRouter) mustFindFreePort(ctx context.Context) int {
	// loop through ifaces listen port
	ifaces, err := IfaceManager.getByRouter(router)
	if err != nil {
		panic(err)
	}
	for sport := 20000; sport < 65536; sport++ {
		notfound := true
		for i := range ifaces {
			if ifaces[i].ListenPort == sport {
				notfound = false
				break
			}
		}
		if notfound {
			return sport
		}
	}
	panic(fmt.Sprintf("cannot find free port for host %s(%s)", router.Name, router.Id))
}

func (router *SRouter) endpointIP() string {
	return router.Host
}

func (router *SRouter) AllowPerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, router, "deploy")
}

func (router *SRouter) PerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := RouterDeploymentManager.requestDeployment(ctx, userCred, router); err != nil {
		return nil, err
	}
	return nil, nil
}
