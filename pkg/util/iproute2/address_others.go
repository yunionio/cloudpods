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

//go:build !linux
// +build !linux

package iproute2

import (
	"github.com/vishvananda/netlink"
)

type Address struct {
	*Link

	addrBad bool
	addrs   []*netlink.Addr

	testcb func()
}

func nop() {}

func NewAddress(ifname string, addresses ...string) *Address {
	l := NewLink(ifname)
	r := &Address{
		Link:   l,
		testcb: nop,
	}

	addrs := make([]*netlink.Addr, len(addresses))
	for i, address := range addresses {
		addr, err := netlink.ParseAddr(address)
		if err != nil {
			r.addrBad = true
			r.addErr(err, "parse addr")
			continue
		}
		addrs[i] = addr
	}
	if !r.addrBad {
		r.addrs = addrs
	}
	return r
}

func (address *Address) Exact() *Address {
	return nil
}

func (address *Address) link() (link netlink.Link, ok bool) {
	if address.addrBad {
		return
	}
	link = address.Link.link
	if link != nil {
		ok = true
	}
	return
}

func (address *Address) Add() *Address {
	link, ok := address.link()
	if !ok {
		return address
	}
	for _, addr := range address.addrs {
		err := netlink.AddrReplace(link, addr)
		if err != nil {
			address.addErr(err, "Add: AddrReplace %s ", addr)
		}
	}
	return address
}
