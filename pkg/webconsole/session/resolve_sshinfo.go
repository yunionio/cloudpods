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

package session

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	webconsole_options "yunion.io/x/onecloud/pkg/webconsole/options"
)

type SSshConnectionInfo struct {
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	KeepUsername bool   `json:"keep_username"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	ResourceType string `json:"resource_type" choices:"host|server"`

	GuestDetails *compute_api.ServerDetails
	HostDetails  *compute_api.HostDetails
}

func ResolveServerSSHIPPortById(ctx context.Context, s *mcclient.ClientSession, id string, ip string, port int) (string, int, *compute_api.ServerDetails, error) {
	if port <= 0 {
		port = 22
	}
	return resolveServerIPPortById(ctx, s, id, ip, port)
}

func resolveServerIPPortById(ctx context.Context, s *mcclient.ClientSession, id string, ip string, port int) (string, int, *compute_api.ServerDetails, error) {
	guestDetails, err := FetchServerInfo(ctx, s, id)
	if err != nil {
		return "", 0, nil, errors.Wrap(err, "fetchServerInfo")
	}
	// list all nic of a server
	input := compute_api.GuestnetworkListInput{}
	input.ServerId = guestDetails.Id
	True := true
	input.Details = &True
	input.ServerFilterListInput.Scope = "max"
	adminSession := auth.GetAdminSession(ctx, webconsole_options.Options.Region)
	result, err := compute.Servernetworks.List(adminSession, jsonutils.Marshal(input))
	if err != nil {
		return "", 0, nil, errors.Wrap(err, "Servernetworks.List")
	}
	if result.Total == 0 {
		// not nic found!!!
		return "", 0, nil, errors.Wrap(httperrors.ErrNotFound, "no nic on server")
	}

	// find nics
	var guestNicDetails *compute_api.GuestnetworkDetails
	if result.Total == 1 {
		gn := compute_api.GuestnetworkDetails{}
		err := result.Data[0].Unmarshal(&gn)
		if err != nil {
			return "", 0, nil, errors.Wrap(err, "Unmarshal guest network info")
		}
		if len(ip) > 0 && ip != gn.EipAddr && ip != gn.IpAddr && ip != gn.Ip6Addr && ip != guestDetails.Eip {
			return "", 0, nil, errors.Wrapf(httperrors.ErrInputParameter, "ip %s not match with server", ip)
		}
		guestNicDetails = &gn
	} else {
		if len(ip) == 0 {
			return "", 0, nil, errors.Wrap(httperrors.ErrInputParameter, "must specify ip")
		}
		for _, gnJson := range result.Data {
			gn := compute_api.GuestnetworkDetails{}
			err := gnJson.Unmarshal(&gn)
			if err != nil {
				return "", 0, nil, errors.Wrap(err, "Unmarshal guest network info")
			}
			if ip == gn.EipAddr || ip == gn.IpAddr || ip == gn.Ip6Addr {
				guestNicDetails = &gn
				break
			}
		}
		if guestNicDetails == nil {
			return "", 0, nil, errors.Wrap(httperrors.ErrInputParameter, "ip specified not match with server")
		}
	}

	if len(ip) == 0 {
		// guest ip
		if len(guestNicDetails.EipAddr) > 0 {
			ip = guestNicDetails.EipAddr
		} else if len(guestNicDetails.IpAddr) > 0 {
			ip = guestNicDetails.IpAddr
		} else if len(guestNicDetails.Ip6Addr) > 0 {
			ip = guestNicDetails.Ip6Addr
		} else {
			return "", 0, nil, errors.Wrap(httperrors.ErrNotSupported, "no valid ipv4 addr")
		}
	}

	if ip == guestNicDetails.IpAddr && len(guestNicDetails.MappedIpAddr) > 0 {
		// need to do open forward
		ip, port, err = acquireForward(ctx, s, guestDetails.Id, ip, "tcp", port)
		if err != nil {
			return "", 0, nil, errors.Wrap(err, "acquireForward")
		}
	}

	return ip, port, guestDetails, nil
}

// hostHasIp checks whether the ip is exactly one of the host's addresses.
func hostHasIp(host *compute_api.HostDetails, ip string) bool {
	if host.AccessIp == ip || host.IpmiIp == ip || host.PublicIp == ip {
		return true
	}
	for _, nic := range host.NicInfo {
		if nic == nil {
			continue
		}
		if nic.IpAddr == ip || nic.Ip6Addr == ip {
			return true
		}
	}
	return false
}

// serverHasIp checks whether the ip is exactly one of the server's addresses.
func serverHasIp(server *compute_api.ServerDetails, ip string) bool {
	if server.Eip == ip {
		return true
	}
	for addr := range strings.SplitSeq(server.IPs, ",") {
		if strings.TrimSpace(addr) == ip {
			return true
		}
	}
	for _, nic := range server.Nics {
		if nic.IpAddr == ip || nic.Ip6Addr == ip {
			return true
		}
	}
	return false
}

// pickHostByIp returns the first host that carries the ip exactly. The list
// query uses a fuzzy (regexp) match on the ip, so entries whose address only
// happens to match the pattern (e.g. 10.22.42.26 for 10.22.42.2) must be
// filtered out, otherwise the ssh session would be redirected to another
// host's address.
func pickHostByIp(hosts []jsonutils.JSONObject, ip string) *compute_api.HostDetails {
	for i := range hosts {
		hostDetails := compute_api.HostDetails{}
		if err := hosts[i].Unmarshal(&hostDetails); err != nil {
			continue
		}
		if hostHasIp(&hostDetails, ip) {
			return &hostDetails
		}
	}
	return nil
}

// orderServerCandidatesByIp puts servers carrying the ip exactly before the
// ones that merely match the fuzzy list filter.
func orderServerCandidatesByIp(servers []jsonutils.JSONObject, ip string) []compute_api.ServerDetails {
	candidates := make([]compute_api.ServerDetails, 0, len(servers))
	exact := make([]compute_api.ServerDetails, 0, len(servers))
	for i := range servers {
		serverDetails := compute_api.ServerDetails{}
		if err := servers[i].Unmarshal(&serverDetails); err != nil {
			continue
		}
		candidates = append(candidates, serverDetails)
		if serverHasIp(&serverDetails, ip) {
			exact = append(exact, serverDetails)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return candidates
}

// ResolveSSHIPPortByIp validates that the ip belongs to a host or server
// accessible by the user and fills the connection info accordingly, so the
// ssh console can not be used to dial arbitrary internal addresses.
func ResolveSSHIPPortByIp(ctx context.Context, s *mcclient.ClientSession, ip string, port int, conn *SSshConnectionInfo) error {
	if port <= 0 {
		port = 22
	}
	// try hosts first: the ip must be one of the host's addresses (access
	// ip, ipmi ip, public ip or a nic ip) of a host visible to the user
	hostQuery := jsonutils.NewDict()
	hostQuery.Set("any_ip", jsonutils.NewStringArray([]string{ip}))
	hostRes, err := compute.Hosts.List(s, hostQuery)
	if err == nil && hostRes.Total > 0 {
		// the any_ip filter is a regexp match, so only accept a host that
		// really carries the requested ip, not one whose address merely
		// matches the pattern
		if hostDetails := pickHostByIp(hostRes.Data, ip); hostDetails != nil {
			conn.IP = hostDetails.AccessIp
			if len(conn.IP) == 0 {
				conn.IP = ip
			}
			conn.Port = port
			conn.HostDetails = hostDetails
			return nil
		}
	}
	// then servers: only servers visible to the user are listed
	serverQuery := jsonutils.NewDict()
	serverQuery.Set("ip_addrs", jsonutils.NewStringArray([]string{ip}))
	serverRes, err := compute.Servers.List(s, serverQuery)
	if err != nil {
		return errors.Wrap(err, "Servers.List")
	}
	if serverRes.Total == 0 {
		return errors.Wrapf(httperrors.ErrNotFound, "no server or host found with ip %s", ip)
	}
	// the ip_addrs filter is a regexp match too, so try the servers carrying
	// the exact ip first; resolveServerIPPortById also validates the ip
	// against the server's network interfaces
	var lastErr error
	for _, serverDetails := range orderServerCandidatesByIp(serverRes.Data, ip) {
		resolvedIp, resolvedPort, guestDetails, err := resolveServerIPPortById(ctx, s, serverDetails.Id, ip, port)
		if err != nil {
			lastErr = err
			continue
		}
		conn.IP = resolvedIp
		conn.Port = resolvedPort
		conn.GuestDetails = guestDetails
		return nil
	}
	if lastErr != nil {
		return errors.Wrap(lastErr, "resolve server ssh info")
	}
	return errors.Wrapf(httperrors.ErrNotFound, "no server or host found with ip %s", ip)
}

func ResolveHostSSHIPPortById(ctx context.Context, s *mcclient.ClientSession, id string, ip string, port int) (string, int, *compute_api.HostDetails, error) {
	if port <= 0 {
		port = 22
	}

	hostDetails, err := FetchHostInfo(ctx, s, id)
	if err != nil {
		return "", 0, nil, errors.Wrap(err, "fetchServerInfo")
	}
	return hostDetails.AccessIp, port, hostDetails, nil
}

type sForwardInfo struct {
	ProxyAddr string `json:"proxy_addr"`
	ProxyPort int    `json:"proxy_port"`
}

func acquireForward(ctx context.Context, session *mcclient.ClientSession, srvid string, ip string, proto string, port int) (string, int, error) {
	lockman.LockRawObject(ctx, "server", srvid)
	defer lockman.ReleaseRawObject(ctx, "server", srvid)

	addr, nport, err := listForward(session, srvid, ip, proto, port)
	if err == nil {
		return addr, nport, nil
	}
	if errors.Cause(err) == httperrors.ErrNotFound {
		return openForward(session, srvid, ip, proto, port)
	} else {
		return "", 0, errors.Wrap(err, "listForward")
	}
}

func listForward(session *mcclient.ClientSession, srvid string, ip string, proto string, port int) (string, int, error) {
	opt := &options.ServerListForwardOptions{
		ServerIdOptions: options.ServerIdOptions{
			ID: srvid,
		},
		Proto: &proto,
		Port:  &port,
		Addr:  &ip,
	}

	params, err := opt.Params()
	if err != nil {
		return "", 0, errors.Wrap(err, "get list forward params")
	}
	jsonItem, err := modules.Servers.PerformAction(session, opt.ID, "list-forward", params)
	if err != nil {
		return "", 0, errors.Wrap(err, "list-forward")
	}

	if jsonItem.Contains("forwards") {
		infoList := make([]sForwardInfo, 0)
		err = jsonItem.Unmarshal(&infoList, "forwards")
		if err != nil {
			return "", 0, errors.Wrap(err, "Unmarshal forwards")
		}
		if len(infoList) > 0 {
			return infoList[0].ProxyAddr, infoList[0].ProxyPort, nil
		}
	}

	return "", 0, errors.Wrap(httperrors.ErrNotFound, "no forwards")
}

func openForward(session *mcclient.ClientSession, srvid string, ip string, proto string, port int) (string, int, error) {
	opt := &options.ServerOpenForwardOptions{
		ServerIdOptions: options.ServerIdOptions{
			ID: srvid,
		},
		Proto: proto,
		Port:  port,
		Addr:  ip,
	}

	params, err := opt.Params()
	if err != nil {
		return "", 0, errors.Wrap(err, "get open forward params")
	}

	jsonItem, err := modules.Servers.PerformAction(session, opt.ID, "open-forward", params)
	if err != nil {
		return "", 0, errors.Wrap(err, "open-forward")
	}

	info := sForwardInfo{}
	err = jsonItem.Unmarshal(&info)
	if err != nil {
		return "", 0, errors.Wrap(err, "Unmarshal")
	}

	return info.ProxyAddr, info.ProxyPort, nil
}
