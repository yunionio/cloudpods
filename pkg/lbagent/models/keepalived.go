package models

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"text/template"

	"yunion.io/x/log"

	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
)

const keepalivedMiscCheckPath = "/opt/yunion/share/lbagent/healthcheck.sh"

type GenKeepalivedConfigOptions struct {
	LoadbalancersEnabled []*Loadbalancer
	AgentParams          *AgentParams
}

func (b *LoadbalancerCorpus) GenKeepalivedConfigs(dir string, opts *GenKeepalivedConfigOptions) error {
	agentParams := opts.AgentParams
	{
		addresses := []string{}
		for _, lb := range opts.LoadbalancersEnabled {
			if lb.Status != "enabled" {
				continue
			}
			if lb.Address == "" {
				continue
			}
			addresses = append(addresses, lb.Address)
		}
		agentParams.SetVrrpParams("addresses", addresses)
	}
	buf := bytes.NewBufferString("# yunion lb auto-generated keepalived.conf\n")
	{
		// write global_defs and vrrp_instance
		keepalivedConfigToplevelTmpl := agentParams.KeepalivedConfigTmpl
		err := keepalivedConfigToplevelTmpl.Execute(buf, agentParams.Data)
		if err != nil {
			return err
		}
	}
	if false {
		// write virtual_server
		for _, lb := range opts.LoadbalancersEnabled {
			for _, listener := range lb.listeners {
				if listener.Status != "enabled" {
					continue
				}
				if listener.BackendGroupId == "" {
					continue
				}
				if listener.ListenerType == "udp" {
					dataReals := []map[string]interface{}{}
					dataVirtual := map[string]interface{}{
						"virtual_ip":   lb.Address,
						"virtual_port": listener.ListenerPort,
					}
					{
						// scheduler
						switch listener.Scheduler {
						case "rr", "wrr", "lc", "wlc":
							dataVirtual["scheduler"] = listener.Scheduler
						case "sch", "tch":
							log.Warningf("scheduler %s converted to sh", listener.Scheduler)
							dataVirtual["scheduler"] = "sh"
						default:
							log.Warningf("scheduler %s converted to sh", listener.Scheduler)
							dataVirtual["scheduler"] = "sh"
						}
					}
					udpCheckEnabled := false
					{
						if listener.HealthCheck == "on" && listener.HealthCheckType == "udp" && listener.HealthCheckReq != "" {
							udpCheckEnabled = true
						}
					}
					backendGroup := lb.backendGroups[listener.BackendGroupId]
					for _, backend := range backendGroup.backends {
						dataReal := map[string]interface{}{
							"real_ip":   backend.Address,
							"real_port": backend.Port,
							"weight":    backend.Weight,
						}
						if udpCheckEnabled {
							checkMiscArgs := []string{
								keepalivedMiscCheckPath,
								fmt.Sprintf("wait=%d", listener.HealthCheckTimeout),
								fmt.Sprintf("req=%s", listener.HealthCheckReq),
								fmt.Sprintf("exp=%s", listener.HealthCheckExp),
								fmt.Sprintf("host=%s", backend.Address),
								fmt.Sprintf("port=%d", backend.Port),
							}
							dataCheckMiscPath := agentutils.KeepalivedConfQuoteScriptArgs(checkMiscArgs)
							dataCheck := map[string]interface{}{
								"misc_path":    dataCheckMiscPath,
								"misc_timeout": listener.HealthCheckTimeout,
								"interval":     listener.HealthCheckInterval,
								"fall":         listener.HealthCheckFall,
								"usergroup":    "nobody",
							}
							dataReal["check"] = dataCheck
						}
						dataReals = append(dataReals, dataReal)
					}
					dataVirtual["real_servers"] = dataReals

					fmt.Fprintf(buf, "\n\n")
					fmt.Fprintf(buf, "## listener %s(%s) backendGroup %s(%s)\n",
						listener.Name, listener.Id,
						backendGroup.Name, backendGroup.Id)
					keepalivedConfTmpl.ExecuteTemplate(buf, "keepalivedVirtualServerUDP", dataVirtual)
				}
			}
		}
	}
	{
		// write keepalived.conf
		d := buf.Bytes()
		p := filepath.Join(dir, "keepalived.conf")
		err := ioutil.WriteFile(p, d, agentutils.FileModeFile)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO retry for down, up?
var keepalivedConfTmpl = template.Must(template.New("").Parse(`
{{- define "keepalivedVirtualServerUDP" -}}
virtual_server {{ .virtual_ip }} {{ .virtual_port }} {
	protocol UDP
	lvs_method NAT
	lvs_sched {{ .scheduler }}
	ha_suspend
	{{- range .real_servers }}
	real_server {{ .real_ip }} {{ .real_port }} {
		weight {{ .weight }}
		inhibit_on_failure
		{{- if .check }}
		MISC_CHECK {
			misc_path {{ .check.misc_path }}
			misc_timeout {{ .check.misc_timeout }}
			delay_loop {{ .check.interval }}
			retry {{ .check.fall }}
			user {{ .check.usergroup }}
		}
		{{- end }}
	}
	{{- end }}
}
{{ end }}
`))
