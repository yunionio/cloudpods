// Copyright 2026 Yunion
// Licensed under the Apache License, Version 2.0.

// Package telegraf contains the shared monitor-agent configuration policy.
package telegraf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// ConfigOption is a typed TOML key/value. RawValue is reserved for existing
// plugin definitions that are already expressed as TOML.
type ConfigSection struct {
	Name     string
	Array    bool
	Nested   bool
	Options  []ConfigOption
	Children []ConfigSection
}

type ConfigOption struct {
	Key      string
	Value    interface{}
	RawValue bool
}

// Config contains caller-supplied, structured Telegraf tables.
type Config struct {
	GlobalTags ConfigSection
	Agent      ConfigSection
	Outputs    []ConfigSection
	Inputs     []ConfigSection
}

func tomlValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strconv.Quote(v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case []string:
		values := make([]string, len(v))
		for i := range v {
			values[i] = strconv.Quote(v[i])
		}
		return "[" + strings.Join(values, ", ") + "]"
	default:
		return fmt.Sprint(v)
	}
}

const configTemplate = "{{define \"section\"}}{{ $section := . }}{{if .Nested}}  {{end}}{{if .Array}}[[{{.Name}}]]{{else}}[{{.Name}}]{{end}}\n" +
	"{{range .Options}}{{if $section.Nested}}    {{else}}  {{end}}{{.Key}} = {{if .RawValue}}{{.Value}}{{else}}{{toml .Value}}{{end}}\n" +
	"{{end}}{{range .Children}}\n{{template \"section\" .}}{{end}}{{end}}" +
	"{{template \"section\" .GlobalTags}}\n{{template \"section\" .Agent}}" +
	"{{range .Outputs}}\n{{template \"section\" .}}{{end}}{{range .Inputs}}\n{{template \"section\" .}}{{end}}"

// RenderConfig renders caller-supplied structured tables using the common TOML template.
func RenderConfig(config Config) (string, error) {
	t, err := template.New("telegraf-config").Funcs(template.FuncMap{"toml": tomlValue}).Parse(configTemplate)
	if err != nil {
		return "", fmt.Errorf("parse telegraf config template: %w", err)
	}
	var out bytes.Buffer
	if err := t.Execute(&out, config); err != nil {
		return "", fmt.Errorf("render telegraf config: %w", err)
	}
	return out.String(), nil
}

// AnsibleVarsPath is the shared vars-file path inside the ansibleserver image.
const AnsibleVarsPath = "/opt/yunion/playbook/monitor-agent/cloudpods-telegraf-vars.json"

type Plugin struct {
	Plugin string   `json:"plugin"`
	Config []string `json:"config,omitempty"`
}

const (
	agentIntervalSeconds         = 60
	agentFlushIntervalSeconds    = 60
	agentFlushJitterSeconds      = 0
	agentCollectionJitterSeconds = 0
	agentMetricBatchSize         = 1000
	agentMetricBufferLimit       = 10000
	agentLogRotationMaxSize      = "10MB"
	agentLogRotationMaxArchives  = 1
	linuxLogfile                 = "/var/log/telegraf.log"
	regionWindowsLogfile         = "/Program Files/Telegraf/telegraf.log"
	outputType                   = "influxdb"
	outputDatabase               = "telegraf"
)

func plugins(names ...string) []Plugin {
	out := make([]Plugin, 0, len(names))
	for _, name := range names {
		out = append(out, Plugin{Plugin: name, Config: []string{`name_prefix = "agent_"`}})
	}
	return out
}

func basePlugins() []Plugin {
	return []Plugin{
		{Plugin: "cpu", Config: []string{`name_prefix = "agent_"`, "percpu = true", "totalcpu = true", "collect_cpu_time = false", "report_active = true"}},
		{Plugin: "disk", Config: []string{`name_prefix = "agent_"`, `ignore_fs = ["tmpfs", "devtmpfs", "overlay", "squashfs", "iso9660"]`}},
		{Plugin: "diskio", Config: []string{`name_prefix = "agent_"`, "skip_serial_number = false"}},
		{Plugin: "mem", Config: []string{`name_prefix = "agent_"`}},
		{Plugin: "processes", Config: []string{`name_prefix = "agent_"`}},
		{Plugin: "swap", Config: []string{`name_prefix = "agent_"`}},
		{Plugin: "system", Config: []string{`name_prefix = "agent_"`}},
		{Plugin: "net", Config: []string{`name_prefix = "agent_"`}},
		{Plugin: "netstat", Config: []string{`name_prefix = "agent_"`}},
		{Plugin: "nstat", Config: []string{`name_prefix = "agent_"`}},
		{Plugin: "internal", Config: []string{`name_prefix = "agent_"`, "collect_memstats = false"}},
	}
}

func clonePlugins(src []Plugin) []Plugin {
	dst := make([]Plugin, len(src))
	for i, p := range src {
		dst[i] = Plugin{Plugin: p.Plugin, Config: append([]string(nil), p.Config...)}
	}
	return dst
}

// Profiles returns independent plugin lists for the supported server profiles.
func Profiles() map[string][]Plugin {
	base := basePlugins()
	linux := clonePlugins(base)
	linux = append(linux, plugins("kernel", "kernel_vmstat", "nvidia_smi")...)
	windows := clonePlugins(base) // Windows intentionally includes swap.
	windows = append(windows, plugins("nvidia_smi")...)
	baremetal := clonePlugins(base)
	baremetal = append(baremetal, plugins(
		"kernel", "kernel_vmstat",
		"nvidia_smi", "amd_rocm_smi", "npu_smi", "ixsmi",
		"ppusmi", "hysmi", "xpusmi", "brsmi", "mthreadsgmi",
	)...)
	return map[string][]Plugin{
		"guest_linux":     linux,
		"guest_windows":   windows,
		"baremetal_linux": baremetal,
	}
}

// AnsibleVars is exported by ansibleserver on startup and loaded by monitor-agent.
func AnsibleVars() map[string]interface{} {
	return map[string]interface{}{
		"telegraf_agent_interval":                      agentIntervalSeconds,
		"telegraf_agent_debug":                         false,
		"telegraf_agent_hostname":                      "",
		"telegraf_agent_round_interval":                true,
		"telegraf_agent_flush_interval":                agentFlushIntervalSeconds,
		"telegraf_agent_flush_jitter":                  agentFlushJitterSeconds,
		"telegraf_agent_collection_jitter":             agentCollectionJitterSeconds,
		"telegraf_agent_metric_batch_size":             agentMetricBatchSize,
		"telegraf_agent_metric_buffer_limit":           agentMetricBufferLimit,
		"telegraf_agent_quiet":                         false,
		"telegraf_agent_omit_hostname":                 true,
		"telegraf_agent_logfile":                       "",
		"telegraf_agent_logfile_rotation_max_size":     agentLogRotationMaxSize,
		"telegraf_agent_logfile_rotation_max_archives": agentLogRotationMaxArchives,
		"telegraf_agent_output": []map[string]interface{}{{
			"type": outputType,
			"config": []string{
				`urls = ["{{ influxdb_url }}"]`,
				`database = "{{ influxdb_name }}"`,
				"insecure_skip_verify = true",
			},
		}},
		"cloudpods_telegraf_profiles": Profiles(),
	}
}

// WriteAnsibleVars writes the generated configuration atomically.
func WriteAnsibleVars(path string) error {
	b, err := json.MarshalIndent(AnsibleVars(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal telegraf ansible vars: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create telegraf vars directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0644); err != nil {
		return fmt.Errorf("write telegraf ansible vars: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("install telegraf ansible vars: %w", err)
	}
	return nil
}

// Render generates the same profile configuration for region-side deployment.
func Render(profile string, tags []map[string]string, url, logfile string) (string, error) {
	profiles := Profiles()
	selected, ok := profiles[profile]
	if !ok {
		return "", fmt.Errorf("unknown telegraf profile %q", profile)
	}
	if logfile == "" {
		logfile = linuxLogfile
	}
	config := Config{
		GlobalTags: ConfigSection{Name: "global_tags"},
		Agent: ConfigSection{Name: "agent", Options: []ConfigOption{
			{Key: "interval", Value: "60s"}, {Key: "debug", Value: false},
			{Key: "hostname", Value: ""}, {Key: "round_interval", Value: true},
			{Key: "flush_interval", Value: "60s"}, {Key: "flush_jitter", Value: "0s"},
			{Key: "collection_jitter", Value: "0s"}, {Key: "metric_batch_size", Value: 1000},
			{Key: "metric_buffer_limit", Value: 10000}, {Key: "quiet", Value: false},
			{Key: "logfile", Value: logfile}, {Key: "logfile_rotation_max_size", Value: agentLogRotationMaxSize},
			{Key: "logfile_rotation_max_archives", Value: agentLogRotationMaxArchives}, {Key: "omit_hostname", Value: true},
		}},
		Outputs: []ConfigSection{{Name: "outputs." + outputType, Array: true, Options: []ConfigOption{
			{Key: "urls", Value: []string{url}}, {Key: "database", Value: outputDatabase},
			{Key: "insecure_skip_verify", Value: true},
		}}},
	}
	for _, values := range tags {
		config.GlobalTags.Options = append(config.GlobalTags.Options, ConfigOption{Key: values["tag_name"], Value: values["tag_value"]})
	}
	for _, plugin := range selected {
		section := ConfigSection{Name: "inputs." + plugin.Plugin, Array: true}
		for _, line := range plugin.Config {
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				return "", fmt.Errorf("invalid config for input %s: %q", plugin.Plugin, line)
			}
			section.Options = append(section.Options, ConfigOption{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value), RawValue: true})
		}
		config.Inputs = append(config.Inputs, section)
	}
	return RenderConfig(config)
}

// RegionWindowsLogfile preserves the Windows path used by region-side delivery.
func RegionWindowsLogfile() string { return regionWindowsLogfile }
