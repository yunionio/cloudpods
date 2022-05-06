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
	"io/ioutil"
	"path/filepath"
	"text/template"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
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
			if lb.NetworkType == computeapi.LB_NETWORK_TYPE_VPC {
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
