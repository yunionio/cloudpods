package main

import (
	"net"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/dhcp"
)

type Handler struct {
}

func (h *Handler) ServeDHCP(pkt *dhcp.Packet, _ *net.Interface) (*dhcp.Packet, error) {
	log.Errorln(pkt.DebugString())
	return nil, nil
}

func main() {
	s := dhcp.NewDHCPServer("0.0.0.0", 67)

	err := s.ListenAndServe(&Handler{})
	if err != nil {
		log.Fatalf(err.Error())
	}
}
