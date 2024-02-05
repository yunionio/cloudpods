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
	"math/rand"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	cloudproxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SForward struct {
	db.SVirtualResourceBase

	ProxyEndpointId string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	ProxyAgentId    string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`

	Type        string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	RemoteAddr  string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	RemotePort  int    `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	BindPortReq int    `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"`

	Opaque string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`

	BindPort        int       `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"`
	LastSeen        time.Time `nullable:"true" get:"user" list:"user"`
	LastSeenTimeout int       `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional" default:"117"`
}

type SForwardManager struct {
	db.SVirtualResourceBaseManager
}

var ForwardManager *SForwardManager

func init() {
	ForwardManager = &SForwardManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SForward{},
			"forwards_tbl",
			"forward",
			"forwards",
		),
	}
	ForwardManager.SetVirtualObject(ForwardManager)
}

func (man *SForwardManager) validateLocalSetPort(ctx context.Context, data *jsonutils.JSONDict, agentId string, portReq int) (*jsonutils.JSONDict, error) {
	var (
		fwds []SForward
		q    = man.Query().
			Equals("proxy_agent_id", agentId).
			Equals("bind_port_req", portReq)
	)
	if err := db.FetchModelObjects(man, q, &fwds); err != nil {
		return nil, httperrors.NewServerError("query forwards by agent failed: %v", err)
	}
	if len(fwds) > 0 {
		return nil, httperrors.NewConflictError("port %d on agent %s was already occupied",
			portReq, agentId)
	}
	data.Set("bind_port", jsonutils.NewInt(int64(portReq)))
	return data, nil
}

func (man *SForwardManager) validateRemoteSetPort(ctx context.Context, data *jsonutils.JSONDict, epId string, portReq int) (*jsonutils.JSONDict, error) {
	var (
		fwds []SForward
		q    = man.Query().
			Equals("proxy_endpoint_id", epId).
			Equals("bind_port_req", portReq)
	)
	if err := db.FetchModelObjects(man, q, &fwds); err != nil {
		return nil, httperrors.NewServerError("query forwards by endpoint id failed: %v", err)
	}
	if len(fwds) > 0 {
		return nil, httperrors.NewConflictError("port %d on proxy endpoint %s was already occupied",
			portReq, epId)
	}
	data.Set("bind_port", jsonutils.NewInt(int64(portReq)))
	return data, nil
}

func (man *SForwardManager) validateLocalSelectAgent(ctx context.Context, data *jsonutils.JSONDict, portReq int) (*jsonutils.JSONDict, error) {
	agents, err := ProxyAgentManager.allAgents(ctx)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	agentsNum := len(agents)
	if agentsNum == 0 {
		return nil, httperrors.NewResourceNotFoundError("empty proxy agents set")
	}
	s := rand.Intn(agentsNum)
	for i := s; ; {
		agent := &agents[i]

		var err error
		data, err = man.validateLocalSetPort(ctx, data, agent.Id, portReq)
		if err == nil {
			data.Set("proxy_agent_id", jsonutils.NewString(agent.Id))
			return data, nil
		}

		i += 1
		if i == agentsNum {
			i = 0
		}
		if i == s {
			break
		}
	}
	return nil, httperrors.NewResourceNotFoundError("no proxy agent accepts request for port %d", portReq)
}

func (man *SForwardManager) validateRemoteSelectAgent(ctx context.Context, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	agents, err := ProxyAgentManager.allAgents(ctx)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	agentsNum := len(agents)
	if agentsNum == 0 {
		return nil, httperrors.NewResourceNotFoundError("empty proxy agents set")
	}
	i := rand.Intn(agentsNum)
	agent := agents[i]
	data.Set("proxy_agent_id", jsonutils.NewString(agent.Id))
	return data, nil
}

func (man *SForwardManager) validatePortReq(
	ctx context.Context,
	typ string, portReq int, agentId, epId string,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	validateOne := func(portReq int) (*jsonutils.JSONDict, error) {
		var err error
		switch typ {
		case cloudproxy_api.FORWARD_TYPE_LOCAL:
			if agentId == "" {
				data, err = man.validateLocalSelectAgent(ctx, data, portReq)
			} else {
				data, err = man.validateLocalSetPort(ctx, data, agentId, portReq)
			}
		case cloudproxy_api.FORWARD_TYPE_REMOTE:
			data, err = man.validateRemoteSetPort(ctx, data, epId, portReq)
		}
		return data, err
	}

	if typ == cloudproxy_api.FORWARD_TYPE_REMOTE && agentId == "" {
		var err error
		data, err = man.validateRemoteSelectAgent(ctx, data)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("select proxy agent: %v", err)
		}
	}

	var err error
	if portReq <= 0 {
		portTotal := cloudproxy_api.BindPortMax - cloudproxy_api.BindPortMin + 1
		portReqStart := rand.Intn(portTotal)
		for portInc := portReqStart; ; {
			data, err = validateOne(cloudproxy_api.BindPortMin + portInc)
			if err == nil {
				break
			}
			portInc += 1
			if portInc == portTotal {
				portInc = 0
			}
			if portInc == portReqStart {
				return nil, httperrors.NewOutOfResourceError("no available port for bind")
			}
		}
	} else {
		data, err = validateOne(portReq)
	}
	return data, err
}

func (man *SForwardManager) PerformCreateFromServer(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *cloudproxy_api.ForwardCreateFromServerInput) (jsonutils.JSONObject, error) {
	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)

	typeV := validators.NewStringChoicesValidator("type", cloudproxy_api.FORWARD_TYPES)
	portReqV := validators.NewRangeValidator("bind_port_req", cloudproxy_api.BindPortMin, cloudproxy_api.BindPortMax)
	remotePortV := validators.NewPortValidator("remote_port")
	{
		for _, v := range []validators.IValidator{
			typeV,
			portReqV.Optional(true),
			remotePortV,

			validators.NewNonNegativeValidator("last_seen_timeout").Optional(true),
		} {
			if err := v.Validate(ctx, data); err != nil {
				return nil, err
			}
		}
	}

	serverId := input.ServerId
	if serverId == "" {
		return nil, httperrors.NewBadRequestError("server_id is required")
	}
	serverInfo, err := getServerInfo(ctx, userCred, serverId)
	if err != nil {
		return nil, err
	}
	nic := serverInfo.GetNic()
	if nic == nil {
		return nil, httperrors.NewBadRequestError("cannot find network interface for this server")
	}
	proxymatch := ProxyMatchManager.findMatch(ctx, nic.NetworkId, nic.VpcId)
	if proxymatch == nil {
		return nil, httperrors.NewBadRequestError("cannot find an endpoint for this server")
	}
	data.Set("opaque", jsonutils.NewString(serverInfo.Server.Id))
	data.Set("remote_addr", jsonutils.NewString(nic.IpAddr))
	data.Set("proxy_endpoint_id", jsonutils.NewString(proxymatch.ProxyEndpointId))

	typ := typeV.Value
	agentId := ""
	epId := proxymatch.ProxyEndpointId
	if data.Contains("bind_port_req") {
		portReq := int(portReqV.Value)
		data, err = man.validatePortReq(ctx, typ, portReq, agentId, epId, data)
	} else {
		data, err = man.validatePortReq(ctx, typ, -1, agentId, epId, data)
	}

	forwardObj, err := db.NewModelObject(man)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	forward := forwardObj.(*SForward)
	if err := data.Unmarshal(forward); err != nil {
		return nil, httperrors.NewServerError("unmarshal create params: %v", err)
	}
	forward.Name = fmt.Sprintf("%s-%s-%d", serverInfo.Server.Name, typ, forward.RemotePort)
	forward.DomainId = userCred.GetProjectDomainId()
	forward.ProjectId = userCred.GetProjectId()
	if err := man.TableSpec().Insert(ctx, forward); err != nil {
		return nil, httperrors.NewServerError("database insertion error: %v", err)
	}

	return db.GetItemDetails(man, forward, ctx, userCred)
}

func (man *SForwardManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	endpointV := validators.NewModelIdOrNameValidator("proxy_endpoint", ProxyEndpointManager.Keyword(), ownerId)
	agentV := validators.NewModelIdOrNameValidator("proxy_agent", ProxyAgentManager.Keyword(), ownerId)
	typeV := validators.NewStringChoicesValidator("type", cloudproxy_api.FORWARD_TYPES)
	portReqV := validators.NewRangeValidator("bind_port_req", cloudproxy_api.BindPortMin, cloudproxy_api.BindPortMax)
	for _, v := range []validators.IValidator{
		endpointV,
		agentV.Optional(true),

		typeV,
		validators.NewIPv4AddrValidator("remote_addr"),
		validators.NewPortValidator("remote_port"),
		portReqV.Optional(true),

		validators.NewNonNegativeValidator("last_seen_timeout").Optional(true),
	} {
		if err := v.Validate(ctx, data); err != nil {
			return nil, err
		}
	}

	typ := typeV.Value
	epId := endpointV.Model.GetId()
	var agentId string
	if agentV.Model != nil {
		agentId = agentV.Model.GetId()
	}

	var err error
	if data.Contains("bind_port_req") {
		portReq := int(portReqV.Value)
		data, err = man.validatePortReq(ctx, typ, portReq, agentId, epId, data)
	} else {
		data, err = man.validatePortReq(ctx, typ, -1, agentId, epId, data)
	}
	return data, err
}

func (fwd *SForward) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	endpointV := validators.NewModelIdOrNameValidator("proxy_endpoint", ProxyEndpointManager.Keyword(), userCred)
	agentV := validators.NewModelIdOrNameValidator("proxy_agent", ProxyAgentManager.Keyword(), userCred)
	portReqV := validators.NewRangeValidator("bind_port_req", cloudproxy_api.BindPortMin, cloudproxy_api.BindPortMax)
	for _, v := range []validators.IValidator{
		endpointV,
		agentV.Optional(true),

		validators.NewIPv4AddrValidator("remote_addr"),
		validators.NewPortValidator("remote_port"),
		portReqV,

		validators.NewNonNegativeValidator("last_seen_timeout"),
	} {
		v.Optional(true)
		if err := v.Validate(ctx, data); err != nil {
			return nil, err
		}
	}

	portReq := int(portReqV.Value)
	if portReq != fwd.BindPortReq {
		var err error
		var agentId string
		if agentV.Model == nil {
			agentId = fwd.ProxyAgentId
		} else {
			agentId = agentV.Model.GetId()
		}
		switch typ := fwd.Type; typ {
		case cloudproxy_api.FORWARD_TYPE_LOCAL:
			data, err = ForwardManager.validateLocalSetPort(ctx, data, agentId, portReq)
		case cloudproxy_api.FORWARD_TYPE_REMOTE:
			data, err = ForwardManager.validateRemoteSetPort(ctx, data, agentId, portReq)
		}
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (man *SForwardManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input cloudproxy_api.ForwardListInput,
) (*sqlchemy.SQuery, error) {
	filters := [][2]string{
		[2]string{"type", input.Type},
		[2]string{"remote_addr", input.RemoteAddr},
		[2]string{"proxy_endpoint_id", input.ProxyEndpointId},
		[2]string{"proxy_agent_id", input.ProxyAgentId},
		[2]string{"opaque", input.Opaque},
	}
	for _, filter := range filters {
		if v := filter[1]; v != "" {
			q = q.Equals(filter[0], v)
		}
	}
	intFilters := []struct {
		name string
		val  *int
	}{
		{"remote_port", input.RemotePort},
		{"bind_port_req", input.BindPortReq},
	}
	for _, filter := range intFilters {
		if v := filter.val; v != nil {
			q = q.Equals(filter.name, *v)
		}
	}
	return q, nil
}

func (man *SForwardManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []*jsonutils.JSONDict {
	fwds := gotypes.ConvertSliceElemType(objs, (**SForward)(nil)).([]*SForward)

	paMap := map[string]*SProxyAgent{}
	peMap := map[string]*SProxyEndpoint{}
	{
		var paIds []string
		var peIds []string
		{
			paIdMap := map[string]string{}
			peIdMap := map[string]string{}
			for _, fwd := range fwds {
				paIdMap[fwd.ProxyAgentId] = ""
				peIdMap[fwd.ProxyEndpointId] = ""
			}
			for id := range paIdMap {
				if id != "" {
					paIds = append(paIds, id)
				}
			}
			for id := range peIdMap {
				if id != "" {
					peIds = append(peIds, id)
				}
			}
		}

		var pas []SProxyAgent
		var pes []SProxyEndpoint
		{
			paQ := ProxyAgentManager.Query().In("id", paIds)
			if err := db.FetchModelObjects(ProxyAgentManager, paQ, &pas); err != nil {
				return nil
			}
			peQ := ProxyEndpointManager.Query().In("id", peIds)
			if err := db.FetchModelObjects(ProxyEndpointManager, peQ, &pes); err != nil {
				return nil
			}
		}

		for i := range pas {
			pa := &pas[i]
			paMap[pa.Id] = pa
		}
		for i := range pes {
			pe := &pes[i]
			peMap[pe.Id] = pe
		}
	}

	r := make([]*jsonutils.JSONDict, len(objs))
	for i, fwd := range fwds {
		d := jsonutils.NewDict()
		pa, paOK := paMap[fwd.ProxyAgentId]
		pe, peOK := peMap[fwd.ProxyEndpointId]
		if paOK || peOK {
			if paOK {
				d.Set("proxy_agent", jsonutils.NewString(pa.Name))
			}
			if peOK {
				d.Set("proxy_endpoint", jsonutils.NewString(pe.Name))
			}
			switch fwd.Type {
			case cloudproxy_api.FORWARD_TYPE_LOCAL:
				if paOK {
					d.Set("bind_addr", jsonutils.NewString(pa.AdvertiseAddr))
				}
			case cloudproxy_api.FORWARD_TYPE_REMOTE:
				if peOK {
					d.Set("bind_addr", jsonutils.NewString(pe.IntranetIpAddr))
				}
			}
			r[i] = d
		}
	}
	return r
}

func (fwd *SForward) PerformHeartbeat(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *cloudproxy_api.ForwardHeartbeatInput) (jsonutils.JSONObject, error) {
	if _, err := db.Update(fwd, func() error {
		fwd.LastSeen = time.Now()
		return nil
	}); err != nil {
		return nil, err
	}
	return nil, nil
}

func (fwd *SForward) GetDetailsLastseen(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	var lastSeen string
	if fwd.LastSeen.IsZero() {
		lastSeen = ""
	} else {
		lastSeen = fwd.LastSeen.Format("2006-01-02T15:04:05.000000Z")
	}
	ret.Add(jsonutils.NewString(lastSeen), "last_seen")
	return ret, nil
}
