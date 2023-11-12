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

package models

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	compute_models "yunion.io/x/onecloud/pkg/compute/models"
	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
)

var haproxyConfigErrNop = errors.Error("nop haproxy config snippet")

type GenHaproxyConfigsResult struct {
	LoadbalancersEnabled []*Loadbalancer
}

func (b *LoadbalancerCorpus) GenHaproxyToplevelConfig(dir string, opts *AgentParams) error {
	buf := bytes.NewBufferString("# yunion lb auto-generated 00-haproxy.cfg\n")
	haproxyConfigTmpl := opts.HaproxyConfigTmpl
	err := haproxyConfigTmpl.Execute(buf, opts.Data)
	if err != nil {
		return err
	}
	data := buf.Bytes()
	p := filepath.Join(dir, "00-haproxy.cfg")
	err = ioutil.WriteFile(p, data, agentutils.FileModeFile)
	if err != nil {
		return err
	}
	return nil
}

func (b *LoadbalancerCorpus) GenHaproxyConfigs(dir string, opts *AgentParams) (*GenHaproxyConfigsResult, error) {
	if len(b.LoadbalancerCertificates) > 0 {
		certsBase := filepath.Join(dir, "certs")
		certsBaseFinal := filepath.Join(agentutils.DirStagingToFinal(dir), "certs")
		err := os.MkdirAll(certsBase, agentutils.FileModeDirSensitive)
		if err != nil {
			return nil, fmt.Errorf("mkdir %s: %s", certsBase, err)
		}
		{
			p := filepath.Join(dir, "01-haproxy.cfg")
			lines := []string{
				"global",
				fmt.Sprintf("	crt-base %s", certsBaseFinal),
				"",
			}
			s := strings.Join(lines, "\n")
			err := ioutil.WriteFile(p, []byte(s), agentutils.FileModeFile)
			if err != nil {
				return nil, fmt.Errorf("write 01-haproxy.cfg: %s", err)
			}
		}
		for _, lbcert := range b.LoadbalancerCertificates {
			d := []byte(lbcert.Certificate)
			if len(d) > 0 && d[len(d)-1] != '\n' {
				d = append(d, '\n')
			}
			d = append(d, []byte(lbcert.PrivateKey)...)
			fn := fmt.Sprintf("%s.pem", lbcert.Id)
			p := filepath.Join(certsBase, fn)
			err := ioutil.WriteFile(p, d, agentutils.FileModeFileSensitive)
			if err != nil {
				return nil, fmt.Errorf("write cert %s: %s", lbcert.Id, err)
			}
		}
	}
	for _, lbacl := range b.LoadbalancerAcls {
		cidrs := []string{}
		if lbacl.AclEntries != nil {
			for _, aclEntry := range *lbacl.AclEntries {
				cidrs = append(cidrs, aclEntry.Cidr)
			}
		}
		if len(cidrs) > 0 {
			s := fmt.Sprintf("## loadbalancer acl %s(%s)\n", lbacl.Name, lbacl.Id)
			s += strings.Join(cidrs, "\n")
			s += "\n"
			p := filepath.Join(dir, "acl-"+lbacl.Id)
			err := ioutil.WriteFile(p, []byte(s), agentutils.FileModeFile)
			if err != nil {
				return nil, err
			}
		}
	}

	r := &GenHaproxyConfigsResult{
		LoadbalancersEnabled: []*Loadbalancer{},
	}
	for _, lb := range b.Loadbalancers {
		if lb.ClusterId != opts.AgentModel.ClusterId {
			continue
		}
		if lb.Status != "enabled" {
			continue
		}
		if lb.Address == "" {
			continue
		}
		if len(lb.Listeners) == 0 {
			continue
		}
		buf := bytes.NewBufferString(fmt.Sprintf("## loadbalancer %s(%s)\n\n", lb.Name, lb.Id))
		hasActiveListener := false
		for _, listener := range lb.Listeners {
			if listener.Status != "enabled" {
				continue
			}
			var err error
			switch listener.ListenerType {
			case "http", "https":
				err = b.genHaproxyConfigHttp(buf, listener, opts)
			case "tcp":
				err = b.genHaproxyConfigTcp(buf, listener, opts)
			case "udp":
				// we record it for use in keepalived, gobetween conf gen
				r.LoadbalancersEnabled = append(r.LoadbalancersEnabled, lb)
				continue
			default:
				log.Infof("haproxy: ignore listener type %s", listener.ListenerType)
				continue
			}
			if err == haproxyConfigErrNop {
				continue
			}
			if err != nil {
				return nil, err
			}
			hasActiveListener = true
			buf.WriteString("\n\n") // listeners sep lines
		}
		if hasActiveListener {
			r.LoadbalancersEnabled = append(r.LoadbalancersEnabled, lb)

			d := buf.Bytes()
			if len(d) > 1 { // this is for sure because of "\n\n"
				d = d[:len(d)-2]
			}
			fn := fmt.Sprintf("%s.%s", lb.Id, agentutils.HaproxyCfgExt)
			p := filepath.Join(dir, fn)
			err := ioutil.WriteFile(p, d, agentutils.FileModeFile)
			if err != nil {
				return nil, err
			}
		}
	}
	return r, nil
}

func (b *LoadbalancerCorpus) genHaproxyConfigCommon(lb *Loadbalancer, listener *LoadbalancerListener, opts *AgentParams) (map[string]interface{}, error) {
	data := map[string]interface{}{
		"comment":       fmt.Sprintf("%s(%s)", listener.Name, listener.Id),
		"id":            listener.Id,
		"listener_type": listener.ListenerType,
	}
	{
		address := lb.GetAddress()
		bind := fmt.Sprintf("%s:%d", address, listener.ListenerPort)
		if listener.ListenerType == "https" && listener.certificate != nil {
			bind += fmt.Sprintf(" ssl crt %s.pem", listener.certificate.Id)
			if listener.TLSCipherPolicy != "" {
				policy := agentutils.HaproxySslPolicy(listener.TLSCipherPolicy)
				if policy != nil {
					bind += fmt.Sprintf(" ssl-min-ver %s", policy.SslMinVer)
				}
			}
			if listener.EnableHttp2 {
				bind += " alpn h2,http/1.1"
			}
		}
		data["bind"] = bind
	}
	{
		agentHaproxyParams := opts.AgentModel.Params.Haproxy
		if agentHaproxyParams.GlobalLog != "" {
			switch listener.ListenerType {
			case "http", "https":
				if agentHaproxyParams.LogHttp {
					data["log"] = true
				}
			case "tcp":
				if agentHaproxyParams.LogTcp {
					data["log"] = true
				}
			}
		}
	}
	if listener.AclStatus == "on" {
		lbacl, ok := b.LoadbalancerAcls[listener.AclId]
		if ok && lbacl.AclEntries != nil && len(*lbacl.AclEntries) > 0 {
			var action, cond string
			switch listener.ListenerType {
			case "tcp":
				action = "tcp-request connection reject"
			case "http", "https":
				action = "http-request deny"
			}
			switch listener.AclType {
			case "black":
				cond = "if"
			case "white":
				cond = "unless"
			}
			if action != "" && cond != "" {
				acl := fmt.Sprintf("%s %s { src -f acl-%s }", action, cond, listener.AclId)
				data["acl"] = acl
			}
		}
	}
	{
		// NOTE timeout tunnel is not set.  We may need to prepare a
		// default section for each frontend/listener
		timeoutsMap := map[string]int{
			"client_request_timeout": listener.ClientRequestTimeout,
			"client_idle_timeout":    listener.ClientIdleTimeout,
		}
		for k, v := range timeoutsMap {
			if v > 0 {
				data[k] = fmt.Sprintf("%ds", v)
			}
		}
	}
	return data, nil
}

func (b *LoadbalancerCorpus) genHaproxyConfigBackend(data map[string]interface{}, lb *Loadbalancer, listener *LoadbalancerListener, backendGroup *LoadbalancerBackendGroup) error {
	var mode string
	var balanceAlgorithm string
	var httpCheck, httpCheckExpect string
	var checkEnable, httpCheckEnable bool
	var err error
	{ // mode
		switch listener.ListenerType {
		case "tcp":
			mode = "tcp"
		case "http", "https":
			mode = "http"
		default:
			return fmt.Errorf("haproxy: unsupported listener type %s", listener.ListenerType)
		}
	}
	{ // balance algorithm
		balanceAlgorithm, err = agentutils.HaproxyBalanceAlgorithm(listener.Scheduler)
		if err != nil {
			return err
		}
	}
	{ // (http) check enabled?
		if listener.HealthCheck == "on" {
			checkEnable = true
			if listener.HealthCheckType == "http" {
				httpCheckEnable = true
				httpCheck = agentutils.HaproxyConfigHttpCheck(
					listener.HealthCheckURI, listener.HealthCheckDomain)
				httpCheckExpect = agentutils.HaproxyConfigHttpCheckExpect(
					listener.HealthCheckHttpCode)
			}
		}
	}
	var stickySessionEnable bool
	if mode == "http" && listener.StickySession == "on" {
		// sticky session
		stickyCookie := ""
		switch listener.StickySessionType {
		case "insert":
			cookie := listener.StickySessionCookie
			if cookie == "" {
				cookie = "SERVERID"
			}
			stickyCookie = fmt.Sprintf("cookie %s insert indirect nocache", cookie)
			if maxIdle := listener.StickySessionCookieTimeout; maxIdle > 0 {
				stickyCookie += fmt.Sprintf(" maxidle %ds", maxIdle)
			}
		case "server":
			cookie := listener.StickySessionCookie
			if cookie != "" {
				stickyCookie = fmt.Sprintf("cookie %q rewrite", cookie)
			}
		}
		if stickyCookie != "" {
			data["stickyCookie"] = stickyCookie
			stickySessionEnable = true
		}
	}
	{
		listenerSendProxy, err := agentutils.HaproxySendProxy(listener.SendProxy)
		if err != nil {
			return fmt.Errorf("listener %s(%s): %v", listener.Name, listener.Id, err)
		}
		serverLines := []string{}
		for _, backend := range backendGroup.Backends {

			address, port := backend.GetAddressPort()
			serverLine := fmt.Sprintf("server %s %s:%d", backend.Id, address, port)
			if listener.Scheduler == "rr" {
				serverLine += " weight 1"
			} else {
				serverLine += fmt.Sprintf(" weight %d", backend.Weight)
			}
			if checkEnable {
				serverLine += fmt.Sprintf(" check rise %d fall %d inter %ds",
					listener.HealthCheckRise, listener.HealthCheckFall, listener.HealthCheckInterval)
			}
			if stickySessionEnable {
				serverLine += fmt.Sprintf(" cookie %q", backend.Id)
			}
			if listenerSendProxy != "" {
				serverLine += " " + listenerSendProxy
			} else if sendProxy, err := agentutils.HaproxySendProxy(backend.SendProxy); err != nil {
				return fmt.Errorf("backend %s(%s): %v", backend.Name, backend.Id, err)
			} else if sendProxy != "" {
				serverLine += " " + sendProxy
			} else {
				// nothing to do
			}
			if backend.Ssl == "on" {
				serverLine += " ssl"
				serverLine += " verify none"
				serverLine += " check-ssl"
			}
			serverLines = append(serverLines, serverLine)
		}
		data["servers"] = serverLines
	}
	if listener.HealthCheckTimeout > 0 {
		data["timeout_check"] = fmt.Sprintf("timeout check %ds", listener.HealthCheckTimeout)
	}
	{
		timeoutsMap := map[string]int{
			"backend_connect_timeout": listener.BackendConnectTimeout,
			"backend_idle_timeout":    listener.BackendIdleTimeout,
			"backend_tunnel_timeout":  listener.BackendIdleTimeout,
		}
		for k, v := range timeoutsMap {
			if v > 0 {
				data[k] = fmt.Sprintf("%ds", v)
			}
		}
	}
	data["mode"] = mode
	data["balanceAlgorithm"] = balanceAlgorithm
	if httpCheckEnable {
		data["httpCheck"] = httpCheck
		data["httpCheckExpect"] = httpCheckExpect
	}
	return nil
}

func (b *LoadbalancerCorpus) genHaproxyConfigHttpRate(data map[string]interface{}, requestRate, requestRatePerSrc int) error {
	periodSecond := 10
	id := data["id"].(string)
	dummyBackends := []map[string]string{}
	rateRules := []string{}

	// order matters here: every src uses up his own quota before touching
	// the shared one
	if requestRatePerSrc > 0 {
		idPerSrc := id + "_persrc"
		dummyBackends = append(dummyBackends, map[string]string{
			"id":          idPerSrc,
			"stick_table": fmt.Sprintf("stick-table type ip size 1m expire 1m store http_req_rate(%ds)", periodSecond),
		})
		rateRules = append(rateRules,
			fmt.Sprintf("http-request deny deny_status 429 if { src_http_req_rate(%s) gt %d }",
				idPerSrc, requestRatePerSrc*periodSecond),
			fmt.Sprintf("http-request track-sc0 src table %s",
				idPerSrc))
	}
	if requestRate > 0 {
		idTotal := id + "_total"
		dummyBackends = append(dummyBackends, map[string]string{
			"id":          idTotal,
			"stick_table": fmt.Sprintf("stick-table type integer size 1 expire 1m store http_req_rate(%ds)", periodSecond),
		})
		rateRules = append(rateRules,
			fmt.Sprintf("http-request deny deny_status 429 if { int(1),table_http_req_rate(%s) gt %d }",
				idTotal, requestRate*periodSecond),
			fmt.Sprintf("http-request track-sc1 int(1) table %s",
				idTotal))
	}
	data["rate_rules"] = rateRules
	data["dummy_backends"] = dummyBackends
	return nil
}

func (b *LoadbalancerCorpus) haproxyRedirectLine(r *compute_models.SLoadbalancerHTTPRedirect, listenerType string) string {
	var (
		code   = r.RedirectCode
		scheme = r.RedirectScheme
		host   = r.RedirectHost
		path   = r.RedirectPath
	)
	if scheme == "" {
		scheme = strings.ToLower(listenerType)
	}
	if host == "" {
		host = "%[req.hdr(host)]"
	}
	if path == "" {
		path = "%[capture.req.uri]"
	}
	line := fmt.Sprintf("http-request redirect code %d location %s://%s%s", code, scheme, host, path)
	return line
}

func (b *LoadbalancerCorpus) genHaproxyConfigHttp(buf *bytes.Buffer, listener *LoadbalancerListener, opts *AgentParams) error {
	var (
		lb = listener.loadbalancer
	)

	data, err := b.genHaproxyConfigCommon(lb, listener, opts)
	if err != nil {
		return errors.Wrap(err, "genHaproxyConfigCommon for http listener")
	}
	{
		// NOTE add X-Real-IP if needed
		//
		//	http-request set-header X-Client-IP %[src]
		//
		data["xforwardedfor"] = listener.XForwardedFor
		data["gzip"] = listener.Gzip
	}

	var (
		rules     = listener.rules.OrderedEnabledList()
		ruleLines = []string{}
		backends  = []interface{}{}

		ruleBackendIdGen = func(id string) string {
			return fmt.Sprintf("backends_rule-%s", id)
		}
	)
	{ // dispatch
		for _, rule := range rules {
			sufCond := ""
			if rule.Domain != "" || rule.Path != "" {
				sufCond += " if"
				if rule.Domain != "" {
					sufCond += fmt.Sprintf(" { hdr_dom(host) %q }", rule.Domain)
				}
				if rule.Path != "" {
					sufCond += fmt.Sprintf(" { path_beg %q }", rule.Path)
				}
			}
			if rule.Redirect == computeapi.LB_REDIRECT_OFF {
				// use_backend rule.Id if xx
				ruleLine := fmt.Sprintf("use_backend %s", ruleBackendIdGen(rule.Id))
				ruleLines = append(ruleLines, ruleLine+sufCond)
				continue
			} else if rule.Redirect == computeapi.LB_REDIRECT_RAW {
				// http-request redirect ... if xx
				ruleLine := b.haproxyRedirectLine(&rule.SLoadbalancerHTTPRedirect, listener.ListenerType)
				ruleLines = append(ruleLines, ruleLine+sufCond)
			} else {
				return haproxyConfigErrNop
			}
		}
		// default is a raw redirect
		if listener.Redirect == computeapi.LB_REDIRECT_RAW {
			ruleLines = append(ruleLines,
				b.haproxyRedirectLine(&listener.SLoadbalancerHTTPRedirect, listener.ListenerType),
			)
		}
		data["rules"] = ruleLines
	}
	{ // those with backend group
		// rules backend group
		for _, rule := range rules {
			// NOTE dup is ok
			if rule.BackendGroupId == "" {
				// just in case
				continue
			}
			if rule.Redirect != computeapi.LB_REDIRECT_OFF {
				continue
			}
			backendGroup := lb.BackendGroups[rule.BackendGroupId]
			backendData := map[string]interface{}{
				"comment": fmt.Sprintf("rule %s(%s) backendGroup %s(%s)",
					rule.Name, rule.Id,
					backendGroup.Name, backendGroup.Id),
				"id": ruleBackendIdGen(rule.Id),
			}
			if err := b.genHaproxyConfigBackend(backendData, lb, listener, backendGroup); err != nil {
				return err
			}
			if err := b.genHaproxyConfigHttpRate(backendData, rule.HTTPRequestRate, rule.HTTPRequestRatePerSrc); err != nil {
				return err
			}
			backends = append(backends, backendData)
		}
		// default backend group
		if listener.Redirect == computeapi.LB_REDIRECT_OFF && listener.BackendGroupId != "" {
			backendGroup := lb.BackendGroups[listener.BackendGroupId]
			backendData := map[string]interface{}{
				"comment": fmt.Sprintf("listener %s(%s) default backendGroup %s(%s)",
					listener.Name, listener.Id,
					backendGroup.Name, backendGroup.Id),
				"id": fmt.Sprintf("backends_listener_default-%s", listener.Id),
			}
			if err := b.genHaproxyConfigBackend(backendData, lb, listener, backendGroup); err != nil {
				return err
			}
			if err := b.genHaproxyConfigHttpRate(backendData, listener.HTTPRequestRate, listener.HTTPRequestRatePerSrc); err != nil {
				return err
			}
			backends = append(backends, backendData)
			data["default_backend"] = backendData
		}
		data["backends"] = backends
	}
	if len(ruleLines) == 0 && len(backends) == 0 {
		// nothing to serve
		return haproxyConfigErrNop
	}
	err = haproxyConfigTmpl.ExecuteTemplate(buf, "httpListen", data)
	return err
}

func (b *LoadbalancerCorpus) genHaproxyConfigTcp(buf *bytes.Buffer, listener *LoadbalancerListener, opts *AgentParams) error {
	lb := listener.loadbalancer
	data, err := b.genHaproxyConfigCommon(lb, listener, opts)
	if err != nil {
		return errors.Wrap(err, "genHaproxyConfigCommon for tcp listener")
	}
	if listener.BackendGroupId != "" {
		backendGroup := lb.BackendGroups[listener.BackendGroupId]
		backendData := map[string]interface{}{
			"comment": fmt.Sprintf("listener %s(%s) backendGroup %s(%s)",
				listener.Name, listener.Id,
				backendGroup.Name, backendGroup.Id),
			"id": fmt.Sprintf("backends_listener-%s", listener.Id),
		}
		err := b.genHaproxyConfigBackend(backendData, lb, listener, backendGroup)
		if err != nil {
			return err
		}
		data["backend"] = backendData
		err = haproxyConfigTmpl.ExecuteTemplate(buf, "tcpListen", data)
		return err
	}
	return haproxyConfigErrNop
}

var haproxyConfigTmpl = template.Must(template.New("").Parse(`
{{ define "tcpListen" -}}
# {{ .listener_type }} listener: {{ .comment }}
listen {{ .id }}
	bind {{ .bind }}
	mode tcp
	{{- println }}
	{{- if .log }}	{{ println "option tcplog" }} {{- end }}
	{{- if .acl }}	{{ println .acl }} {{- end}}
	{{- if .client_idle_timeout }}	timeout client {{ println .client_idle_timeout }} {{- end}}
	default_backend {{ .backend.id }}
{{ template "backend" .backend }}
{{- end }}

{{ define "httpListen" -}}
# {{ .listener_type }} listener: {{ .comment }}
{{- range .dummy_backends }}
backend {{ .id }}
	{{ println .stick_table }}
{{- end }}
frontend {{ .id }}
	bind {{ .bind }}
	mode http
	{{- println }}
	{{- if .log }}	{{ println "option httplog clf" }} {{- end }}
	{{- if .acl }}	{{ println .acl }} {{- end}}
	{{- if .client_request_timeout }}	timeout http-request {{ println .client_request_timeout }} {{- end}}
	{{- if .client_idle_timeout }}	timeout http-keep-alive {{ println .client_idle_timeout }} {{- end}}
	{{- if .xforwardedfor }}	{{ println "option forwardfor" }} {{- end}}
	{{- if .gzip }}	{{ println "compression algo gzip" }} {{- end}}
	{{- range .rules }}	{{ println . }} {{- end }}
	{{- if .default_backend.id }}	default_backend {{ println .default_backend.id }} {{- end }}
{{- range .backends }}
{{- template "backend" . }}
{{- end }}
{{- end }}

{{ define "backend" -}}
# {{ .comment }}
{{- range .dummy_backends }}
backend {{ .id }}
	{{ println .stick_table }}
{{- end }}
backend {{ .id }}
	mode {{ .mode }}
	balance {{ .balanceAlgorithm }}
	{{- println }}
	{{- range .rate_rules }}	{{ println . }} {{- end }}
	{{- if .backend_connect_timeout }}	timeout connect {{ println .backend_connect_timeout }} {{- end}}
	{{- if .backend_idle_timeout }}	timeout server {{ println .backend_idle_timeout }} {{- end}}
	{{- if .timeout_check }}	{{ println .timeout_check }} {{- end }}
	{{- if .stickyCookie }}	{{ println .stickyCookie }} {{- end }}
	{{- if .httpCheck }}	{{ println .httpCheck }} {{- end }}
	{{- if .httpCheckExpect }}	{{ println .httpCheckExpect }} {{- end }}
	{{- range .servers }}	{{ println . }} {{- end }}
{{- end }}
`))
