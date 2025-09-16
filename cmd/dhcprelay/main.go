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

package main

import (
	"fmt"
	"os"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostdhcp"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

type Options struct {
	Help bool `help:"Show help"`

	Interface string `help:"Listening interface, e.g. eth0"`

	Ip    string `help:"Listening interface IP, e.g. 192.168.22.2"`
	Port  int    `help:"listening port" default:"67"`
	Relay string `help:"Relay server address, e.g. 192.168.22.23"`

	Ip6    string `help:"Listening interface IP, e.g. 2001:db8::1"`
	Port6  int    `help:"listening port" default:"547"`
	Relay6 string `help:"Relay server address, e.g. 2001:db8::23"`
}

func relayMain() error {
	parse, err := structarg.NewArgumentParser(&Options{},
		"dhcprelay",
		"An independent dhcp relay.",
		`See "dhcprelay --help" for help.`)

	if err != nil {
		return err
	}

	err = parse.ParseArgs(os.Args[1:], false)
	if err != nil {
		return err
	}

	options := parse.Options().(*Options)

	if options.Help {
		fmt.Print(parse.HelpString())
		return errors.Error("Need help!")
	}

	if len(options.Interface) == 0 {
		return errors.Error("Missing interface")
	}
	if len(options.Ip) == 0 && len(options.Ip6) == 0 {
		return errors.Error("Missing interface IP or IP6")
	}

	if len(options.Ip) > 0 {
		if len(options.Relay) == 0 {
			return errors.Error("Missing DHCP relay server or relay server6")
		}

		relayConfig := &hostdhcp.SDHCPRelayUpstream{}
		relayConfig.IP = options.Relay
		relayConfig.Port = 67
		srv, err := hostdhcp.NewGuestDHCPServer(options.Interface, options.Port, relayConfig)
		if err != nil {
			return errors.Wrap(err, "NewGuestDHCPServer")
		}

		srv.Start(false)

		srv.RelaySetup(options.Ip)
	}

	if len(options.Ip6) > 0 {
		if len(options.Relay6) == 0 {
			return errors.Error("Missing DHCP relay server or relay server6")
		}

		relayConfig6 := &hostdhcp.SDHCPRelayUpstream{}
		relayConfig6.IP = options.Relay6
		relayConfig6.Port = options.Port6
		srv6, err := hostdhcp.NewGuestDHCP6Server(options.Interface, options.Port6, relayConfig6)
		if err != nil {
			return errors.Wrap(err, "NewGuestDHCP6Server")
		}

		srv6.Start(false)

		srv6.RelaySetup(options.Ip6)
	}

	for {
		time.Sleep(time.Hour)
	}

	// return nil
}

func main() {
	defer atexit.Handle()

	err := relayMain()

	if err != nil {
		fmt.Fprintf(os.Stdout, "Error: %s", err)
		os.Exit(-1)
	}
}
