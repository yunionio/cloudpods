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

package utils

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	ansible_api "yunion.io/x/onecloud/pkg/apis/ansible"
	proxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type sServerInfo struct {
	ServerId      string
	VpcId         string
	NetworkIds    []string
	serverDetails *comapi.ServerDetails
}

type sProxyEndpoint struct {
	Id      string
	Address string
}

func proxyEndpoints(ctx context.Context, proxyEndpointId string, info sServerInfo) ([]sProxyEndpoint, error) {
	pes := make([]sProxyEndpoint, 0)
	session := auth.GetAdminSession(ctx, "", "")
	if len(proxyEndpointId) > 0 {
		ep, err := cloudproxy.ProxyEndpoints.Get(session, proxyEndpointId, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get proxy endpoint %s", proxyEndpointId)
		}
		address, _ := ep.GetString("intranet_ip_addr")
		pes = append(pes, sProxyEndpoint{proxyEndpointId, address})
		return pes, nil
	}
	proxyEndpointIds := sets.NewString()
	for _, netId := range info.NetworkIds {
		filter := jsonutils.NewDict()
		filter.Set("network_id", jsonutils.NewString(netId))
		lr, err := cloudproxy.ProxyEndpoints.List(session, filter)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to list proxy endpoint in network %q", netId)
		}
		for i := range lr.Data {
			proxyEndpointId, _ := lr.Data[i].GetString("id")
			address, _ := lr.Data[i].GetString("intranet_ip_addr")
			if proxyEndpointIds.Has(proxyEndpointId) {
				continue
			}
			pes = append(pes, sProxyEndpoint{proxyEndpointId, address})
			proxyEndpointIds.Insert(proxyEndpointId)
		}
	}
	filter := jsonutils.NewDict()
	filter.Set("vpc_id", jsonutils.NewString(info.VpcId))
	lr, err := cloudproxy.ProxyEndpoints.List(session, filter)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list proxy endpoint in vpc %q", info.VpcId)
	}
	for i := range lr.Data {
		proxyEndpointId, _ := lr.Data[i].GetString("id")
		address, _ := lr.Data[i].GetString("intranet_ip_addr")
		if proxyEndpointIds.Has(proxyEndpointId) {
			continue
		}
		pes = append(pes, sProxyEndpoint{proxyEndpointId, address})
		proxyEndpointIds.Insert(proxyEndpointId)
	}
	return pes, nil
}

func getServerInfo(ctx context.Context, serverId string) (sServerInfo, error) {
	// check server
	session := auth.GetAdminSession(ctx, "", "")
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

func convertInfluxdbUrl(ctx context.Context, pUrl string, endpointId string) (port int64, recycle func() error, err error) {
	session := auth.AdminSessionWithInternal(ctx, "", "", "")
	filter := jsonutils.NewDict()
	filter.Set("proxy_endpoint_id", jsonutils.NewString(endpointId))
	filter.Set("opaque", jsonutils.NewString(pUrl))
	filter.Set("scope", jsonutils.NewString("system"))
	lr, err := cloudproxy.Forwards.List(session, filter)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to list forward")
	}
	if len(lr.Data) > 0 {
		port, _ = lr.Data[0].Int("bind_port")
	} else {
		var rUrl *url.URL
		rUrl, err = url.Parse(pUrl)
		if err != nil {
			err = errors.Wrap(err, "invalid influxdbUrl?")
			return
		}
		// create one
		createP := jsonutils.NewDict()
		createP.Set("proxy_endpoint", jsonutils.NewString(endpointId))
		createP.Set("type", jsonutils.NewString(proxy_api.FORWARD_TYPE_REMOTE))
		createP.Set("remote_addr", jsonutils.NewString(rUrl.Hostname()))
		createP.Set("remote_port", jsonutils.NewString(rUrl.Port()))
		createP.Set("generate_name", jsonutils.NewString("influxdb proxy"))
		createP.Set("opaque", jsonutils.NewString(pUrl))
		var forward jsonutils.JSONObject
		forward, err = cloudproxy.Forwards.Create(session, createP)
		if err != nil {
			err = errors.Wrapf(err, "unable to create forward with create params %s", createP.String())
			return
		}
		forwardId, _ := forward.GetString("id")
		recycle = func() error {
			_, err := cloudproxy.Forwards.Delete(session, forwardId, nil)
			return err
		}
		port, _ = forward.Int("bind_port")
	}
	return
}

func checkProxyEndpoint(ctx context.Context, influxdbUrl, proxyEndpointId, address string, host *ansible_api.AnsibleHost) (string, error) {
	port, recycle, err := convertInfluxdbUrl(ctx, influxdbUrl, proxyEndpointId)
	if err != nil {
		return "", err
	}
	nUrl := fmt.Sprintf("https://%s:%d", address, port)
	ok, err := checkUrl(ctx, nUrl, host)
	if err != nil {
		return "", errors.Wrapf(err, "check url %q", nUrl)
	}
	if !ok {
		if recycle != nil {
			err := recycle()
			if err != nil {
				return "", errors.Wrapf(err, "unble to recycle remote forward of proxyEndpoint %s", proxyEndpointId)
			}
		}
		return "", nil
	}
	return nUrl, nil
}

func findValidInfluxdbUrl(ctx context.Context, influxdbUrl, proxyEndpointId string, info sServerInfo, host *ansible_api.AnsibleHost) (string, error) {
	if len(proxyEndpointId) > 0 {
		pes, err := proxyEndpoints(ctx, proxyEndpointId, info)
		if err != nil {
			return "", err
		}
		url, err := checkProxyEndpoint(ctx, influxdbUrl, proxyEndpointId, pes[0].Address, host)
		if err != nil {
			return "", err
		}
		if len(url) > 0 {
			return url, nil
		}
	}
	pes, err := proxyEndpoints(ctx, "", info)
	if err != nil {
		return "", err
	}
	for _, pe := range pes {
		url, err := checkProxyEndpoint(ctx, influxdbUrl, pe.Id, pe.Address, host)
		if err != nil {
			return "", err
		}
		if len(url) > 0 {
			return url, nil
		}
	}
	// check direct
	ok, err := checkUrl(ctx, influxdbUrl, host)
	if err != nil {
		return "", err
	}
	if ok {
		return influxdbUrl, nil
	}
	return "", nil
}

func checkUrl(ctx context.Context, url string, host *ansible_api.AnsibleHost) (bool, error) {
	session := auth.GetAdminSession(ctx, "", "")
	ahost := ansible.Host{}
	ahost.Name = host.IP
	ahost.Vars = map[string]string{
		"ansible_port": fmt.Sprintf("%d", host.Port),
		"ansible_user": host.User,
	}
	mod := ansible.Module{
		Name: "uri",
		Args: []string{
			fmt.Sprintf("url=%s/ping", url),
			"method=GET",
			"status_code=204",
			"validate_certs=no",
		},
	}

	playbook := ansible.NewPlaybook()
	playbook.Inventory = ansible.Inventory{
		Hosts: []ansible.Host{
			ahost,
		},
	}
	playbook.Modules = []ansible.Module{
		mod,
	}
	apCreateInput := ansible_api.AnsiblePlaybookCreateInput{
		Name:     db.DefaultUUIDGenerator(),
		Playbook: *playbook,
	}
	apb, err := modules.AnsiblePlaybooks.Create(session, apCreateInput.JSON(apCreateInput))
	if err != nil {
		return false, errors.Wrap(err, "create ansible playbook")
	}
	id, _ := apb.GetString("id")
	defer func() {
		_, err := modules.AnsiblePlaybooks.Delete(session, id, nil)
		if err != nil {
			log.Errorf("unable to delete ansibleplaybook %s: %v", id, err)
		}
	}()
	times, waitTimes := 0, time.Second
	for times < 5 {
		time.Sleep(waitTimes)
		times++
		waitTimes += time.Second * time.Duration(times)
		apd, err := modules.AnsiblePlaybooks.GetSpecific(session, id, "status", nil)
		if err != nil {
			return false, errors.Wrapf(err, "unable to get ansibleplaybook %s status", id)
		}
		status, _ := apd.GetString("status")
		switch status {
		case ansible_api.AnsiblePlaybookStatusInit, ansible_api.AnsiblePlaybookStatusRunning:
			continue
		case ansible_api.AnsiblePlaybookStatusFailed, ansible_api.AnsiblePlaybookStatusCanceled, ansible_api.AnsiblePlaybookStatusUnknown:
			return false, nil
		case ansible_api.AnsiblePlaybookStatusSucceeded:
			return true, nil
		}
	}
	return false, nil
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

var ErrCannotReachInfluxbd = errors.Error("no suitable network to reach influxdb")

func GetArgs(ctx context.Context, serverId, proxyEndpointId string, others interface{}) (map[string]interface{}, error) {
	host, ok := others.(*ansible_api.AnsibleHost)
	if !ok {
		return nil, errors.Error("unknown others, want *AnsibleHost")
	}
	info, err := getServerInfo(ctx, serverId)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get serverInfo of server %s", serverId)
	}

	influxdbUrl, err := getInfluxdbUrl(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get influxdbUrl")
	}
	influxdbUrl, err = findValidInfluxdbUrl(ctx, influxdbUrl, proxyEndpointId, info, host)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to convertInfluxdbUrl %s", influxdbUrl)
	}
	if len(influxdbUrl) == 0 {
		return nil, errors.Wrap(ErrCannotReachInfluxbd, "please create usable Proxy Endpoint for server and try again")
	}
	vmId := info.serverDetails.Id
	tenantId := info.serverDetails.ProjectId
	domainId := info.serverDetails.DomainId
	ret := map[string]interface{}{
		"influxdb_url":       influxdbUrl,
		"influxdb_name":      "telegraf",
		"onecloud_vm_id":     vmId,
		"onecloud_tenant_id": tenantId,
		"onecloud_domain_id": domainId,
	}
	return ret, nil
}
