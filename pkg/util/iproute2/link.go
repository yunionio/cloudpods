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

package iproute2

import (
	"net"

	"github.com/vishvananda/netlink"

	"yunion.io/x/pkg/errors"
)

type Link struct {
	ifname string
	link   netlink.Link

	errs []error
}

func NewLink(ifname string) *Link {
	l := &Link{
		ifname: ifname,
	}
	{
		link, err := netlink.LinkByName(l.ifname)
		if err != nil {
			l.addErr(err, "LinkByName %s", ifname)
			return l
		}
		l.link = link
	}
	return l
}

func (l *Link) addErr(err error, fmtStr string, vals ...interface{}) {
	l.errs = append(l.errs, errors.Wrapf(err, fmtStr, vals...))
}

func (l *Link) addErr2(err error) {
	l.errs = append(l.errs, err)
}

func (l *Link) Err() error {
	err := errors.NewAggregate(l.errs)
	if err != nil {
		return errors.Wrapf(err, "Link %s", l.ifname)
	}
	return nil
}

func (l *Link) ResetErr() {
	l.errs = nil
}

func (l *Link) Up() *Link {
	if l.link != nil {
		if err := netlink.LinkSetUp(l.link); err != nil {
			l.addErr(err, "LinkSetUp")
		}
	}
	return l
}

func (l *Link) Down() *Link {
	if l.link != nil {
		if err := netlink.LinkSetDown(l.link); err != nil {
			l.addErr(err, "LinkSetDown")
		}
	}
	return l
}

func (l *Link) MTU(mtu int) *Link {
	if l.link != nil {
		if err := netlink.LinkSetMTU(l.link, mtu); err != nil {
			l.addErr(err, "LinkSetMTU")
		}
	}
	return l
}

func (l *Link) Address(address string) *Link {
	if l.link != nil {
		hwaddr, err := net.ParseMAC(address)
		if err != nil {
			l.addErr(err, "bad hwaddr: %s", address)
			return l
		}
		if err := netlink.LinkSetHardwareAddr(l.link, hwaddr); err != nil {
			l.addErr(err, "LinkSetHardwareAddr")
		}
	}
	return l
}
