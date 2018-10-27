package models

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"yunion.io/x/log"

	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
)

var haproxyConfigErrNop = errors.New("nop haproxy config snippet")

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
			if d[len(d)-1] != '\n' {
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
		if lb.Status != "enabled" {
			continue
		}
		if lb.Address == "" {
			continue
		}
		if len(lb.listeners) == 0 {
			continue
		}
		buf := bytes.NewBufferString(fmt.Sprintf("## loadbalancer %s(%s)\n\n", lb.Name, lb.Id))
		hasActiveListener := false
		for _, listener := range lb.listeners {
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
			d = d[:len(d)-2]
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

func (b *LoadbalancerCorpus) genHaproxyConfigCommon(lb *Loadbalancer, listener *LoadbalancerListener, opts *AgentParams) map[string]interface{} {
	data := map[string]interface{}{
		"comment":       fmt.Sprintf("%s(%s)", listener.Name, listener.Id),
		"id":            listener.Id,
		"listener_type": listener.ListenerType,
	}
	{
		bind := fmt.Sprintf("%s:%d", lb.Address, listener.ListenerPort)
		if listener.ListenerType == "https" && listener.certificate != nil {
			bind += fmt.Sprintf(" ssl crt %s.pem", listener.certificate.Id)
			if listener.TLSCipherPolicy != "" {
				policy := agentutils.HaproxySslPolicy(listener.TLSCipherPolicy)
				if policy != nil {
					bind += fmt.Sprintf(" ssl-min-ver %s", policy.SslMinVer)
				}
			}
			if listener.EnableHttp2 {
				bind += fmt.Sprintf(" alpn h2,http/1.1")
			}
		}
		data["bind"] = bind
	}
	{
		agentHaproxyParams := opts.AgentModel.Params.Haproxy
		if agentHaproxyParams.GlobalLog != "" {
			if listener.ListenerType == "http" && agentHaproxyParams.LogHttp {
				data["log"] = true
			} else if listener.ListenerType == "tcp" && agentHaproxyParams.LogTcp {
				data["log"] = true
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
	return data
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
			stickyCookie = "cookie SERVERID insert indirect nocache"
			if maxIdle := listener.StickySessionCookieTimeout; maxIdle > 0 {
				stickyCookie += fmt.Sprintf(" maxidle %ds", maxIdle)
			}
		case "server":
			cookie := listener.StickySessionCookie
			if cookie != "" {
				stickyCookie = fmt.Sprintf("cookie %q rewrite nocache", cookie)
			}
		}
		if stickyCookie != "" {
			data["stickyCookie"] = stickyCookie
			stickySessionEnable = true
		}
	}
	{
		serverLines := []string{}
		for _, backend := range backendGroup.backends {
			serverLine := fmt.Sprintf("server %s %s:%d", backend.Id, backend.Address, backend.Port)
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

func (b *LoadbalancerCorpus) genHaproxyConfigHttp(buf *bytes.Buffer, listener *LoadbalancerListener, opts *AgentParams) error {
	lb := listener.loadbalancer
	rules := LoadbalancerListenerRules{}
	for id, rule := range listener.rules {
		if rule.Status != "enabled" {
			continue
		}
		rules[id] = rule
	}
	data := b.genHaproxyConfigCommon(lb, listener, opts)
	ruleBackendIdGen := func(id string) string {
		return fmt.Sprintf("backends_rule-%s", id)
	}
	{
		// NOTE add X-Real-IP if needed
		//
		//	http-request set-header X-Client-IP %[src]
		//
		data["xforwardedfor"] = listener.XForwardedFor
		data["gzip"] = listener.Gzip
	}
	{ // use_backend rule.Id if xx
		ruleLines := []string{}
		for _, rule := range rules {
			ruleLine := fmt.Sprintf("use_backend %s", ruleBackendIdGen(rule.Id))
			if rule.Domain != "" || rule.Path != "" {
				ruleLine += " if"
				if rule.Domain != "" {
					ruleLine += fmt.Sprintf(" { hdr_dom(host) %q }", rule.Domain)
				}
				if rule.Path != "" {
					ruleLine += fmt.Sprintf(" { path_beg %q }", rule.Path)
				}
			}
			ruleLines = append(ruleLines, ruleLine)
		}
		data["rules"] = ruleLines
	}
	{
		backends := []interface{}{}
		// rules backend group
		for _, rule := range rules {
			// NOTE dup is ok
			if rule.BackendGroupId == "" {
				// just in case
				continue
			}
			backendGroup := lb.backendGroups[rule.BackendGroupId]
			backendData := map[string]interface{}{
				"comment": fmt.Sprintf("rule %s(%s) backendGroup %s(%s)",
					rule.Name, rule.Id,
					backendGroup.Name, backendGroup.Id),
				"id": ruleBackendIdGen(rule.Id),
			}
			err := b.genHaproxyConfigBackend(backendData, lb, listener, backendGroup)
			if err != nil {
				return err
			}
			backends = append(backends, backendData)
		}
		// default backend group
		if listener.BackendGroupId != "" {
			backendGroup := lb.backendGroups[listener.BackendGroupId]
			backendData := map[string]interface{}{
				"comment": fmt.Sprintf("listener %s(%s) default backendGroup %s(%s)",
					listener.Name, listener.Id,
					backendGroup.Name, backendGroup.Id),
				"id": fmt.Sprintf("backends_listener_default-%s", listener.Id),
			}
			err := b.genHaproxyConfigBackend(backendData, lb, listener, backendGroup)
			if err != nil {
				return err
			}
			backends = append(backends, backendData)
			data["default_backend"] = backendData
		}
		if len(backends) == 0 {
			// no backendgroup specified, nothing to serve
			return haproxyConfigErrNop
		}
		data["backends"] = backends
	}
	err := haproxyConfigTmpl.ExecuteTemplate(buf, "httpListen", data)
	return err
}

func (b *LoadbalancerCorpus) genHaproxyConfigTcp(buf *bytes.Buffer, listener *LoadbalancerListener, opts *AgentParams) error {
	lb := listener.loadbalancer
	data := b.genHaproxyConfigCommon(lb, listener, opts)
	if listener.BackendGroupId != "" {
		backendGroup := lb.backendGroups[listener.BackendGroupId]
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
backend {{ .id }}
	mode {{ .mode }}
	balance {{ .balanceAlgorithm }}
	{{- println }}
	{{- if .backend_connect_timeout }}	timeout connect {{ println .backend_connect_timeout }} {{- end}}
	{{- if .backend_idle_timeout }}	timeout server {{ println .backend_idle_timeout }} {{- end}}
	{{- if .timeout_check }}	{{ println .timeout_check }} {{- end }}
	{{- if .stickyCookie }}	{{ println .stickyCookie }} {{- end }}
	{{- if .httpCheck }}	{{ println .httpCheck }} {{- end }}
	{{- if .httpCheckExpect }}	{{ println .httpCheckExpect }} {{- end }}
	{{- range .servers }}	{{ println . }} {{- end }}
{{- end }}
`))
