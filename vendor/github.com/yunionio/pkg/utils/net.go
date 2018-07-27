package utils

import (
	"net"
	"strconv"
	"strings"
)

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return net.IP{}
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

func GetAddrPort(addrPort string) (string, int) {
	parts := strings.Split(addrPort, ":")
	port, _ := strconv.Atoi(parts[1])
	return parts[0], port
}
