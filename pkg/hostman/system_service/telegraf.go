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

package system_service

import (
	"fmt"
	"strings"
)

type STelegraf struct {
	*SBaseSystemService
}

func NewTelegrafService() *STelegraf {
	return &STelegraf{NewBaseSystemService("telegraf", nil)}
}

func (s *STelegraf) GetConfig(kwargs map[string]interface{}) string {
	conf := ""
	conf += "[global_tags]\n"
	if tags, ok := kwargs["tags"]; ok {
		tgs, _ := tags.(map[string]string)
		for k, v := range tgs {
			conf += fmt.Sprintf("  %s = \"%s\"\n", k, v)
		}
	}
	conf += "\n"
	conf += "[agent]\n"
	conf += "  interval = \"60s\"\n"
	conf += "  round_interval = true\n"
	conf += "  metric_batch_size = 1000\n"
	conf += "  metric_buffer_limit = 10000\n"
	conf += "  collection_jitter = \"0s\"\n"
	conf += "  flush_interval = \"60s\"\n"
	conf += "  flush_jitter = \"0s\"\n"
	conf += "  precision = \"\"\n"
	conf += "  debug = false\n"
	conf += "  quiet = false\n"
	conf += "  logfile = \"/var/log/telegraf/telegraf.err.log\"\n"
	var hostname string
	if hn, ok := kwargs["hostname"]; ok {
		hostname, _ = hn.(string)
	}
	conf += fmt.Sprintf("  hostname = \"%s\"\n", hostname)
	conf += "  omit_hostname = false\n"
	conf += "\n"
	if ifluxb, ok := kwargs["influxdb"]; ok {
		influxdb, _ := ifluxb.(map[string]interface{})
		inUrls, _ := influxdb["url"]
		tUrls, _ := inUrls.([]string)
		inDatabase, _ := influxdb["database"]
		tdb, _ := inDatabase.(string)
		urls := []string{}
		for _, u := range tUrls {
			urls = append(urls, fmt.Sprintf("\"%s\"", u))
		}
		conf += "[[outputs.influxdb]]\n"
		conf += fmt.Sprintf("  urls = [%s]\n", strings.Join(urls, ", "))
		conf += fmt.Sprintf("  database = \"%s\"\n", tdb)
		conf += "  insecure_skip_verify = true\n"
		conf += "\n"
	}
	if kafka, ok := kwargs["kafka"]; ok {
		ka, _ := kafka.(map[string]interface{})
		bks, _ := ka["brokers"]
		tbk, _ := bks.([]string)
		brokers := []string{}
		for _, b := range tbk {
			brokers = append(brokers, fmt.Sprintf("\"%s\"", b[len("kafka://\n"):]))
		}
		conf += "[[outputs.kafka]]\n"
		conf += fmt.Sprintf("  brokers = [%s]\n", strings.Join(brokers, ", "))

		topic, _ := ka["topic"]
		itopic, _ := topic.(string)
		conf += fmt.Sprintf("  topic = \"%s\"\n", itopic)
		conf += "  compression_codec = 0\n"
		conf += "  required_acks = -1\n"
		conf += "  max_retry = 3\n"
		conf += "  data_format = \"json\"\n"
		conf += "  json_timestamp_units = \"1ms\"\n"
		conf += "  routing_tag = \"host\"\n"
		conf += "\n"
	}
	conf += "[[inputs.cpu]]\n"
	conf += "  percpu = false\n"
	conf += "  totalcpu = true\n"
	conf += "  collect_cpu_time = false\n"
	conf += "  report_active = true\n"
	conf += "\n"
	conf += "[[inputs.disk]]\n"
	conf += "  ignore_fs = [\"tmpfs\", \"devtmpfs\", \"overlay\", \"squashfs\", \"iso9660\"]\n"
	conf += "\n"
	conf += "[[inputs.diskio]]\n"
	conf += "  skip_serial_number = false\n"
	conf += "  excludes = \"^nbd\"\n"
	conf += "\n"
	conf += "[[inputs.kernel]]\n"
	conf += "\n"
	conf += "[[inputs.kernel_vmstat]]\n"
	conf += "\n"
	conf += "[[inputs.mem]]\n"
	conf += "\n"
	conf += "[[inputs.processes]]\n"
	conf += "\n"
	conf += "[[inputs.swap]]\n"
	conf += "\n"
	conf += "[[inputs.system]]\n"
	conf += "\n"
	conf += "[[inputs.net]]\n"
	if nics, ok := kwargs["nics"]; ok {
		ns, _ := nics.([]map[string]interface{})
		infs := []string{}
		for _, n := range ns {
			iname, _ := n["name"]
			name, _ := iname.(string)
			infs = append(infs, fmt.Sprintf("\"%s\"", name))
		}
		conf += fmt.Sprintf("  interfaces = [%s]\n", strings.Join(infs, ", "))
		conf += "\n"
		for _, n := range ns {
			iname, _ := n["name"]
			name, _ := iname.(string)
			ialias, _ := n["alias"]
			alias, _ := ialias.(string)
			ispeed, _ := n["speed"]
			speed, _ := ispeed.(int)

			conf += "  [[inputs.net.interface_conf]]\n"
			conf += fmt.Sprintf("    name = \"%s\"\n", name)
			conf += fmt.Sprintf("    alias = \"%s\"\n", alias)
			conf += fmt.Sprintf("    speed = %d\n", speed)
			conf += "\n"
		}
	}
	conf += "[[inputs.netstat]]\n"
	conf += "\n"
	conf += "[[inputs.nstat]]\n"
	conf += "\n"
	conf += "[[inputs.ntpq]]\n"
	conf += "  dns_lookup = false\n"
	conf += "\n"
	if pidFile, ok := kwargs["pid_file"]; ok {
		pf, _ := pidFile.(string)
		conf += "[[inputs.procstat]]\n"
		conf += fmt.Sprintf("  pid_file = \"%s\"\n", pf)
		conf += "\n"
	}
	conf += "[[inputs.internal]]\n"
	conf += "  collect_memstats = false\n"
	conf += "\n"
	conf += "[[inputs.http_listener]]\n"
	conf += "  service_address = \"localhost:8087\"\n"
	conf += "\n"
	return conf
}

func (s *STelegraf) GetConfigFile() string {
	return "/etc/telegraf/telegraf.conf"
}

func (s *STelegraf) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *STelegraf) BgReload(kwargs map[string]interface{}) {
	go s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}
