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

package ansible

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	compute "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type AnsibleHostsOptions struct {
	List       bool   `help:"List all ansible inventory"`
	Host       string `help:"List of a host"`
	PrivateKey string `help:"path to private key to use for ansible"`
	Port       int    `help:"optional port, if port is not 22"`
	User       string `help:"username to try"`
	UserBecome string `help:"username to sudo"`
}

func serverGetNameIP(srv jsonutils.JSONObject) (string, string, error) {
	host, _ := srv.GetString("name")
	if len(host) == 0 {
		return "", "", fmt.Errorf("No name for server")
	}
	ips, _ := srv.GetString("ips")
	if len(ips) == 0 {
		return "", "", fmt.Errorf("no ips for server %s", host)
	}
	ipList := strings.Split(ips, ",")
	if len(ipList) == 0 {
		return "", "", fmt.Errorf("no valid ips for server %s", host)
	}
	return host, ipList[0], nil
}

func doList(s *mcclient.ClientSession, args *AnsibleHostsOptions) error {
	hostVars := jsonutils.NewDict()
	hosts := jsonutils.NewArray()

	limit := 2048
	total := 1
	for hosts.Size() < total {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(int64(limit)), "limit")
		params.Add(jsonutils.NewInt(int64(hosts.Size())), "offset")
		params.Add(jsonutils.JSONTrue, "details")
		retList, err := compute.Servers.List(s, params)
		if err != nil {
			return err
		}
		if len(retList.Data) > 0 {
			for i := 0; i < len(retList.Data); i += 1 {
				host, ip, err := serverGetNameIP(retList.Data[i])
				if err == nil {
					hosts.Add(jsonutils.NewString(host))
					hostVars.Add(jsonutils.NewString(ip), host, "ansible_host")
					if args.Port > 0 {
						hostVars.Add(jsonutils.NewInt(int64(args.Port)), host, "ansible_port")
					}
					if len(args.PrivateKey) > 0 {
						hostVars.Add(jsonutils.NewString(args.PrivateKey), host, "ansible_ssh_private_key_file")
					}
					if len(args.User) > 0 {
						hostVars.Add(jsonutils.NewString(args.User), host, "ansible_user")
					}
					if len(args.UserBecome) > 0 {
						hostVars.Add(jsonutils.NewString(args.UserBecome), host, "ansible_become_user")
					}
				}
			}
		}
		total = retList.Total
	}

	output := jsonutils.NewDict()
	output.Add(hostVars, "_meta", "hostvars")
	children := jsonutils.NewArray(jsonutils.NewString("ungrouped"))
	output.Add(children, "all", "children")
	output.Add(hosts, "ungrouped", "hosts")

	fmt.Printf("%s\n", output.PrettyString())

	return nil
}

func doHost(s *mcclient.ClientSession, host string, args *AnsibleHostsOptions) error {
	srv, err := compute.Servers.Get(s, host, nil)
	if err != nil {
		return err
	}
	_, ipstr, err := serverGetNameIP(srv)
	if err != nil {
		return err
	}
	hostVar := jsonutils.NewDict()
	hostVar.Add(jsonutils.NewString(ipstr), "ansible_host")
	if args.Port > 0 {
		hostVar.Add(jsonutils.NewInt(int64(args.Port)), "ansible_port")
	}
	if len(args.PrivateKey) > 0 {
		hostVar.Add(jsonutils.NewString(args.PrivateKey), "ansible_ssh_private_key_file")
	}
	if len(args.User) > 0 {
		hostVar.Add(jsonutils.NewString(args.User), "ansible_user")
	}
	if len(args.UserBecome) > 0 {
		hostVar.Add(jsonutils.NewString(args.UserBecome), "ansible_become_user")
	}

	fmt.Printf("%s\n", hostVar.PrettyString())

	return nil
}

func init() {
	R(&AnsibleHostsOptions{}, "ansible-hosts", "List ansible inventory", func(s *mcclient.ClientSession, args *AnsibleHostsOptions) error {
		if len(args.Host) > 0 {
			return doHost(s, args.Host, args)
		} else if args.List {
			return doList(s, args)
		} else {
			return fmt.Errorf("Invalid arguments, either --list or --host must be specified")
		}
	})
}
