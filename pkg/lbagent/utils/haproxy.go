package utils

import (
	"fmt"
	"strings"
)

const HaproxyCfgExt = "cfg"

func HaproxyBalanceAlgorithm(scheduler string) (balance string, err error) {
	switch scheduler {
	case "rr", "wrr":
		balance = "roundrobin"
	case "wlc":
		balance = "leastconn"
	case "sch":
		balance = "source"
	case "tch":
		// NOTE haproxy supports only TCP type proxy
		balance = "source"
	default:
		err = fmt.Errorf("unknown scheduler type %q", scheduler)
	}
	return
}

type HaproxySslPolicyParams struct {
	SslMinVer string
	Ciphers   string
}

// TODO restrict ciphers as noted in https://help.aliyun.com/document_detail/90740.html
func HaproxySslPolicy(policy string) *HaproxySslPolicyParams {
	r := &HaproxySslPolicyParams{}
	switch policy {
	case "tls_cipher_policy_1_0":
		r.SslMinVer = "TLSv1.0"
	case "tls_cipher_policy_1_1":
		r.SslMinVer = "TLSv1.1"
	case "tls_cipher_policy_1_2":
		r.SslMinVer = "TLSv1.2"
	case "tls_cipher_policy_1_2_strict":
		r.SslMinVer = "TLSv1.2"
	default:
		return nil
	}
	return r
}

func HaproxyConfigHttpCheck(uri, domain string) string {
	if uri == "" {
		uri = "/"
	}
	s := fmt.Sprintf("option httpchk HEAD %s HTTP/1.0", uri)
	if domain != "" {
		s += `\r\nHost:\ ` + domain
	}
	return s
}

func HaproxyConfigHttpCheckExpect(s string) string {
	ss := []string{}
	for _, s := range strings.Split(s, ",") {
		s = s[len("http_"):]
		s = strings.Replace(s, "x", ".", -1)
		ss = append(ss, s)
	}
	s = strings.Join(ss, "|")
	s = fmt.Sprintf("http-check expect rstatus %s", s)
	return s
}
