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
	"net"
	"time"

	"yunion.io/x/log"

	cloudproxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	ansible_modules "yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
	"yunion.io/x/onecloud/pkg/util/ansible"
	ssh_util "yunion.io/x/onecloud/pkg/util/ssh"
)

func (proxyendpoint *SProxyEndpoint) remoteCheckMake(ctx context.Context, userCred mcclient.TokenCredential) error {
	ctx, _ = context.WithTimeout(ctx, 7*time.Second)
	conf := ssh_util.ClientConfig{
		Username:   proxyendpoint.User,
		Host:       proxyendpoint.Host,
		Port:       proxyendpoint.Port,
		PrivateKey: proxyendpoint.PrivateKey,
	}
	client, err := conf.ConnectContext(ctx)
	if err != nil {
		return httperrors.NewBadRequestError("ssh connect failed: %v", err)
	}
	defer client.Close()

	if err := proxyendpoint.remoteConfigure(ctx, userCred); err != nil {
		return err
	}

	// total wait time
	tmo := time.NewTimer(23 * time.Second)
	tmoErr := httperrors.NewOutOfResourceError("timeout testing remote config.  please retry later")

	// find a remote port for bind and listen
	var (
		listenAddr string
		listener   net.Listener
	)
	portMin := cloudproxy_api.BindPortMax + 1
	portTotal := 65535 - cloudproxy_api.BindPortMax
	portReqStart := rand.Intn(portTotal)
	for portInc := portReqStart; ; {
		port := portMin + portInc
		addr := net.JoinHostPort(proxyendpoint.IntranetIpAddr, fmt.Sprintf("%d", port))
		listener_, err := client.Listen("tcp", addr)
		if err == nil {
			// we assume the port is not occupied by intranet addr,
			// even though there is the possibility that the above
			// test only happen against 127.0.0.1
			listenAddr = addr
			listener = listener_
			break
		}
		select {
		case <-tmo.C:
			return tmoErr
		default:
		}
		portInc += 1
		if portInc == portTotal {
			portInc = 0
		}
		if portInc == portReqStart {
			return httperrors.NewOutOfResourceError("no available port for bind test")
		}
	}

	// test for good news
	for {
		if listener != nil {
			if conn, err := client.Dial("tcp", listenAddr); err == nil {
				conn.Close()
				listener.Close()
				return nil
			}
			listener.Close()
		}

		var err error
		listener, err = client.Listen("tcp", listenAddr)
		if err != nil {
			log.Warningf("retry ssh listen %s: %v", listenAddr, err)
		}
		select {
		case <-tmo.C:
			return tmoErr
		case <-time.After(2 * time.Second):
		}
	}
	// return httperrors.NewConflictError("remote sshd_config may have a problem with GatewayPorts")
}

func (proxyendpoint *SProxyEndpoint) remoteConfigure(ctx context.Context, userCred mcclient.TokenCredential) error {
	host := ansible.Host{
		Name: "0.0.0.0", // ansibleserver requires this field to be either ip_addr, or name of guest, host
	}
	host.SetVar("ansible_user", proxyendpoint.User)
	host.SetVar("ansible_host", proxyendpoint.Host)
	host.SetVar("ansible_port", fmt.Sprintf("%d", proxyendpoint.Port))
	host.SetVar("ansible_become", "yes")
	pb := &ansible.Playbook{
		PrivateKey: []byte(proxyendpoint.PrivateKey),
		Inventory: ansible.Inventory{
			Hosts: []ansible.Host{host},
		},
		Modules: []ansible.Module{
			{
				Name: "lineinfile",
				Args: []string{
					"dest=/etc/ssh/sshd_config",
					"state=present",
					fmt.Sprintf("regexp=%q", "^GatewayPorts "),
					fmt.Sprintf("line=%q", "GatewayPorts clientspecified"),
					fmt.Sprintf("validate=%q", "sshd -T -f %s"),
				},
			},
			{
				Name: "service",
				Args: []string{
					"name=sshd",
					"state=restarted",
				},
			},
			{
				Name: "service",
				Args: []string{
					"name=ssh", // ubuntu uses this name.  It can fail for centos, but we do not care
					"state=restarted",
				},
			},
		},
	}

	cliSess := auth.GetSession(ctx, userCred, "", "")
	pbId := ""
	pbName := "pe-remote-configure-" + proxyendpoint.Name
	_, err := ansible_modules.AnsiblePlaybooks.UpdateOrCreatePbModel(
		ctx, cliSess, pbId, pbName, pb,
	)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	return nil
}
