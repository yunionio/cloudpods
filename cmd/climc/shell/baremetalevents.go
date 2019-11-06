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

package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type BaremetalEventListOptions struct {
		HostId       string `help:"filter by hostId"`
		HostName     string `help:"filter by hostname"`
		IpmiIp       string `help:"filter by ipmi_ip"`
		Type         string `help:"filter by type" choices:"system|manager"`
		PagingMarker string `help:"marker for pagination"`
		Limit        int    `help:"page limit, default 20" default:"20"`
		Severity     string `help:"filter by severity"`
		Since        string `help:"Show logs since specific date" metavar:"DATETIME"`
		Until        string `help:"Show logs until specific date" metavar:"DATETIME"`
	}
	R(&BaremetalEventListOptions{}, "baremetal-event-list", "List baremetal events", func(s *mcclient.ClientSession, args *BaremetalEventListOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.BaremetalEvents.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.BaremetalEvents.GetColumns(s))
		return nil
	})

	type BaremetalEventCreateOptions struct {
		HOST     string `json:"-" help:"Host Id"`
		Type     string `json:"type" help:"event type" choices:"system|manager"`
		EventId  string `json:"event_id" help:"original event Id"`
		MESSAGE  string `json:"message" help:"content of event"`
		Severity string `json:"severity" help:"Severity of event"`
		CREATED  string `json:"created" help:"when event was created"`
	}
	R(&BaremetalEventCreateOptions{}, "baremetal-event-create", "Create baremetal event", func(s *mcclient.ClientSession, args *BaremetalEventCreateOptions) error {
		host, err := modules.Hosts.Get(s, args.HOST, nil)
		if err != nil {
			return err
		}
		hostId, _ := host.Get("id")
		hostName, _ := host.Get("name")
		ipmiIp, _ := host.Get("ipmi_ip")
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		params.Add(hostId, "host_id")
		params.Add(hostName, "host_name")
		params.Add(ipmiIp, "ipmi_ip")
		result, err := modules.BaremetalEvents.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
