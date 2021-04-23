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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	cloudproxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	compute_apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

// Add revision?
type SProxyEndpoint struct {
	db.SVirtualResourceBase

	User       string `nullable:"false" list:"user" update:"user" create:"optional"`
	Host       string `nullable:"false" list:"user" update:"user" create:"required"`
	Port       int    `nullable:"false" list:"user" update:"user" create:"optional"`
	PrivateKey string `nullable:"false" update:"user" list:"admin" get:"admin" create:"required"` // do not allow get, list

	IntranetIpAddr string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"required"`

	StatusDetail string `width:"128" charset:"ascii" nullable:"false" default:"init" list:"user" create:"optional" json:"status_detail"`
}

type SProxyEndpointManager struct {
	db.SVirtualResourceBaseManager
}

var ProxyEndpointManager *SProxyEndpointManager

func init() {
	ProxyEndpointManager = &SProxyEndpointManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SProxyEndpoint{},
			"proxy_endpoints_tbl",
			"proxy_endpoint",
			"proxy_endpoints",
		),
	}
	ProxyEndpointManager.SetVirtualObject(ProxyEndpointManager)
}

func (man *SProxyEndpointManager) AllowPerformCreateFromServer(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAllowClassPerform(rbacutils.ScopeProject, userCred, man, "create-from-server")
}

func (man *SProxyEndpointManager) PerformCreateFromServer(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *cloudproxy_api.ProxyEndpointCreateFromServerInput) (jsonutils.JSONObject, error) {
	serverId := input.ServerId
	if serverId == "" {
		return nil, httperrors.NewBadRequestError("server_id is required")
	}

	serverInfo, err := getServerInfo(ctx, userCred, serverId)
	if err != nil {
		return nil, err
	}
	if serverInfo.PrivateKey == "" {
		return nil, httperrors.NewBadRequestError("cannot find ssh private key for this server")
	}

	nic := serverInfo.GetNic()
	if nic == nil {
		return nil, httperrors.NewBadRequestError("cannot find usable network interface for this server")
	}
	host := serverInfo.Server.Eip
	if host == "" && nic.VpcId == compute_apis.DEFAULT_VPC_ID {
		host = nic.IpAddr
	}
	if host == "" {
		return nil, httperrors.NewBadRequestError("cannot find ssh host ip address for this server")
	}

	name := input.Name
	if name == "" {
		name = serverInfo.Server.Name
	}
	if err := db.NewNameValidator(man, userCred, name, nil); err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	proxyendpoint := &SProxyEndpoint{
		User:       "cloudroot",
		Host:       host,
		Port:       22,
		PrivateKey: serverInfo.PrivateKey,

		IntranetIpAddr: nic.IpAddr,
	}
	proxyendpoint.Name = name
	proxyendpoint.DomainId = userCred.GetProjectDomainId()
	proxyendpoint.ProjectId = userCred.GetProjectId()
	if err := man.TableSpec().Insert(ctx, proxyendpoint); err != nil {
		return nil, httperrors.NewServerError("database insertion error: %v", err)
	}

	var proxymatches []*SProxyMatch
	if nic.VpcId != "" {
		pm := &SProxyMatch{
			ProxyEndpointId: proxyendpoint.Id,
			MatchScope:      cloudproxy_api.PM_SCOPE_VPC,
			MatchValue:      nic.VpcId,
		}
		pm.Name = "vpc-" + nic.VpcId
		proxymatches = append(proxymatches, pm)
	}

	if nic.NetworkId != "" {
		pm := &SProxyMatch{
			ProxyEndpointId: proxyendpoint.Id,
			MatchScope:      cloudproxy_api.PM_SCOPE_NETWORK,
			MatchValue:      nic.NetworkId,
		}
		pm.Name = "network-" + nic.NetworkId
		proxymatches = append(proxymatches, pm)
	}
	for _, proxymatch := range proxymatches {
		proxymatch.DomainId = userCred.GetProjectDomainId()
		proxymatch.ProjectId = userCred.GetProjectId()
		if err := ProxyMatchManager.TableSpec().Insert(ctx, proxymatch); err != nil {
			log.Errorf("failed insertion of proxy match %s: %v", proxymatch.Name, err)
		}
	}

	return jsonutils.Marshal(proxyendpoint), nil
}

func (man *SProxyEndpointManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input cloudproxy_api.ProxyEndpointCreateInput,
) (*jsonutils.JSONDict, error) {
	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)

	if input, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput); err != nil {
		return nil, err
	} else {
		data.Update(jsonutils.Marshal(input))
	}

	vs := []validators.IValidator{
		validators.NewStringNonEmptyValidator("user").Default("cloudroot"),
		validators.NewStringNonEmptyValidator("host"),
		validators.NewPortValidator("port").Default(22),
		validators.NewSSHKeyValidator("private_key").Optional(true),

		validators.NewIPv4AddrValidator("intranet_ip_addr"),
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

func (man *SProxyEndpointManager) getById(id string) (*SProxyEndpoint, error) {
	m, err := db.FetchById(man, id)
	if err != nil {
		return nil, err
	}
	proxyendpoint := m.(*SProxyEndpoint)
	return proxyendpoint, err
}

func (proxyendpoint *SProxyEndpoint) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input cloudproxy_api.ProxyEndpointUpdateInput) (cloudproxy_api.ProxyEndpointUpdateInput, error) {
	var err error
	input.VirtualResourceBaseUpdateInput, err = proxyendpoint.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}

	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	vs := []validators.IValidator{
		validators.NewStringNonEmptyValidator("user"),
		validators.NewStringNonEmptyValidator("host"),
		validators.NewPortValidator("port"),
		validators.NewSSHKeyValidator("private_key").Optional(true),
	}
	for _, v := range vs {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return input, err
		}
	}
	return input, nil
}

func (proxyendpoint *SProxyEndpoint) ValidateDeleteCondition(ctx context.Context) error {
	q := ForwardManager.Query().Equals("proxy_endpoint_id", proxyendpoint.Id)
	if count, err := q.CountWithError(); err != nil {
		return httperrors.NewServerError("count forwards using proxy endpoint %s(%s)",
			proxyendpoint.Name, proxyendpoint.Id)
	} else if count > 0 {
		return httperrors.NewConflictError("proxy endpoint %s(%s) is still used by %d forward(s)",
			proxyendpoint.Name, proxyendpoint.Id, count)
	} else {
		return nil
	}
}

func (proxyendpoint *SProxyEndpoint) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	var pms []SProxyMatch
	q := ProxyMatchManager.Query().Equals("proxy_endpoint_id", proxyendpoint.Id)
	if err := db.FetchModelObjects(ProxyMatchManager, q, &pms); err != nil {
		return httperrors.NewServerError("fetch proxy matches for endpoint %s(%s)",
			proxyendpoint.Name, proxyendpoint.Id)
	}

	for i := range pms {
		pm := &pms[i]
		err := db.DeleteModel(ctx, userCred, pm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (man *SProxyEndpointManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input cloudproxy_api.ProxyEndpointListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	filters := [][2]string{
		[2]string{cloudproxy_api.PM_SCOPE_VPC, input.VpcId},
		[2]string{cloudproxy_api.PM_SCOPE_NETWORK, input.NetworkId},
	}
	for _, filter := range filters {
		if v := filter[1]; v != "" {
			pmQ := ProxyMatchManager.Query("proxy_endpoint_id").
				Equals("match_scope", filter[0]).
				Equals("match_value", v)
			q = q.In("id", pmQ.SubQuery())
		}
	}
	return q, nil
}
