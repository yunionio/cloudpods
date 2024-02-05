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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// bind_addr, default 0.0.0.0
// advertise_addr, default default route adddr, maybe k8s cluster ip
type SProxyAgent struct {
	db.SStandaloneResourceBase

	BindAddr      string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"admin"`
	AdvertiseAddr string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"admin"`
}

type SProxyAgentManager struct {
	db.SStandaloneResourceBaseManager
}

var ProxyAgentManager *SProxyAgentManager

func init() {
	ProxyAgentManager = &SProxyAgentManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SProxyAgent{},
			"proxy_agents_tbl",
			"proxy_agent",
			"proxy_agents",
		),
	}
	ProxyAgentManager.SetVirtualObject(ProxyAgentManager)
}

func (man *SProxyAgentManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	vs := []validators.IValidator{
		validators.NewIPv4AddrValidator("bind_addr").Optional(true),
		validators.NewIPv4AddrValidator("advertise_addr").Optional(true),
	}
	for _, v := range vs {
		if err := v.Validate(ctx, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (proxyagent *SProxyAgent) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	vs := []validators.IValidator{
		validators.NewIPv4AddrValidator("bind_addr"),
		validators.NewIPv4AddrValidator("advertise_addr"),
	}
	for _, v := range vs {
		v.Optional(true)
		if err := v.Validate(ctx, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (proxyagent *SProxyAgent) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	q := ForwardManager.Query().Equals("proxy_agent_id", proxyagent.Id)
	if count, err := q.CountWithError(); err != nil {
		return httperrors.NewServerError("count forwards using proxy endpoint %s(%s)",
			proxyagent.Name, proxyagent.Id)
	} else if count > 0 {
		return httperrors.NewConflictError("proxy endpoint %s(%s) is still used by %d forward(s)",
			proxyagent.Name, proxyagent.Id, count)
	} else {
		return nil
	}
}

func (man *SProxyAgentManager) allAgents(ctx context.Context) ([]SProxyAgent, error) {
	var (
		agents []SProxyAgent
		q      = man.Query()
	)
	if err := db.FetchModelObjects(man, q, &agents); err != nil {
		return nil, httperrors.NewServerError("query forwards by agent failed: %v", err)
	}
	return agents, nil
}
