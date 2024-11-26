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
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	TELEGRAF_INPUT_RADEONTOP           = "radeontop"
	TELEGRAF_INPUT_RADEONTOP_DEV_PATHS = "device_paths"
	TELEGRAF_INPUT_CONF_BIN_PATH       = "bin_path"
	TELEGAF_INPUT_NETDEV               = "ni_rsrc_mon"
	TELEGAF_INPUT_VASMI                = "vasmi"
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
		keys := []string{}
		for k := range tgs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			conf += fmt.Sprintf("  %s = \"%s\"\n", k, tgs[k])
		}
	}
	conf += "\n"
	conf += "[agent]\n"
	intVal := 60
	if v, ok := kwargs["interval"]; ok {
		intvalInt, _ := v.(int)
		if intvalInt > 0 {
			intVal = intvalInt
		}
	}
	conf += fmt.Sprintf("  interval = \"%ds\"\n", intVal)
	conf += "  round_interval = true\n"
	conf += "  metric_batch_size = 1000\n"
	conf += "  metric_buffer_limit = 10000\n"
	conf += "  collection_jitter = \"0s\"\n"
	conf += "  flush_interval = \"60s\"\n"
	conf += "  flush_jitter = \"0s\"\n"
	conf += "  precision = \"\"\n"
	conf += "  debug = false\n"
	conf += "  quiet = false\n"
	// conf += "  logfile = \"/var/log/telegraf/telegraf.err.log\"\n"
	var hostname string
	if hn, ok := kwargs["hostname"]; ok {
		hostname, _ = hn.(string)
	}
	conf += fmt.Sprintf("  hostname = \"%s\"\n", hostname)
	conf += "  omit_hostname = false\n"
	conf += "\n"
	if influx, ok := kwargs[apis.SERVICE_TYPE_INFLUXDB]; ok {
		influxdb, _ := influx.(map[string]interface{})
		inUrls, _ := influxdb["url"]
		tUrls, _ := inUrls.([]string)
		inDatabase, _ := influxdb["database"]
		isVM := false
		if tsdbType, ok := influxdb["tsdb_type"]; ok {
			if tsdbType.(string) == apis.SERVICE_TYPE_VICTORIA_METRICS {
				isVM = true
			}
		}
		tdb, _ := inDatabase.(string)
		urls := []string{}
		for _, u := range tUrls {
			urls = append(urls, fmt.Sprintf("\"%s\"", u))
		}
		conf += "[[outputs.influxdb]]\n"
		conf += fmt.Sprintf("  urls = [%s]\n", strings.Join(urls, ", "))
		conf += fmt.Sprintf("  database = \"%s\"\n", tdb)
		conf += "  insecure_skip_verify = true\n"
		if isVM {
			conf += "  skip_database_creation = true\n"
		}
		conf += "  timeout = \"30s\"\n"
		conf += "\n"
	}
	/*
	 *
	 * [[outputs.kafka]]
	 *   ## URLs of kafka brokers
	 *   brokers = ["localhost:9092"]
	 *   ## Kafka topic for producer messages
	 *   topic = "telegraf"
	 *   ## Optional SASL Config
	 *   sasl_username = "kafka"
	 *   sasl_password = "secret"
	 *   ## Optional SASL:
	 *   ## one of: OAUTHBEARER, PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, GSSAPI
	 *   ## (defaults to PLAIN)
	 *   sasl_mechanism = "PLAIN"
	 */
	if kafka, ok := kwargs["kafka"]; ok {
		kafkaConf, _ := kafka.(map[string]interface{})
		conf += "[[outputs.kafka]]\n"
		for _, k := range []string{
			"brokers",
			"topic",
			"sasl_username",
			"sasl_password",
			"sasl_mechanism",
		} {
			if val, ok := kafkaConf[k]; ok {
				if k == "brokers" {
					brokers, _ := val.([]string)
					for i := range brokers {
						brokers[i] = fmt.Sprintf("\"%s\"", brokers[i])
					}
					conf += fmt.Sprintf("  brokers = [%s]\n", strings.Join(brokers, ", "))
				} else {
					conf += fmt.Sprintf("  %s = \"%s\"\n", k, val)
				}
			}
		}
		conf += "  compression_codec = 0\n"
		conf += "  required_acks = -1\n"
		conf += "  max_retry = 3\n"
		conf += "  data_format = \"json\"\n"
		conf += "  json_timestamp_units = \"1ms\"\n"
		conf += "  routing_tag = \"host\"\n"
		conf += "\n"
	}
	/*
	 * [[outputs.opentsdb]]
	 *   host = "http://127.0.0.1"
	 *   port = 17000
	 *   http_batch_size = 50
	 *   http_path = "/opentsdb/put"
	 *   debug = false
	 *   separator = "_"
	 */
	if opentsdb, ok := kwargs["opentsdb"]; ok {
		opentsdbConf, _ := opentsdb.(map[string]interface{})
		urlstr := opentsdbConf["url"].(string)
		urlParts, err := url.Parse(urlstr)
		if err != nil {
			log.Errorf("malformed opentsdb url: %s: %s", urlstr, err)
		} else {
			port := urlParts.Port()
			if len(port) == 0 {
				if urlParts.Scheme == "http" {
					port = "80"
				} else if urlParts.Scheme == "https" {
					port = "443"
				}
			}
			conf += "[[outputs.opentsdb]]\n"
			conf += fmt.Sprintf("  host = \"%s://%s\"\n", urlParts.Scheme, urlParts.Hostname())
			conf += fmt.Sprintf("  port = %s\n", urlParts.Port())
			conf += "  http_batch_size = 50\n"
			conf += fmt.Sprintf("  http_path = \"%s\"\n", urlParts.Path)
			conf += "  debug = false\n"
			conf += "  separator = \"_\"\n"
			conf += "\n"
		}
	}
	conf += "[[inputs.cpu]]\n"
	conf += "  percpu = true\n"
	conf += "  totalcpu = true\n"
	conf += "  collect_cpu_time = false\n"
	conf += "  report_active = true\n"
	conf += "\n"
	conf += "[[inputs.disk]]\n"
	ignoreMountPoints := []string{
		"/etc/telegraf",
		"/etc/hosts",
		"/etc/hostname",
		"/etc/resolv.conf",
		"/dev/termination-log",
	}
	for i := range ignoreMountPoints {
		ignoreMountPoints[i] = fmt.Sprintf("%q", ignoreMountPoints[i])
	}
	ignorePathSegments := []string{
		"/run/k3s/containerd/",
	}
	ignorePathSegments = append(ignorePathSegments, kwargs["server_path"].(string))
	for i := range ignorePathSegments {
		ignorePathSegments[i] = fmt.Sprintf("%q", ignorePathSegments[i])
	}
	conf += "  ignore_mount_points = [" + strings.Join(ignoreMountPoints, ", ") + "]\n"
	conf += "  ignore_path_segments = [" + strings.Join(ignorePathSegments, ", ") + "]\n"
	conf += "  ignore_fs = [\"tmpfs\", \"devtmpfs\", \"devfs\", \"overlay\", \"squashfs\", \"iso9660\", \"rootfs\", \"hugetlbfs\", \"autofs\", \"aufs\"]\n"
	conf += "\n"
	conf += "[[inputs.diskio]]\n"
	conf += "  skip_serial_number = false\n"
	conf += "  excludes = \"^(nbd|loop)\"\n"
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
	conf += "[[inputs.smart]]\n"
	conf += "  path=\"/usr/sbin/smartctl\"\n"
	conf += "\n"
	conf += "[[inputs.sensors]]\n"
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
			name, _ := n["name"].(string)
			alias, _ := n["alias"].(string)
			speed, _ := n["speed"].(int)

			conf += "  [[inputs.net.interface_conf]]\n"
			conf += fmt.Sprintf("    name = \"%s\"\n", name)
			conf += fmt.Sprintf("    alias = \"%s\"\n", alias)
			conf += fmt.Sprintf("    speed = %d\n", speed)
			conf += "\n"
		}
	}
	conf += "[[inputs.netstat]]\n"
	conf += "\n"
	conf += "[[inputs.bond]]\n"
	conf += "\n"
	conf += "[[inputs.temp]]\n"
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
	conf += "[[inputs.linux_sysctl_fs]]\n"
	conf += "\n"
	conf += "[[inputs.http_listener_v2]]\n"
	conf += "  service_address = \"127.0.0.1:8087\"\n"
	conf += "  path = \"/write\"\n"
	conf += "  data_source = \"body\"\n"
	conf += "  data_format = \"influx\"\n"
	conf += "\n"
	if haproxyConf, ok := kwargs["haproxy"]; ok {
		haproxyConfMap, _ := haproxyConf.(map[string]interface{})
		haIntVal, _ := haproxyConfMap["interval"].(int)
		statsSocket, _ := haproxyConfMap["stats_socket_path"].(string)
		conf += "[[inputs.haproxy]]\n"
		conf += fmt.Sprintf("  interval = \"%ds\"\n", haIntVal)
		conf += fmt.Sprintf("  servers = [\"%s\"]\n", statsSocket)
		conf += "  keep_field_names = true\n"
		conf += "\n"
	}

	if radontop, ok := kwargs[TELEGRAF_INPUT_RADEONTOP]; ok {
		radontopMap, _ := radontop.(map[string]interface{})
		devPaths := radontopMap[TELEGRAF_INPUT_RADEONTOP_DEV_PATHS].([]string)
		devPathStr := make([]string, len(devPaths))
		for i, devPath := range devPaths {
			devPathStr[i] = fmt.Sprintf("\"%s\"", devPath)
		}
		conf += fmt.Sprintf("[[inputs.%s]]\n", TELEGRAF_INPUT_RADEONTOP)
		conf += fmt.Sprintf("  bin_path = \"%s\"\n", radontopMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string))
		conf += fmt.Sprintf("  %s = [%s]\n", TELEGRAF_INPUT_RADEONTOP_DEV_PATHS, strings.Join(devPathStr, ", "))
		conf += "\n"
	}

	if netdev, ok := kwargs[TELEGAF_INPUT_NETDEV]; ok {
		netdevMap, _ := netdev.(map[string]interface{})
		conf += fmt.Sprintf("[[inputs.%s]]\n", TELEGAF_INPUT_NETDEV)
		conf += fmt.Sprintf("  bin_path = \"%s\"\n", netdevMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string))
		conf += "\n"
	}

	if vasmi, ok := kwargs[TELEGAF_INPUT_VASMI]; ok {
		vasmiMap, _ := vasmi.(map[string]interface{})
		conf += fmt.Sprintf("[[inputs.%s]]\n", TELEGAF_INPUT_VASMI)
		conf += fmt.Sprintf("  bin_path = \"%s\"\n", vasmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string))
		conf += "\n"
	}
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

func (s *STelegraf) BgReloadConf(kwargs map[string]interface{}) {
	go func() {
		reload, err := s.reloadConf(s.GetConfig(kwargs), s.GetConfigFile())
		if err != nil {
			log.Errorf("Failed reload conf: %s", err)
		}
		if reload {
			err := s.ReloadTelegraf()
			if err != nil {
				log.Errorf("failed reload telegraf: %s", err)
			}
		}
	}()
}

func (s *STelegraf) ReloadTelegraf() error {
	log.Infof("Start reloading telegraf...")
	errs := []error{}
	if err := s.reloadTelegrafByDocker(); err != nil {
		errs = append(errs, errors.Wrap(err, "reloadTelegrafByDocker"))
		if err := s.reloadTelegrafByHTTP(); err != nil {
			errs = append(errs, errors.Wrap(err, "reloadTelegrafByHTTP"))
			return errors.NewAggregate(errs)
		}
	}
	log.Infof("Finish reloading telegraf")
	return nil
}

func (s *STelegraf) reloadTelegrafByDocker() error {
	log.Infof("Reloading telegraf by docker...")
	output, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", "/usr/bin/docker ps --filter 'label=io.kubernetes.container.name=telegraf' --format '{{.ID}}'").Output()
	if err != nil {
		return errors.Wrap(err, "using docker ps find telegraf container")
	}
	id := strings.TrimSpace(string(output))
	if len(id) == 0 {
		return errors.Errorf("not found telegraf running container")
	}
	if err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", "/usr/bin/docker restart "+id).Run(); err != nil {
		return errors.Wrapf(err, "restart telegraf container %q", id)
	}
	return nil
}

func (s *STelegraf) reloadTelegrafByHTTP() error {
	telegrafReoladUrl := "http://127.0.0.1:8087/reload"
	log.Infof("Reloading telegraf by %q ...", telegrafReoladUrl)
	if _, _, err := httputils.JSONRequest(
		httputils.GetDefaultClient(), context.Background(),
		"POST", telegrafReoladUrl, nil, nil, false,
	); err != nil {
		return errors.Wrap(err, "reload telegraf by http reload api")
	}
	return nil
}
