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

package utils

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/apis/compute"
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

func HaproxySendProxy(s string) (r string, err error) {
	switch s {
	case compute.LB_SENDPROXY_OFF, "":
	case compute.LB_SENDPROXY_V1:
		r = "send-proxy"
	case compute.LB_SENDPROXY_V2:
		r = "send-proxy-v2"
	case compute.LB_SENDPROXY_V2_SSL:
		r = "send-proxy-v2-ssl"
	case compute.LB_SENDPROXY_V2_SSL_CN:
		r = "send-proxy-v2-ssl-cn"
	default:
		err = fmt.Errorf("unknown SendProxy: %s", s)
	}
	return
}
