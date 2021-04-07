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
	"net/url"
	"sync"

	"github.com/coredns/coredns/plugin/pkg/log"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	proxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SScript struct {
	db.SSharableVirtualResourceBase
	// remote
	Type                string `width:"16" nullable:"false"`
	PlaybookReferenceId string `width:"128" nullable:"false"`
	MaxTryTimes         int    `default:"1"`
}

type SScriptManager struct {
	db.SSharableVirtualResourceBaseManager
}

var ScriptManager *SScriptManager

func init() {
	ScriptManager = &SScriptManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SScript{},
			"script_tbl",
			"script",
			"scripts",
		),
	}
	ScriptManager.SetVirtualObject(ScriptManager)
	registerArgGenerator(MonitorAgent, getArgs)
}

type argGenerator func(ctx context.Context, input api.ScriptApplyInput, details *comapi.ServerDetails) (map[string]interface{}, error)

var argGenerators = &sync.Map{}

func registerArgGenerator(name string, ag argGenerator) {
	argGenerators.Store(name, ag)
}

func getArgGenerator(name string) (argGenerator, bool) {
	v, ok := argGenerators.Load(name)
	if !ok {
		return nil, ok
	}
	return v.(argGenerator), ok
}

func convertInfluxdbUrl(ctx context.Context, pUrl string, endpointId string) (string, error) {
	session := auth.AdminSessionWithInternal(ctx, "", "", "")
	filter := jsonutils.NewDict()
	filter.Set("proxy_endpoint_id", jsonutils.NewString(endpointId))
	filter.Set("opaque", jsonutils.NewString(pUrl))
	filter.Set("scope", jsonutils.NewString("system"))
	lr, err := cloudproxy.Forwards.List(session, filter)
	if err != nil {
		return "", errors.Wrap(err, "failed to list forward")
	}
	var port int64
	if len(lr.Data) > 0 {
		port, _ = lr.Data[0].Int("bind_port")
	} else {
		rUrl, err := url.Parse(pUrl)
		if err != nil {
			return "", errors.Wrap(err, "invalid influxdbUrl?")
		}
		// create one
		createP := jsonutils.NewDict()
		createP.Set("proxy_endpoint", jsonutils.NewString(endpointId))
		createP.Set("type", jsonutils.NewString(proxy_api.FORWARD_TYPE_REMOTE))
		createP.Set("remote_addr", jsonutils.NewString(rUrl.Hostname()))
		createP.Set("remote_port", jsonutils.NewString(rUrl.Port()))
		createP.Set("generate_name", jsonutils.NewString("influxdb proxy"))
		createP.Set("opaque", jsonutils.NewString(pUrl))
		forward, err := cloudproxy.Forwards.Create(session, createP)
		if err != nil {
			return "", errors.Wrapf(err, "unable to create forward with create params %s", createP.String())
		}
		port, _ = forward.Int("bind_port")
	}
	// fetch proxy_endpoint address
	ep, err := cloudproxy.ProxyEndpoints.Get(session, endpointId, nil)
	if err != nil {
		return "", errors.Wrapf(err, "unable to get proxy endpoint %s", endpointId)
	}
	address, _ := ep.GetString("intranet_ip_addr")
	return fmt.Sprintf("https://%s:%d", address, port), nil
}

func getArgs(ctx context.Context, input api.ScriptApplyInput, detail *comapi.ServerDetails) (map[string]interface{}, error) {
	influxdbUrl, err := getInfluxdbUrl(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get influxdbUrl")
	}
	// convert influxdbUrl
	if len(input.ProxyEndpointId) > 0 {
		influxdbUrl, err = convertInfluxdbUrl(ctx, influxdbUrl, input.ProxyEndpointId)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convertInfluxdbUrl %s", influxdbUrl)
		}
	}
	vmId := detail.Id
	tenantId := detail.ProjectId
	domainId := detail.DomainId
	ret := map[string]interface{}{
		"influxdb_url":       influxdbUrl,
		"influxdb_name":      "telegraf",
		"onecloud_vm_id":     vmId,
		"onecloud_tenant_id": tenantId,
		"onecloud_domain_id": domainId,
	}
	return ret, nil
}

var influxdbUrl string

func getInfluxdbUrl(ctx context.Context) (string, error) {
	if len(influxdbUrl) > 0 {
		return influxdbUrl, nil
	}
	session := auth.GetAdminSession(ctx, "", "")
	params := jsonutils.NewDict()
	params.Set("interface", jsonutils.NewString("public"))
	params.Set("service", jsonutils.NewString("influxdb"))
	ret, err := modules.EndpointsV3.List(session, params)
	if err != nil {
		return "", err
	}
	if len(ret.Data) == 0 {
		return "", fmt.Errorf("no sucn endpoint with 'internal' interface and 'influxdb' service")
	}
	url, _ := ret.Data[0].GetString("url")
	return url, nil
}

var MonitorAgent = "monitor agent"

func (sm *SScriptManager) InitializeData() error {
	q := sm.Query().Equals("playbook_reference", MonitorAgent)
	n, err := q.CountWithError()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	s := SScript{
		PlaybookReferenceId: MonitorAgent,
	}
	s.ProjectId = "system"
	s.IsPublic = true
	s.PublicScope = "system"
	err = sm.TableSpec().Insert(context.Background(), &s)
	if err != nil {
		return err
	}
	return nil
}

func (sm *SScriptManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ScriptCreateInput) (api.ScriptCreateInput, error) {
	// check ansible playbook reference
	session := auth.GetSessionWithInternal(ctx, userCred, "", "")
	pr, err := modules.AnsiblePlaybookReference.Get(session, input.PlaybookReference, nil)
	if err != nil {
		return input, errors.Wrapf(err, "unable to get AnsiblePlaybookReference %q", input.PlaybookReference)
	}
	id, _ := pr.GetString("id")
	input.PlaybookReference = id
	return input, nil
}

func (s *SScript) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	s.Status = api.SCRIPT_STATUS_READY
	s.PlaybookReferenceId, _ = data.GetString("playbook_reference")
	return nil
}

func (sm *SScriptManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.ScriptDetails {
	vDetails := sm.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	details := make([]api.ScriptDetails, len(objs))
	for i := range details {
		details[i].SharableVirtualResourceDetails = vDetails[i]
		script := objs[i].(*SScript)
		ais, err := script.ApplyInfos()
		if err != nil {
			log.Errorf("unable to get ApplyInfos of script %s: %v", script.Id, err)
		}
		details[i].ApplyInfos = ais
	}
	return details
}

func (s *SScript) ApplyInfos() ([]api.SApplyInfo, error) {
	q := ScriptApplyManager.Query().Equals("script_id", s.Id)
	var sa []SScriptApply
	err := db.FetchModelObjects(ScriptApplyManager, q, &sa)
	if err != nil {
		return nil, err
	}
	ai := make([]api.SApplyInfo, len(sa))
	for i := range ai {
		ai[i].ServerId = sa[i].GuestId
		ai[i].EipFirst = sa[i].EipFirst.Bool()
		ai[i].ProxyEndpointId = sa[i].ProxyEndpointId
		ai[i].TryTimes = sa[i].TryTimes
	}
	return ai, nil
}

func (s *SScript) AllowPerformApply(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return s.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, s, "apply")
}

func (s *SScript) PerformApply(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ScriptApplyInput) (api.ScriptApplyOutput, error) {
	output := api.ScriptApplyOutput{}
	serverInfo, err := s.checkServer(ctx, userCred, input.ServerID)
	if err != nil {
		return output, err
	}
	// select proxyEndpoint automatically
	if len(input.ProxyEndpointId) == 0 && input.AutoChooseProxyEndpoint {
		var proxyEndpointId string
		// find suitable proxyEndpoint
		// network first
		session := auth.GetAdminSession(ctx, "", "")
		for _, netId := range serverInfo.NetworkIds {
			filter := jsonutils.NewDict()
			filter.Set("network_id", jsonutils.NewString(netId))
			lr, err := cloudproxy.ProxyEndpoints.List(session, filter)
			if err != nil {
				return output, errors.Wrapf(err, "unable to list proxy endpoint in network %q", netId)
			}
			if len(lr.Data) == 0 {
				continue
			}
			proxyEndpointId, _ = lr.Data[0].GetString("id")
			break
		}
		if len(proxyEndpointId) == 0 {
			filter := jsonutils.NewDict()
			filter.Set("vpc_id", jsonutils.NewString(serverInfo.VpcId))
			lr, err := cloudproxy.ProxyEndpoints.List(session, filter)
			if err != nil {
				return output, errors.Wrapf(err, "unable to list proxy endpoint in vpc %q", serverInfo.VpcId)
			}
			if len(lr.Data) > 0 {
				// TODO Choose strictly
				proxyEndpointId, _ = lr.Data[0].GetString("id")
			}
		}
		if len(proxyEndpointId) == 0 {
			return output, httperrors.NewInputParameterError("can't find suitable proxy endpoint for server %s, please connect with admin to create one", serverInfo.serverDetails.Name)
		}
		input.ProxyEndpointId = proxyEndpointId
	}
	ag, _ := getArgGenerator(MonitorAgent)
	args, err := ag(ctx, input, serverInfo.serverDetails)
	if err != nil {
		return output, errors.Wrapf(err, "unable to get args of server %s", serverInfo.ServerId)
	}
	sa, err := ScriptApplyManager.createScriptApply(ctx, s.Id, serverInfo.ServerId, input.ProxyEndpointId, input.EipFirst, args)
	if err != nil {
		return output, errors.Wrapf(err, "unable to apply script to server %s", serverInfo.ServerId)
	}
	err = sa.StartApply(ctx, userCred)
	if err != nil {
		return output, errors.Wrapf(err, "unable to apply script to server %s", serverInfo.ServerId)
	}
	output.ScriptApplyId = sa.Id
	return output, nil
}

type sServerInfo struct {
	ServerId      string
	VpcId         string
	NetworkIds    []string
	serverDetails *comapi.ServerDetails
}

func (s *SScript) checkServer(ctx context.Context, userCred mcclient.TokenCredential, serverId string) (sServerInfo, error) {
	session := auth.GetSessionWithInternal(ctx, userCred, "", "")
	// check server
	data, err := modules.Servers.Get(session, serverId, nil)
	if err != nil {
		if httputils.ErrorCode(err) == 404 {
			return sServerInfo{}, httperrors.NewInputParameterError("no such server %s", serverId)
		}
		return sServerInfo{}, fmt.Errorf("unable to get server %s: %s", serverId, httputils.ErrorMsg(err))
	}
	info := sServerInfo{}
	var serverDetails comapi.ServerDetails
	err = data.Unmarshal(&serverDetails)
	if err != nil {
		return info, errors.Wrap(err, "unable to unmarshal serverDetails")
	}
	if serverDetails.Status != comapi.VM_RUNNING {
		return info, httperrors.NewInputParameterError("can only apply scripts to %s server", comapi.VM_RUNNING)
	}
	info.serverDetails = &serverDetails
	info.ServerId = serverDetails.Id

	networkIds := sets.NewString()
	for _, nic := range serverDetails.Nics {
		networkIds.Insert(nic.NetworkId)
		info.VpcId = nic.VpcId
	}
	info.NetworkIds = networkIds.UnsortedList()
	return info, nil
}
