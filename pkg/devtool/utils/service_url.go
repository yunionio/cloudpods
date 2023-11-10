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
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
	ansible_api "yunion.io/x/onecloud/pkg/apis/ansible"
	proxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	ansible_modules "yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
	"yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type Service struct {
	Name string
	Url  string
}

func serviceComplete(serviceName, address string, port int) (url, checkUrl string, expectedCode int) {
	switch serviceName {
	case apis.SERVICE_TYPE_INFLUXDB, apis.SERVICE_TYPE_VICTORIA_METRICS:
		return fmt.Sprintf("https://%s:%d", address, port), fmt.Sprintf("https://%s:%d/ping", address, port), 204
	case "repo":
		return fmt.Sprintf("http://%s:%d", address, port), fmt.Sprintf("http://%s:%d", address, port), 200
	default:
		return fmt.Sprintf("http://%s:%d", address, port), fmt.Sprintf("http://%s:%d", address, port), 200
	}
}

func serviceComplete2(service Service) (completeUrl string, expectedCode int) {
	switch service.Name {
	case apis.SERVICE_TYPE_INFLUXDB, apis.SERVICE_TYPE_VICTORIA_METRICS:
		return fmt.Sprintf("%s/ping", service.Url), 204
	case "repo":
		return service.Url, 200
	default:
		return service.Url, 200
	}
}

func proxyEndpoints(ctx context.Context, proxyEndpointId string, info sServerInfo) ([]sProxyEndpoint, error) {
	pes := make([]sProxyEndpoint, 0)
	session := auth.GetAdminSession(ctx, "")
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
		filter.Set("scope", jsonutils.NewString("system"))
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
	filter.Set("scope", jsonutils.NewString("system"))
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

func serviceUrlDirect(ctx context.Context, service Service, proxyEndpointId string, info sServerInfo, host *ansible_api.AnsibleHost) (string, error) {
	url, code := serviceComplete2(service)
	ok, err := checkUrl(ctx, url, code, host)
	if err != nil {
		return "", err
	}
	if ok {
		return service.Url, nil
	}
	return "", nil
}

func GetServerInfo(ctx context.Context, serverId string) (sServerInfo, error) {
	// check server
	session := auth.GetAdminSession(ctx, "")
	data, err := compute.Servers.Get(session, serverId, nil)
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

func serviceUrlViaProxyEndpoint(ctx context.Context, service Service, proxyEndpointId string, info sServerInfo, host *ansible_api.AnsibleHost) (string, error) {
	if len(proxyEndpointId) > 0 {
		pes, err := proxyEndpoints(ctx, proxyEndpointId, info)
		if err != nil {
			return "", err
		}
		url, err := checkProxyEndpoint(ctx, service, proxyEndpointId, pes[0].Address, host)
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
		url, err := checkProxyEndpoint(ctx, service, pe.Id, pe.Address, host)
		if err != nil {
			return "", err
		}
		if len(url) > 0 {
			return url, nil
		}
	}
	return "", nil
}

func FindValidServiceUrl(ctx context.Context, service Service, proxyEndpointId string, info sServerInfo, host *ansible_api.AnsibleHost) (string, error) {
	findFuncs := []func(ctx context.Context, service Service, proxyEndpointId string, info sServerInfo, host *ansible_api.AnsibleHost) (string, error){}
	if info.serverDetails.Hypervisor == comapi.HYPERVISOR_KVM || info.serverDetails.Hypervisor == comapi.HYPERVISOR_BAREMETAL {
		findFuncs = append(findFuncs, serviceUrlDirect, serviceUrlViaProxyEndpoint)
	} else {
		findFuncs = append(findFuncs, serviceUrlViaProxyEndpoint, serviceUrlDirect)
	}
	for _, find := range findFuncs {
		url, err := find(ctx, service, proxyEndpointId, info, host)
		if err != nil {
			return "", err
		}
		if len(url) > 0 {
			return url, nil
		}
	}
	return "", nil
}

func checkProxyEndpoint(ctx context.Context, service Service, proxyEndpointId, address string, host *ansible_api.AnsibleHost) (string, error) {
	port, recycle, err := convertServiceUrl(ctx, service, proxyEndpointId)
	if err != nil {
		return "", err
	}
	url, cUrl, code := serviceComplete(service.Name, address, int(port))
	ok, err := checkUrl(ctx, cUrl, code, host)
	if err != nil {
		return "", errors.Wrapf(err, "check url %q", cUrl)
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
	return url, nil
}

type sServerInfo struct {
	ServerId      string
	VpcId         string
	NetworkIds    []string
	serverDetails *comapi.ServerDetails
}

var serviceUrls map[string]string = map[string]string{}

func GetServiceUrl(ctx context.Context, serviceName string) (string, error) {
	if url, ok := serviceUrls[serviceName]; ok {
		return url, nil
	}
	session := auth.GetAdminSession(ctx, "")
	params := jsonutils.NewDict()
	params.Set("interface", jsonutils.NewString("public"))
	params.Set("service", jsonutils.NewString(serviceName))
	ret, err := identity.EndpointsV3.List(session, params)
	if err != nil {
		return "", err
	}
	log.Infof("params to list endpoint: %v", params)
	log.Infof("ret to list endpoint: %s", jsonutils.Marshal(ret))
	if len(ret.Data) == 0 {
		return "", fmt.Errorf("no sucn endpoint with 'internal' interface and 'influxdb' service")
	}
	url, _ := ret.Data[0].GetString("url")
	serviceUrls[serviceName] = url
	return url, nil
}

func convertServiceUrl(ctx context.Context, service Service, endpointId string) (port int64, recycle func() error, err error) {
	session := auth.AdminSessionWithInternal(ctx, "", "")
	filter := jsonutils.NewDict()
	filter.Set("proxy_endpoint_id", jsonutils.NewString(endpointId))
	filter.Set("opaque", jsonutils.NewString(service.Url))
	filter.Set("scope", jsonutils.NewString("system"))
	lr, err := cloudproxy.Forwards.List(session, filter)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to list forward")
	}
	var forwardId string
	var lastSeen string
	if len(lr.Data) > 0 {
		port, _ = lr.Data[0].Int("bind_port")
		forwardId, _ = lr.Data[0].GetString("id")
		lastSeen, _ = lr.Data[0].GetString("last_seen")
	} else {
		var rUrl *url.URL
		rUrl, err = url.Parse(service.Url)
		if err != nil {
			err = errors.Wrap(err, "invalid serviceUrl?")
			return
		}
		// create one
		createP := jsonutils.NewDict()
		createP.Set("proxy_endpoint", jsonutils.NewString(endpointId))
		createP.Set("type", jsonutils.NewString(proxy_api.FORWARD_TYPE_REMOTE))
		createP.Set("remote_addr", jsonutils.NewString(rUrl.Hostname()))
		createP.Set("remote_port", jsonutils.NewString(rUrl.Port()))
		createP.Set("generate_name", jsonutils.NewString(service.Name+" proxy"))
		createP.Set("opaque", jsonutils.NewString(service.Url))
		var forward jsonutils.JSONObject
		forward, err = cloudproxy.Forwards.Create(session, createP)
		if err != nil {
			err = errors.Wrapf(err, "unable to create forward with create params %s", createP.String())
			return
		}
		forwardId, _ = forward.GetString("id")
		lastSeen, _ = forward.GetString("last_seen")
		recycle = func() error {
			_, err := cloudproxy.Forwards.Delete(session, forwardId, nil)
			return err
		}
		port, _ = forward.Int("bind_port")
	}
	// wait forward last seen not empty
	times, waitTime := 0, time.Second
	var data jsonutils.JSONObject
	for lastSeen == "" && times < 10 {
		time.Sleep(waitTime)
		times += 1
		waitTime += time.Second * time.Duration(times)
		data, err = cloudproxy.Forwards.GetSpecific(session, forwardId, "lastseen", nil)
		if err != nil {
			err = errors.Wrapf(err, "unable to check last_seen for forward %s", forwardId)
			return
		}
		log.Infof("data of last seen: %s", data)
		lastSeen, _ = data.GetString("last_seen")
	}
	if lastSeen == "" {
		err = errors.Wrapf(err, "last_seen of forward %s always is empty, something wrong", forwardId)
	}
	return
}

func checkUrl(ctx context.Context, completeUrl string, expectedCode int, host *ansible_api.AnsibleHost) (bool, error) {
	session := auth.GetAdminSession(ctx, "")
	ahost := ansible.Host{}
	ahost.Name = host.IP
	ahost.Vars = map[string]string{
		"ansible_port": fmt.Sprintf("%d", host.Port),
		"ansible_user": host.User,
	}
	if host.OsType == "Windows" {
		ahost.Vars["ansible_password"] = host.Password
		ahost.Vars["ansible_connection"] = "winrm"
		ahost.Vars["ansible_winrm_server_cert_validation"] = "ignore"
		ahost.Vars["ansible_winrm_transport"] = "ntlm"
	}
	modulename := "uri"
	if host.OsType == "Windows" {
		modulename = "ansible.windows.win_uri"
	}
	mod := ansible.Module{
		Name: modulename,
		Args: []string{
			fmt.Sprintf("url=%s", completeUrl),
			"method=GET",
			fmt.Sprintf("status_code=%d", expectedCode),
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
	apb, err := ansible_modules.AnsiblePlaybooks.Create(session, apCreateInput.JSON(apCreateInput))
	if err != nil {
		return false, errors.Wrap(err, "create ansible playbook")
	}
	id, _ := apb.GetString("id")
	defer func() {
		_, err := ansible_modules.AnsiblePlaybooks.Delete(session, id, nil)
		if err != nil {
			log.Errorf("unable to delete ansibleplaybook %s: %v", id, err)
		}
	}()
	times, waitTimes := 0, time.Second
	for times < 10 {
		time.Sleep(waitTimes)
		times++
		waitTimes += time.Second * time.Duration(times)
		apd, err := ansible_modules.AnsiblePlaybooks.GetSpecific(session, id, "status", nil)
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
