package netutils2

import (
	"net/http"
	"strings"
)

func GetHttpRequestIp(r *http.Request) string {
	ipStr := r.Header.Get("X-Real-Ip")
	if len(ipStr) == 0 {
		ipStr = r.Header.Get("X-Forwarded-For")
		if len(ipStr) == 0 {
			ipStr = r.RemoteAddr
			colonPos := strings.Index(ipStr, ":")
			if colonPos > 0 {
				ipStr = ipStr[:colonPos]
			}
		}
	}
	return ipStr
}
