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
