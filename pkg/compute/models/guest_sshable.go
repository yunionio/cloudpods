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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	cloudproxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/sshkeys"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
	cloudproxy_module "yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/httputils"
	ssh_util "yunion.io/x/onecloud/pkg/util/ssh"
)

type GuestSshableTryData struct {
	DryRun bool

	User       string
	Host       string
	Port       int
	PrivateKey string
	PublicKey  string

	MethodTried []compute_api.GuestSshableMethodData
}

func (tryData *GuestSshableTryData) AddMethodTried(tryMethodData compute_api.GuestSshableMethodData) {
	tryData.MethodTried = append(tryData.MethodTried, tryMethodData)
}

func (tryData *GuestSshableTryData) outputJSON() jsonutils.JSONObject {
	out := compute_api.GuestSshableOutput{
		User:      tryData.User,
		PublicKey: tryData.PublicKey,

		MethodTried: tryData.MethodTried,
	}
	outJSON := jsonutils.Marshal(out)
	return outJSON
}

func (guest *SGuest) AllowGetDetailsSshable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return db.IsProjectAllowGetSpec(userCred, guest, "sshable")
}

func (guest *SGuest) GetDetailsSshable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if guest.Status != compute_api.VM_RUNNING {
		return nil, httperrors.NewBadRequestError("server sshable state can only be checked when in running state")
	}

	tryData := &GuestSshableTryData{
		User: "cloudroot",
	}

	// - get admin key
	privateKey, publicKey, err := sshkeys.GetSshAdminKeypair(ctx)
	if err != nil {
		return nil, httperrors.NewInternalServerError("fetch ssh private key: %v", err)
	}
	tryData.PrivateKey = privateKey
	tryData.PublicKey = publicKey

	if err := guest.sshableTryEach(ctx, userCred, tryData); err != nil {
		return nil, err
	}

	{
		sshable := false
		for i := range tryData.MethodTried {
			if tryData.MethodTried[i].Sshable {
				sshable = true
				break
			}
		}
		if _, err := db.Update(guest, func() error {
			guest.SshableLastState = tristate.NewFromBool(sshable)
			return nil
		}); err != nil {
			log.Errorf("update guest %s(%s) sshable_last_state to %v: %v", guest.Name, guest.Id, sshable, err)
		}
	}

	return tryData.outputJSON(), nil
}

func (guest *SGuest) sshableTryEach(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	tryData *GuestSshableTryData,
) error {
	gns, err := guest.GetNetworks("")
	if err != nil {
		return httperrors.NewInternalServerError("fetch network interface information: %v", err)
	}
	type gnInfo struct {
		guestNetwork *SGuestnetwork
		network      *SNetwork
		vpc          *SVpc
	}
	var gnInfos []gnInfo
	for i := range gns {
		gn := &gns[i]
		network := gn.GetNetwork()
		if network == nil {
			continue
		}
		vpc := network.GetVpc()
		if vpc == nil {
			continue
		}
		if vpc.Id == compute_api.DEFAULT_VPC_ID {
			//   - vpc_id == "default"
			if ok := guest.sshableTryDefaultVPC(ctx, tryData, gn); ok {
				return nil
			}
		} else {
			gnInfos = append(gnInfos, gnInfo{
				guestNetwork: gn,
				network:      network,
				vpc:          vpc,
			})
		}
	}

	//   - check eip
	if eip, err := guest.GetEipOrPublicIp(); err == nil && eip != nil {
		if ok := guest.sshableTryEip(ctx, tryData, eip); ok {
			return nil
		}
	}

	sess := auth.GetSession(ctx, userCred, "", "")
	//   - check existing proxy forward
	proxyforwardTried := false
	for i := range gnInfos {
		gnInfo := &gnInfos[i]
		gn := gnInfo.guestNetwork
		port := 22
		input := &cloudproxy_api.ForwardListInput{
			Type:       cloudproxy_api.FORWARD_TYPE_LOCAL,
			RemoteAddr: gn.IpAddr,
			RemotePort: &port,
			Opaque:     guest.Id,
		}
		params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
		params.Set("details", jsonutils.JSONTrue)
		res, err := cloudproxy_module.Forwards.List(sess, params)
		if err != nil {
			log.Warningf("list cloudproxy forwards: %v", err)
			continue
		}
		proxyforwardTried = len(res.Data) != 0
		for _, data := range res.Data {
			var fwd cloudproxy_api.ForwardDetails
			if err := data.Unmarshal(&fwd); err != nil {
				log.Warningf("unmarshal cloudproxy forward list data: %v", err)
				continue
			}
			if ok := guest.sshableTryForward(ctx, tryData, &fwd); ok {
				return nil
			}
		}
	}
	if !proxyforwardTried {
		//   - create and use new proxy forward
		fwdCreateInput := cloudproxy_api.ForwardCreateFromServerInput{
			ServerId:   guest.Id,
			Type:       cloudproxy_api.FORWARD_TYPE_LOCAL,
			RemotePort: 22,
		}
		fwdCreateParams := jsonutils.Marshal(fwdCreateInput)
		res, err := cloudproxy_module.Forwards.PerformClassAction(sess, "create-from-server", fwdCreateParams)
		if err == nil {
			var fwd cloudproxy_api.ForwardDetails
			if err := res.Unmarshal(&fwd); err == nil {
				if ok := guest.sshableTryForward(ctx, tryData, &fwd); ok {
					return nil
				}
			}
		} else {
			var reason string
			if jce, ok := err.(*httputils.JSONClientError); ok {
				reason = jce.Details
			} else {
				reason = err.Error()
			}
			tryData.AddMethodTried(compute_api.GuestSshableMethodData{
				Method: compute_api.MethodProxyForward,
				Reason: reason,
			})
		}
	}

	//   - existing dnat rule
	for i := range gnInfos {
		gnInfo := &gnInfos[i]
		gn := gnInfo.guestNetwork
		vpc := gnInfo.vpc

		natgwq := NatGatewayManager.Query().SubQuery()
		q := NatDEntryManager.Query().
			Equals("internal_ip", gn.IpAddr).
			Equals("internal_port", 22).
			Equals("ip_protocol", "tcp")
		q = q.Join(natgwq, sqlchemy.AND(
			sqlchemy.In(natgwq.Field("vpc_id"), vpc.Id),
			sqlchemy.Equals(natgwq.Field("id"), q.Field("natgateway_id")),
		))

		var dnats []SNatDEntry
		if err := db.FetchModelObjects(NatDEntryManager, q, &dnats); err != nil {
			log.Warningf("query dnat to ssh service: %v", err)
			continue
		}
		for j := range dnats {
			dnat := &dnats[j]
			if ok := guest.sshableTryDnat(ctx, tryData, dnat); ok {
				return nil
			}
		}
	}

	return nil
}

func (guest *SGuest) sshableTryDnat(
	ctx context.Context,
	tryData *GuestSshableTryData,
	dnat *SNatDEntry,
) bool {
	methodData := compute_api.GuestSshableMethodData{
		Method: compute_api.MethodDNAT,
		Host:   dnat.ExternalIP,
		Port:   dnat.ExternalPort,
	}
	return guest.sshableTry(
		ctx, tryData, methodData,
	)
}

func (guest *SGuest) sshableTryForward(
	ctx context.Context,
	tryData *GuestSshableTryData,
	fwd *cloudproxy_api.ForwardDetails,
) bool {
	if fwd.BindAddr != "" && fwd.BindPort > 0 {
		methodData := compute_api.GuestSshableMethodData{
			Method: compute_api.MethodProxyForward,
			Host:   fwd.BindAddr,
			Port:   fwd.BindPort,
			ForwardDetails: compute_api.ForwardDetails{
				ProxyAgentId:    fwd.ProxyAgentId,
				ProxyEndpointId: fwd.ProxyEndpointId,
			},
		}
		return guest.sshableTry(
			ctx, tryData, methodData,
		)
	}
	return false
}

func (guest *SGuest) sshableTryEip(
	ctx context.Context,
	tryData *GuestSshableTryData,
	eip *SElasticip,
) bool {
	methodData := compute_api.GuestSshableMethodData{
		Method: compute_api.MethodEIP,
		Host:   eip.IpAddr,
		Port:   22,
	}
	return guest.sshableTry(
		ctx, tryData, methodData,
	)
}

func (guest *SGuest) sshableTryDefaultVPC(
	ctx context.Context,
	tryData *GuestSshableTryData,
	gn *SGuestnetwork,
) bool {
	methodData := compute_api.GuestSshableMethodData{
		Method: compute_api.MethodDirect,
		Host:   gn.IpAddr,
		Port:   22,
	}
	return guest.sshableTry(
		ctx, tryData, methodData,
	)
}

func (guest *SGuest) sshableTry(
	ctx context.Context,
	tryData *GuestSshableTryData,
	methodData compute_api.GuestSshableMethodData,
) bool {
	if tryData.DryRun {
		tryData.AddMethodTried(methodData)
		return true
	}

	ctx, _ = context.WithTimeout(ctx, 7*time.Second)
	conf := ssh_util.ClientConfig{
		Username:   tryData.User,
		Host:       methodData.Host,
		Port:       methodData.Port,
		PrivateKey: tryData.PrivateKey,
	}
	ok := false
	if client, err := conf.ConnectContext(ctx); err == nil {
		defer client.Close()
		methodData.Sshable = true
		ok = true
	} else {
		methodData.Reason = err.Error()
	}
	tryData.AddMethodTried(methodData)
	return ok
}

func (guest *SGuest) AllowPerformMakeSshable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return db.IsProjectAllowGetSpec(userCred, guest, "make-sshable")
}

func (guest *SGuest) PerformMakeSshable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input compute_api.GuestMakeSshableInput,
) (output compute_api.GuestMakeSshableOutput, err error) {
	if guest.Status != compute_api.VM_RUNNING {
		return output, httperrors.NewBadRequestError("make-sshable can only be performed when in running state")
	}

	if input.User == "" {
		return output, httperrors.NewBadRequestError("missing username")
	}
	if input.PrivateKey == "" && input.Password == "" {
		return output, httperrors.NewBadRequestError("private_key and password cannot both be empty")
	}

	_, projectPublicKey, err := sshkeys.GetSshProjectKeypair(ctx, guest.ProjectId)
	if err != nil {
		return output, httperrors.NewInternalServerError("fetch project public key: %v", err)
	}
	_, adminPublicKey, err := sshkeys.GetSshAdminKeypair(ctx)
	if err != nil {
		return output, httperrors.NewInternalServerError("fetch admin public key: %v", err)
	}

	tryData := &GuestSshableTryData{
		DryRun: true,
	}
	if err := guest.sshableTryEach(ctx, userCred, tryData); err != nil {
		return output, httperrors.NewNotAcceptableError("searching for usable ssh address: %v", err)
	} else if len(tryData.MethodTried) == 0 {
		return output, httperrors.NewNotAcceptableError("no usable ssh address")
	}

	host := ansible.Host{
		Name: guest.Name,
	}
	host.SetVar("ansible_user", input.User)
	host.SetVar("ansible_host", tryData.MethodTried[0].Host)
	host.SetVar("ansible_port", fmt.Sprintf("%d", tryData.MethodTried[0].Port))
	host.SetVar("ansible_become", "yes")
	pb := &ansible.Playbook{
		Inventory: ansible.Inventory{
			Hosts: []ansible.Host{host},
		},
		Modules: []ansible.Module{
			{
				Name: "group",
				Args: []string{
					"name=cloudroot",
					"state=present",
				},
			},
			{
				Name: "user",
				Args: []string{
					"name=cloudroot",
					"state=present",
					"group=cloudroot",
				},
			},
			{
				Name: "authorized_key",
				Args: []string{
					"user=cloudroot",
					"state=present",
					fmt.Sprintf("key=%q", adminPublicKey),
				},
			},
			{
				Name: "authorized_key",
				Args: []string{
					"user=cloudroot",
					"state=present",
					fmt.Sprintf("key=%q", projectPublicKey),
				},
			},
			{
				Name: "lineinfile",
				Args: []string{
					"dest=/etc/sudoers",
					"state=present",
					fmt.Sprintf("regexp=%q", "^cloudroot "),
					fmt.Sprintf("line=%q", "cloudroot ALL=(ALL) NOPASSWD: ALL"),
					fmt.Sprintf("validate=%q", "visudo -cf %s"),
				},
			},
		},
	}
	if input.PrivateKey != "" {
		pb.PrivateKey = []byte(input.PrivateKey)
	} else if input.Password != "" {
		host.SetVar("ansible_password", input.Password)
	}

	cliSess := auth.GetSession(ctx, userCred, "", "")
	pbId := ""
	pbName := "make-sshable-" + guest.Name
	pbModel, err := mcclient_modules.AnsiblePlaybooks.UpdateOrCreatePbModel(
		ctx, cliSess, pbId, pbName, pb,
	)
	if err != nil {
		return output, httperrors.NewGeneralError(err)
	}

	output = compute_api.GuestMakeSshableOutput{
		AnsiblePlaybookId: pbModel.Id,
	}
	return output, nil
}

func (guest *SGuest) AllowGetDetailsMakeSshableCmd(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return db.IsProjectAllowGetSpec(userCred, guest, "make-sshable-cmd")
}

func (guest *SGuest) GetDetailsMakeSshableCmd(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (output compute_api.GuestMakeSshableCmdOutput, err error) {
	_, projectPublicKey, err := sshkeys.GetSshProjectKeypair(ctx, guest.ProjectId)
	if err != nil {
		return output, httperrors.NewInternalServerError("fetch project public key: %v", err)
	}
	_, adminPublicKey, err := sshkeys.GetSshAdminKeypair(ctx)
	if err != nil {
		return output, httperrors.NewInternalServerError("fetch admin public key: %v", err)
	}

	varVals := [][2]string{
		[2]string{"user", "cloudroot"},
		[2]string{"adminpub", strings.TrimSpace(adminPublicKey)},
		[2]string{"projpub", strings.TrimSpace(projectPublicKey)},
	}
	shellCmd := ""
	for i := range varVals {
		varVal := varVals[i]
		shellCmd += fmt.Sprintf("%s=%q\n", varVal[0], varVal[1])
	}

	shellCmd += `
group="$user"
sshdir="/home/$user/.ssh"
keyfile="$sshdir/authorized_keys"
`
	shellCmd += `
id -g "$group" &>/dev/null || groupadd "$group"
id -u "$user"  &>/dev/null || useradd --create-home --gid "$group" "$user"
mkdir -p "$sshdir"
grep -q -F "$adminpub" "$keyfile" &>/dev/null || echo "$adminpub" >>"$keyfile"
grep -q -F "$projpub" "$keyfile"  &>/dev/null || echo "$projpub" >>"$keyfile"
chown -R "$user:$group" "$sshdir"
chmod -R 700 "$sshdir"
chmod -R 600 "$keyfile"

if ! grep -q "^$user " /etc/sudoers; then
  echo "$user ALL=(ALL) NOPASSWD: ALL" | EDITOR='tee -a' visudo
fi
`
	output = compute_api.GuestMakeSshableCmdOutput{
		ShellCmd: shellCmd,
	}
	return output, nil
}
