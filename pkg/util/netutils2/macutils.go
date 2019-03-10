package netutils2

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
)

type SMacAddr [6]byte

func ErrMacFormat(macStr string) error {
	return errors.New(fmt.Sprintf("invalid mac format: %s", macStr))
}

func ParseMac(macStr string) (SMacAddr, error) {
	mac := SMacAddr{}
	macStr = netutils.FormatMacAddr(macStr)
	parts := strings.Split(macStr, ":")
	if len(parts) != 6 {
		return mac, ErrMacFormat(macStr)
	}
	for i := 0; i < 6; i += 1 {
		bt, err := strconv.ParseInt(parts[i], 16, 64)
		if err != nil {
			return mac, ErrMacFormat(macStr)
		}
		mac[i] = byte(bt)
	}
	return mac, nil
}

func (mac SMacAddr) Add(step int) SMacAddr {
	mac2 := SMacAddr{}
	leftOver := step
	for i := 5; i >= 2; i -= 1 { // skip first 2 vendor bytes
		newByte := int(mac[i]) + leftOver
		res := 0
		if newByte < 0 {
			log.Debugf("%d %d", mac[i], newByte)
			res = ((-newByte) / 0x100) + 1
			newByte = newByte + res*0x100
			log.Debugf("%d %d", newByte, res)
		}
		mac2[i] = byte(newByte % 0x100)
		leftOver = newByte/0x100 - res
	}
	return mac2
}

func (mac SMacAddr) String() string {
	var parts [6]string
	for i := 0; i < len(parts); i += 1 {
		parts[i] = fmt.Sprintf("%02x", mac[i])
	}
	return strings.Join(parts[:], ":")
}
