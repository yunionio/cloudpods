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
	"fmt"
	"net"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	cloudproxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
)

var ErrServerNotSshable = errors.Error("server is not sshable")

type SSHable struct {
	Ok     bool
	Reason string

	User string
	Host string
	Port int

	ServerName       string
	ServerHypervisor string

	ProxyEndpointId string
	ProxyAgentId    string
	ProxyForwardId  string
}

func checkSshableForOtherCloud(session *mcclient.ClientSession, serverId string) (SSHable, error) {
	data, err := modules.Servers.GetSpecific(session, serverId, "sshable", nil)
	if err != nil {
		return SSHable{}, errors.Wrapf(err, "unable to get sshable info of server %s", serverId)
	}
	log.Infof("data to sshable: %v", data)
	var sshableOutput comapi.GuestSshableOutput
	err = data.Unmarshal(&sshableOutput)
	if err != nil {
		return SSHable{}, errors.Wrapf(err, "unable to marshal output of server sshable: %s", data)
	}
	sshable := SSHable{
		User: sshableOutput.User,
	}
	reasons := make([]string, 0, len(sshableOutput.MethodTried))
	for _, methodTried := range sshableOutput.MethodTried {
		if !methodTried.Sshable {
			reasons = append(reasons, methodTried.Reason)
			continue
		}
		sshable.Ok = true
		switch methodTried.Method {
		case comapi.MethodDirect, comapi.MethodEIP, comapi.MethodDNAT:
			sshable.Host = methodTried.Host
			sshable.Port = methodTried.Port
		case comapi.MethodProxyForward:
			sshable.ProxyAgentId = methodTried.ForwardDetails.ProxyAgentId
			sshable.ProxyEndpointId = methodTried.ForwardDetails.ProxyEndpointId
		}
	}
	if !sshable.Ok {
		sshable.Reason = strings.Join(reasons, "; ")
	}
	return sshable, nil
}

func checkSshableForYunionCloud(session *mcclient.ClientSession, serverDetail *comapi.ServerDetails) (sshable SSHable, clean bool, err error) {
	if serverDetail.IPs == "" {
		err = fmt.Errorf("empty ips for server %s", serverDetail.Id)
		return
	}
	ips := strings.Split(serverDetail.IPs, ",")
	ip := strings.TrimSpace(ips[0])
	port, err := getServerSshport(session, serverDetail.Id)
	if err != nil {
		err = errors.Wrapf(err, "unable to get ssh port of server %s", serverDetail.Id)
		return
	}
	if serverDetail.Hypervisor == comapi.HYPERVISOR_BAREMETAL || serverDetail.VpcId == "" || serverDetail.VpcId == comapi.DEFAULT_VPC_ID {
		sshable = SSHable{
			Ok:   true,
			User: "cloudroot",
			Host: ip,
			Port: port,
		}
		return
	}
	lfParams := jsonutils.NewDict()
	lfParams.Set("proto", jsonutils.NewString("tcp"))
	lfParams.Set("port", jsonutils.NewInt(int64(port)))
	lfParams.Set("addr", jsonutils.NewString(ip))
	data, err := modules.Servers.PerformAction(session, serverDetail.Id, "list-forward", lfParams)
	if err != nil {
		err = errors.Wrapf(err, "unable to List Forward for server %s", serverDetail.Id)
		return
	}
	var openForward bool
	var forwards []jsonutils.JSONObject
	if !data.Contains("forwards") {
		openForward = true
	} else {
		forwards, err = data.GetArray("forwards")
		if err != nil {
			err = errors.Wrap(err, "parse response of List Forward")
			return
		}
		openForward = len(forwards) == 0
	}

	var forward jsonutils.JSONObject
	if openForward {
		forward, err = modules.Servers.PerformAction(session, serverDetail.Id, "open-forward", lfParams)
		if err != nil {
			err = errors.Wrapf(err, "unable to Open Forward for server %s", serverDetail.Id)
			return
		}
		clean = true
	} else {
		forward = forwards[0]
	}
	proxyAddr, _ := forward.GetString("proxy_addr")
	proxyPort, _ := forward.Int("proxy_port")
	// register
	sshable = SSHable{
		Ok:   true,
		User: "cloudroot",
		Host: proxyAddr,
		Port: int(proxyPort),
	}
	return
}

func CheckSSHable(session *mcclient.ClientSession, serverId string) (sshable SSHable, cleanFunc func() error, err error) {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	data, err := modules.Servers.GetById(session, serverId, params)
	if err != nil {
		err = errors.Wrapf(err, "unable to fetch server %s", serverId)
		return
	}
	var serverDetail comapi.ServerDetails
	err = data.Unmarshal(&serverDetail)
	if err != nil {
		err = errors.Wrapf(err, "unable to unmarshal %q to ServerDetails", data)
		return
	}

	// check sshable
	var clean bool
	if serverDetail.Hypervisor == comapi.HYPERVISOR_KVM || serverDetail.Hypervisor == comapi.HYPERVISOR_BAREMETAL {
		sshable, clean, err = checkSshableForYunionCloud(session, &serverDetail)
		if err != nil {
			return
		}
		if clean {
			cleanFunc = func() error {
				proxyAddr := sshable.Host
				proxyPort := sshable.Port
				params := jsonutils.NewDict()
				params.Set("proto", jsonutils.NewString("tcp"))
				params.Set("proxy_addr", jsonutils.NewString(proxyAddr))
				params.Set("proxy_port", jsonutils.NewInt(int64(proxyPort)))
				_, err := modules.Servers.PerformAction(session, serverDetail.Id, "close-forward", params)
				if err != nil {
					return errors.Wrapf(err, "unable to close forward(addr %q, port %d, proto %q) for server %s", proxyAddr, proxyPort, "tcp", serverDetail.Id)
				}
				return nil
			}
		}
	} else {
		sshable, err = checkSshableForOtherCloud(session, serverDetail.Id)
		if err != nil {
			return
		}
	}
	if !sshable.Ok {
		err = ErrServerNotSshable
		if len(sshable.Reason) > 0 {
			err = errors.Wrap(err, sshable.Reason)
		}
		return
	}
	sshable.ServerName = serverDetail.Name
	sshable.ServerHypervisor = serverDetail.Hypervisor
	// make sure user
	if sshable.User == "" {
		switch {
		case serverDetail.Hypervisor == comapi.HYPERVISOR_KVM:
			sshable.User = "root"
		default:
			sshable.User = "cloudroot"
		}
	}

	var forwardId string
	if len(sshable.ProxyEndpointId) == 0 {
		return
	} else {
		var sshport int
		sshport, err = getServerSshport(session, serverDetail.Id)
		if err != nil {
			err = errors.Wrapf(err, "unable to get sshport of server %s", serverDetail.Id)
		}
		// create local forward
		createP := jsonutils.NewDict()
		createP.Set("type", jsonutils.NewString(cloudproxy_api.FORWARD_TYPE_LOCAL))
		createP.Set("remote_port", jsonutils.NewInt(int64(sshport)))
		createP.Set("server_id", jsonutils.NewString(serverDetail.Id))

		var forward jsonutils.JSONObject

		forward, err = cloudproxy.Forwards.PerformClassAction(session, "create-from-server", createP)
		if err != nil {
			err = errors.Wrapf(err, "fail to create local forward from server %q", serverDetail.Id)
			return
		}

		cleanFunc = func() error {
			return clearLocalForward(session, forwardId)
		}

		var agent jsonutils.JSONObject
		port, _ := forward.Int("bind_port")
		forwardId, _ = forward.GetString("id")
		agentId, _ := forward.GetString("proxy_agent_id")
		agent, err = cloudproxy.ProxyAgents.Get(session, agentId, nil)
		if err != nil {
			err = errors.Wrapf(err, "fail to get proxy agent %q", agentId)
			return
		}
		address, _ := agent.GetString("advertise_addr")
		// check proxy forward
		if ok := ensureLocalForwardWork(address, int(port)); !ok {
			err = errors.Error("The created local forward is actually not usable")
			return
		}
		sshable.Host = address
		sshable.Port = int(port)
		sshable.ProxyForwardId = forwardId
	}
	return
}

func GetCleanFunc(session *mcclient.ClientSession, hypervisor, serverId, host, forward string, port int) func() error {
	if hypervisor == comapi.HYPERVISOR_KVM || hypervisor == comapi.HYPERVISOR_BAREMETAL {
		return func() error {
			proxyAddr := host
			proxyPort := port
			params := jsonutils.NewDict()
			params.Set("proto", jsonutils.NewString("tcp"))
			params.Set("proxy_addr", jsonutils.NewString(proxyAddr))
			params.Set("proxy_port", jsonutils.NewInt(int64(proxyPort)))
			_, err := modules.Servers.PerformAction(session, serverId, "close-forward", params)
			if err != nil {
				return errors.Wrapf(err, "unable to close forward(addr %q, port %d, proto %q) for server %s", proxyAddr, proxyPort, "tcp", serverId)
			}
			return nil
		}
	}
	return func() error {
		return clearLocalForward(session, forward)
	}
}

func getServerSshport(session *mcclient.ClientSession, serverId string) (int, error) {
	data, err := modules.Servers.GetSpecific(session, serverId, "sshport", nil)
	if err != nil {
		return 0, err
	}
	port, _ := data.Int("port")
	if port == 0 {
		port = 22
	}
	return int(port), nil
}

func clearLocalForward(s *mcclient.ClientSession, forwardId string) error {
	if len(forwardId) == 0 {
		return nil
	}
	_, err := cloudproxy.Forwards.Delete(s, forwardId, nil)
	return err
}

func ensureLocalForwardWork(host string, port int) bool {
	maxWaitTimes, wt := 10, 1*time.Second
	waitTimes := 1
	address := fmt.Sprintf("%s:%d", host, port)
	for waitTimes < maxWaitTimes {
		_, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			return true
		}
		log.Debugf("no.%d times, try to connect to %s failed: %s", waitTimes, address, err)
		time.Sleep(wt)
		waitTimes += 1
		wt += 1 * time.Second
	}
	return false
}
