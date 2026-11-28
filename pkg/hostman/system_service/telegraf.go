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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/procutils"
	telegrafutil "yunion.io/x/onecloud/pkg/util/telegraf"
)

const (
	TELEGRAF_INPUT_RADEONTOP           = "radeontop"
	TELEGRAF_INPUT_ROCM_SMI            = "rocm-smi"
	TELEGRAF_INPUT_RADEONTOP_DEV_PATHS = "device_paths"
	TELEGRAF_INPUT_CONF_BIN_PATH       = "bin_path"
	TELEGRAF_INPUT_NETDEV              = "ni_rsrc_mon"
	TELEGRAF_INPUT_VASMI               = "vasmi"
	TELEGRAF_INPUT_HYSMI               = "hysmi"
	TELEGRAF_INPUT_IXSMI               = "ixsmi"
	TELEGRAF_INPUT_PPUSMI              = "ppusmi"
	TELEGRAF_INPUT_XPUSMI              = "xpusmi"
	TELEGRAF_INPUT_MTHREADSGMI         = "mthreadsgmi"
	TELEGRAF_INPUT_NVIDIASMI           = "nvidia-smi"
	TELEGRAF_INPUT_NPUSMI              = "npu-smi"
	TELEGRAF_INPUT_CONF_LIB_PATH       = "lib_path"
)

type STelegraf struct {
	*SBaseSystemService
}

func NewTelegrafService() *STelegraf {
	return &STelegraf{NewBaseSystemService("telegraf", nil)}
}

func (s *STelegraf) GetConfig(kwargs map[string]interface{}) string {
	config := telegrafutil.Config{
		GlobalTags: telegrafutil.ConfigSection{Name: "global_tags"},
		Agent:      telegrafutil.ConfigSection{Name: "agent"},
	}
	if tags, ok := kwargs["tags"]; ok {
		tgs, _ := tags.(map[string]string)
		keys := []string{}
		for k := range tgs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			config.GlobalTags.Options = append(config.GlobalTags.Options, telegrafutil.ConfigOption{Key: k, Value: tgs[k]})
		}
	}
	intVal := 60
	if v, ok := kwargs["interval"]; ok {
		intvalInt, _ := v.(int)
		if intvalInt > 0 {
			intVal = intvalInt
		}
	}
	config.Agent.Options = append(config.Agent.Options,
		telegrafutil.ConfigOption{Key: "interval", Value: fmt.Sprintf("%ds", intVal)},
		telegrafutil.ConfigOption{Key: "round_interval", Value: true},
		telegrafutil.ConfigOption{Key: "metric_batch_size", Value: 1000},
		telegrafutil.ConfigOption{Key: "metric_buffer_limit", Value: 10000},
		telegrafutil.ConfigOption{Key: "collection_jitter", Value: "0s"},
		telegrafutil.ConfigOption{Key: "flush_interval", Value: "60s"},
		telegrafutil.ConfigOption{Key: "flush_jitter", Value: "0s"},
		telegrafutil.ConfigOption{Key: "precision", Value: ""},
		telegrafutil.ConfigOption{Key: "debug", Value: false},
		telegrafutil.ConfigOption{Key: "quiet", Value: false})
	// conf += "  logfile = \"/var/log/telegraf/telegraf.err.log\"\n"
	var hostname string
	if hn, ok := kwargs["hostname"]; ok {
		hostname, _ = hn.(string)
	}
	config.Agent.Options = append(config.Agent.Options,
		telegrafutil.ConfigOption{Key: "hostname", Value: hostname},
		telegrafutil.ConfigOption{Key: "omit_hostname", Value: false})
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
		output := telegrafutil.ConfigSection{Name: "outputs.influxdb", Array: true}
		output.Options = append(output.Options,
			telegrafutil.ConfigOption{Key: "urls", Value: tUrls},
			telegrafutil.ConfigOption{Key: "database", Value: tdb},
			telegrafutil.ConfigOption{Key: "insecure_skip_verify", Value: true})
		if isVM {
			output.Options = append(output.Options, telegrafutil.ConfigOption{Key: "skip_database_creation", Value: true})
		}
		output.Options = append(output.Options, telegrafutil.ConfigOption{Key: "timeout", Value: "30s"})
		config.Outputs = append(config.Outputs, output)
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
		output := telegrafutil.ConfigSection{Name: "outputs.kafka", Array: true}
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
					output.Options = append(output.Options, telegrafutil.ConfigOption{Key: "brokers", Value: brokers})
				} else {
					output.Options = append(output.Options, telegrafutil.ConfigOption{Key: k, Value: fmt.Sprint(val)})
				}
			}
		}
		output.Options = append(output.Options,
			telegrafutil.ConfigOption{Key: "compression_codec", Value: 0},
			telegrafutil.ConfigOption{Key: "required_acks", Value: -1},
			telegrafutil.ConfigOption{Key: "max_retry", Value: 3},
			telegrafutil.ConfigOption{Key: "data_format", Value: "json"},
			telegrafutil.ConfigOption{Key: "json_timestamp_units", Value: "1ms"},
			telegrafutil.ConfigOption{Key: "routing_tag", Value: "host"})
		config.Outputs = append(config.Outputs, output)
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
			config.Outputs = append(config.Outputs, telegrafutil.ConfigSection{
				Name: "outputs.opentsdb", Array: true,
				Options: []telegrafutil.ConfigOption{
					{Key: "host", Value: fmt.Sprintf("%s://%s", urlParts.Scheme, urlParts.Hostname())},
					{Key: "port", Value: urlParts.Port(), RawValue: true},
					{Key: "http_batch_size", Value: 50},
					{Key: "http_path", Value: urlParts.Path},
					{Key: "debug", Value: false},
					{Key: "separator", Value: "_"},
				},
			})
		}
	}
	config.Inputs = append(config.Inputs, telegrafutil.ConfigSection{
		Name: "inputs.cpu", Array: true,
		Options: []telegrafutil.ConfigOption{
			{Key: "percpu", Value: true}, {Key: "totalcpu", Value: true},
			{Key: "collect_cpu_time", Value: false}, {Key: "report_active", Value: true},
		},
	})
	ignoreMountPoints := []string{
		"/etc/telegraf",
		"/etc/hosts",
		"/etc/hostname",
		"/etc/resolv.conf",
		"/dev/termination-log",
	}
	ignorePathSegments := []string{
		"/run/k3s/containerd/",
		"/run/onecloud/containerd/",
		"/var/lib/",
	}
	if sp, ok := kwargs["server_path"]; ok {
		ignorePathSegments = append(ignorePathSegments, sp.(string))
	}
	config.Inputs = append(config.Inputs, telegrafutil.ConfigSection{
		Name: "inputs.disk", Array: true,
		Options: []telegrafutil.ConfigOption{
			{Key: "ignore_mount_points", Value: ignoreMountPoints},
			{Key: "ignore_path_segments", Value: ignorePathSegments},
			{Key: "ignore_fs", Value: []string{"devtmpfs", "devfs", "overlayfs", "overlay", "squashfs", "iso9660", "rootfs", "hugetlbfs", "autofs", "aufs"}},
		},
	})
	config.Inputs = append(config.Inputs,
		telegrafutil.ConfigSection{Name: "inputs.diskio", Array: true, Options: []telegrafutil.ConfigOption{{Key: "skip_serial_number", Value: false}, {Key: "excludes", Value: "^(nbd|loop)"}}},
		telegrafutil.ConfigSection{Name: "inputs.kernel", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.kernel_vmstat", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.mem", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.processes", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.swap", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.system", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.smart", Array: true, Options: []telegrafutil.ConfigOption{{Key: "path", Value: "/usr/sbin/smartctl"}}},
		telegrafutil.ConfigSection{Name: "inputs.sensors", Array: true})
	netInput := telegrafutil.ConfigSection{Name: "inputs.net", Array: true}
	if nics, ok := kwargs["nics"]; ok {
		ns, _ := nics.([]map[string]interface{})
		infs := []string{}
		for _, n := range ns {
			iname, _ := n["name"]
			name, _ := iname.(string)
			infs = append(infs, name)
		}
		netInput.Options = append(netInput.Options, telegrafutil.ConfigOption{Key: "interfaces", Value: infs})
		for _, n := range ns {
			name, _ := n["name"].(string)
			alias, _ := n["alias"].(string)
			speed, _ := n["speed"].(int)

			netInput.Children = append(netInput.Children, telegrafutil.ConfigSection{
				Name: "inputs.net.interface_conf", Array: true, Nested: true,
				Options: []telegrafutil.ConfigOption{
					{Key: "name", Value: name}, {Key: "alias", Value: alias}, {Key: "speed", Value: speed},
				},
			})
		}
	}
	config.Inputs = append(config.Inputs, netInput)
	config.Inputs = append(config.Inputs,
		telegrafutil.ConfigSection{Name: "inputs.netstat", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.bond", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.temp", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.nstat", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.ntpq", Array: true, Options: []telegrafutil.ConfigOption{{Key: "dns_lookup", Value: false}}})
	if pidFile, ok := kwargs["pid_file"]; ok {
		pf, _ := pidFile.(string)
		config.Inputs = append(config.Inputs, telegrafutil.ConfigSection{Name: "inputs.procstat", Array: true, Options: []telegrafutil.ConfigOption{{Key: "pid_file", Value: pf}}})
	}
	config.Inputs = append(config.Inputs,
		telegrafutil.ConfigSection{Name: "inputs.internal", Array: true, Options: []telegrafutil.ConfigOption{{Key: "collect_memstats", Value: false}}},
		telegrafutil.ConfigSection{Name: "inputs.linux_sysctl_fs", Array: true},
		telegrafutil.ConfigSection{Name: "inputs.http_listener_v2", Array: true, Options: []telegrafutil.ConfigOption{
			{Key: "service_address", Value: "localhost:8087"}, {Key: "path", Value: "/write"},
			{Key: "paths", Value: []string{"/write"}}, {Key: "data_source", Value: "body"}, {Key: "data_format", Value: "influx"},
		}})
	if haproxyConf, ok := kwargs["haproxy"]; ok {
		haproxyConfMap, _ := haproxyConf.(map[string]interface{})
		haIntVal, _ := haproxyConfMap["interval"].(int)
		statsSocket, _ := haproxyConfMap["stats_socket_path"].(string)
		config.Inputs = append(config.Inputs, telegrafutil.ConfigSection{Name: "inputs.haproxy", Array: true, Options: []telegrafutil.ConfigOption{
			{Key: "interval", Value: fmt.Sprintf("%ds", haIntVal)}, {Key: "servers", Value: []string{statsSocket}}, {Key: "keep_field_names", Value: true},
		}})
	}

	addInput := func(name string, options ...telegrafutil.ConfigOption) {
		config.Inputs = append(config.Inputs, telegrafutil.ConfigSection{
			Name: "inputs." + name, Array: true, Options: options,
		})
	}
	if radontop, ok := kwargs[TELEGRAF_INPUT_RADEONTOP]; ok {
		radontopMap, _ := radontop.(map[string]interface{})
		devPaths := radontopMap[TELEGRAF_INPUT_RADEONTOP_DEV_PATHS].([]string)
		addInput(TELEGRAF_INPUT_RADEONTOP,
			telegrafutil.ConfigOption{Key: "bin_path", Value: radontopMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)},
			telegrafutil.ConfigOption{Key: TELEGRAF_INPUT_RADEONTOP_DEV_PATHS, Value: devPaths})
	}

	if netdev, ok := kwargs[TELEGRAF_INPUT_NETDEV]; ok {
		netdevMap, _ := netdev.(map[string]interface{})
		addInput(TELEGRAF_INPUT_NETDEV, telegrafutil.ConfigOption{Key: "bin_path", Value: netdevMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)})
	}

	if vasmi, ok := kwargs[TELEGRAF_INPUT_VASMI]; ok {
		vasmiMap, _ := vasmi.(map[string]interface{})
		addInput(TELEGRAF_INPUT_VASMI, telegrafutil.ConfigOption{Key: "bin_path", Value: vasmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)})
	}

	if hysmi, ok := kwargs[TELEGRAF_INPUT_HYSMI]; ok {
		hysmiMap, _ := hysmi.(map[string]interface{})
		addInput(TELEGRAF_INPUT_HYSMI, telegrafutil.ConfigOption{Key: "bin_path", Value: hysmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)})
	}

	if ixsmi, ok := kwargs[TELEGRAF_INPUT_IXSMI]; ok {
		ixsmiMap, _ := ixsmi.(map[string]interface{})
		options := []telegrafutil.ConfigOption{{Key: "bin_path", Value: ixsmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)}}
		if libPath, _ := ixsmiMap[TELEGRAF_INPUT_CONF_LIB_PATH].(string); libPath != "" {
			options = append(options, telegrafutil.ConfigOption{Key: "lib_path", Value: libPath})
		}
		addInput(TELEGRAF_INPUT_IXSMI, options...)
	}

	if ppusmi, ok := kwargs[TELEGRAF_INPUT_PPUSMI]; ok {
		ppusmiMap, _ := ppusmi.(map[string]interface{})
		options := []telegrafutil.ConfigOption{{Key: "bin_path", Value: ppusmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)}}
		if libPath, _ := ppusmiMap[TELEGRAF_INPUT_CONF_LIB_PATH].(string); libPath != "" {
			options = append(options, telegrafutil.ConfigOption{Key: "lib_path", Value: libPath})
		}
		addInput(TELEGRAF_INPUT_PPUSMI, options...)
	}

	if xpusmi, ok := kwargs[TELEGRAF_INPUT_XPUSMI]; ok {
		xpusmiMap, _ := xpusmi.(map[string]interface{})
		options := []telegrafutil.ConfigOption{{Key: "bin_path", Value: xpusmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)}}
		if libPath, _ := xpusmiMap[TELEGRAF_INPUT_CONF_LIB_PATH].(string); libPath != "" {
			options = append(options, telegrafutil.ConfigOption{Key: "lib_path", Value: libPath})
		}
		addInput(TELEGRAF_INPUT_XPUSMI, options...)
	}

	if mtgmi, ok := kwargs[TELEGRAF_INPUT_MTHREADSGMI]; ok {
		mtgmiMap, _ := mtgmi.(map[string]interface{})
		addInput(TELEGRAF_INPUT_MTHREADSGMI, telegrafutil.ConfigOption{Key: "bin_path", Value: mtgmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)})
	}

	if _, ok := kwargs[TELEGRAF_INPUT_NVIDIASMI]; ok {
		addInput("nvidia_smi", telegrafutil.ConfigOption{Key: "timeout", Value: "30s"})
	}

	if npusmi, ok := kwargs[TELEGRAF_INPUT_NPUSMI]; ok {
		npusmiMap, _ := npusmi.(map[string]interface{})
		addInput("npu_smi", telegrafutil.ConfigOption{Key: "bin_path", Value: npusmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)})
	}

	if rocmsmi, ok := kwargs[TELEGRAF_INPUT_ROCM_SMI]; ok {
		rocmsmiMap, _ := rocmsmi.(map[string]interface{})
		addInput("amd_rocm_smi", telegrafutil.ConfigOption{Key: "bin_path", Value: rocmsmiMap[TELEGRAF_INPUT_CONF_BIN_PATH].(string)})
	}

	// 检查 IPMI 设备文件是否存在
	if hasIPMIDevice() {
		addInput("ipmi_sensor",
			telegrafutil.ConfigOption{Key: "metric_version", Value: 2},
			telegrafutil.ConfigOption{Key: "sensors", Value: []string{"sdr"}})
	}

	conf, err := telegrafutil.RenderConfig(config)
	if err != nil {
		log.Errorf("render telegraf config: %v", err)
		return ""
	}
	return conf
}

func (s *STelegraf) GetConfigFile() string {
	dir := getTelegrafConfigDir()
	procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", dir).Run()
	return filepath.Join(dir, "telegraf.conf")
}

func getTelegrafConfigDir() string {
	defaultTelegrafConfigDir := "/etc/telegraf"
	telegrafConfigDir := os.Getenv("HOST_TELEGRAF_CONFIG_DIR")
	if telegrafConfigDir == "" {
		telegrafConfigDir = defaultTelegrafConfigDir
	}
	return telegrafConfigDir
}

func GetTelegrafConfDDir() string {
	dir := getTelegrafConfigDir()
	return filepath.Join(dir, "telegraf.d")
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
	if err := s.reloadTelegrafByHTTP(); err != nil {
		errs = append(errs, errors.Wrap(err, "reloadTelegrafByHTTP"))
		return errors.NewAggregate(errs)
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
	telegrafReoladUrl := "http://localhost:8087/reload"
	log.Infof("Reloading telegraf by %q ...", telegrafReoladUrl)
	if _, _, err := httputils.JSONRequest(
		httputils.GetDefaultClient(), context.Background(),
		"POST", telegrafReoladUrl, nil, nil, false,
	); err != nil {
		return errors.Wrap(err, "reload telegraf by http reload api")
	}
	return nil
}

// hasIPMIDevice 检查 IPMI 设备文件是否存在
func hasIPMIDevice() bool {
	ipmiDevices := []string{
		"/dev/ipmi0",
		"/dev/ipmi/0",
		"/dev/ipmidev/0",
	}
	for _, device := range ipmiDevices {
		if _, err := os.Stat(device); err == nil {
			return true
		}
	}
	return false
}
