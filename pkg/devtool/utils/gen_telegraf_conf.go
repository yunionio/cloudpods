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
	"strings"
	"text/template"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/osprofile"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func GetValFromMap(valMap map[string]string, key string) string {
	return valMap[key]
}

// copy from: https://github.com/yunionio/ansible-telegraf/blob/master/templates/telegraf.conf.j2
const TELEGRAF_CONF_TEMPLETE = `### MANAGED BY ansible-telegraf ANSIBLE ROLE ###

[global_tags]
{{ range $value := .telegraf_global_tags }}
    {{ GetValFromMap $value "tag_name" }} = "{{ GetValFromMap $value "tag_value" }}"
{{- end }}

# Configuration for telegraf agent
[agent]
    interval = "60s"
    debug = false
    hostname = ""
    round_interval = true
    flush_interval = "60s"
    flush_jitter = "0s"
    collection_jitter = "0s"
    metric_batch_size = 1000
    metric_buffer_limit = 10000
    quiet = false
    logfile = "{{ .telegraf_agent_logfile }}"
    logfile_rotation_max_size = "10MB"
    logfile_rotation_max_archives = 1
    omit_hostname = true

###############################################################################
#                                  OUTPUTS                                    #
###############################################################################

[[outputs.influxdb]]
    urls = ["{{ .influxdb_url }}"]
    database = "{{ .influxdb_name }}"
    insecure_skip_verify = true

###############################################################################
#                                  INPUTS                                     #
###############################################################################`

const TELEGRAF_INPUT_LINUX = `
[[inputs.cpu]]
    name_prefix = "agent_"
    percpu = true
    totalcpu = true
    collect_cpu_time = false
    report_active = true
[[inputs.disk]]
    name_prefix = "agent_"
    ignore_fs = ["tmpfs", "devtmpfs", "overlay", "squashfs", "iso9660"]
[[inputs.diskio]]
    name_prefix = "agent_"
    skip_serial_number = false
[[inputs.kernel]]
    name_prefix = "agent_"
[[inputs.kernel_vmstat]]
    name_prefix = "agent_"
[[inputs.mem]]
    name_prefix = "agent_"
[[inputs.processes]]
    name_prefix = "agent_"
[[inputs.swap]]
    name_prefix = "agent_"
[[inputs.system]]
    name_prefix = "agent_"
[[inputs.net]]
    name_prefix = "agent_"
[[inputs.netstat]]
    name_prefix = "agent_"
[[inputs.nstat]]
    name_prefix = "agent_"
[[inputs.internal]]
    name_prefix = "agent_"
    collect_memstats = false
[[inputs.nvidia_smi]]
    name_prefix = "agent_"
`

const TELEGRAF_INPUT_WINDOWS = `
[[inputs.cpu]]
    name_prefix = "agent_"
    percpu = true
    totalcpu = true
    collect_cpu_time = false
    report_active = true
[[inputs.disk]]
    name_prefix = "agent_"
    ignore_fs = ["tmpfs", "devtmpfs", "overlay", "squashfs", "iso9660"]
[[inputs.diskio]]
    name_prefix = "agent_"
    skip_serial_number = false
[[inputs.mem]]
    name_prefix = "agent_"
[[inputs.processes]]
    name_prefix = "agent_"
[[inputs.swap]]
    name_prefix = "agent_"
[[inputs.system]]
    name_prefix = "agent_"
[[inputs.net]]
    name_prefix = "agent_"
[[inputs.netstat]]
    name_prefix = "agent_"
[[inputs.nstat]]
    name_prefix = "agent_"
[[inputs.internal]]
    name_prefix = "agent_"
    collect_memstats = false
[[inputs.nvidia_smi]]
    name_prefix = "agent_"
`

const TELEGRAF_INPUT_BAREMETAL = `
[[inputs.cpu]]
    name_prefix = "agent_"
    percpu = true
    totalcpu = true
    collect_cpu_time = false
    report_active = true
[[inputs.disk]]
    name_prefix = "agent_"
    ignore_fs = ["tmpfs", "devtmpfs", "overlay", "squashfs", "iso9660"]
[[inputs.diskio]]
    name_prefix = "agent_"
    skip_serial_number = false
[[inputs.sensors]]
    name_prefix = "agent_"
[[inputs.smart]]
    name_prefix = "agent_"
    use_sudo = true
[[inputs.mem]]
    name_prefix = "agent_"
[[inputs.processes]]
    name_prefix = "agent_"
[[inputs.swap]]
    name_prefix = "agent_"
[[inputs.system]]
    name_prefix = "agent_"
[[inputs.net]]
    name_prefix = "agent_"
[[inputs.netstat]]
    name_prefix = "agent_"
[[inputs.nstat]]
    name_prefix = "agent_"
[[inputs.internal]]
    name_prefix = "agent_"
    collect_memstats = false
[[inputs.nvidia_smi]]
    name_prefix = "agent_"
`

var temp *template.Template

func init() {
	var err error
	temp, err = template.New("").Funcs(template.FuncMap{
		"GetValFromMap": GetValFromMap,
	}).Parse(TELEGRAF_CONF_TEMPLETE)
	if err != nil {
		log.Fatalf("parse telegraf template: %s", err)
	}
}

func getTelegrafInputs(hypervisor, osType string) string {
	if hypervisor == api.HYPERVISOR_BAREMETAL {
		return TELEGRAF_INPUT_BAREMETAL
	} else {
		if osType == osprofile.OS_TYPE_WINDOWS {
			return TELEGRAF_INPUT_WINDOWS
		} else {
			return TELEGRAF_INPUT_LINUX
		}
	}
}

func GenerateTelegrafConf(
	serverDetails *api.ServerDetails, influxdbUrl, osType, hypervisor string,
) (string, error) {
	telegrafArgs := GetLocalArgs(serverDetails, influxdbUrl)
	if osType == osprofile.OS_TYPE_WINDOWS {
		telegrafArgs["telegraf_agent_logfile"] = "/Program Files/Telegraf/telegraf.log"
	} else {
		telegrafArgs["telegraf_agent_logfile"] = "/var/log/telegraf.log"
	}
	strBuild := strings.Builder{}
	log.Infof("telegraf args %v", telegrafArgs)
	err := temp.Execute(&strBuild, telegrafArgs)
	if err != nil {
		return "", errors.Wrap(err, "build telegraf config")
	}
	return strBuild.String() + getTelegrafInputs(hypervisor, osType), nil
}
