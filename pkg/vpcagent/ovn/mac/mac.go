package mac

import (
	"crypto/md5"
	"fmt"
)

func HashMac(in ...string) string {
	h := md5.New()
	for _, s := range in {
		h.Write([]byte(s))
	}
	sum := h.Sum(nil)
	b := sum[0]
	b &= 0xfe
	b |= 0x02
	mac := fmt.Sprintf("%02x", b)
	for _, b := range sum[1:6] {
		mac += fmt.Sprintf(":%02x", b)
	}
	return mac
}

func HashVpcHostDistgwMac(hostId string) string {
	return HashMac(hostId)
}
