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
	"text/template"
)

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
	baremetal = append(baremetal, plugins("kernel", "kernel_vmstat", "nvidia_smi", "npu_smi", "ixsmi", "ppusmi", "hysmi", "xpusmi", "brsmi")...)
	return map[string][]Plugin{
		"guest_linux":   linux,
		"guest_windows": windows,
		"baremetal":     baremetal,
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

const configTemplate = `### MANAGED BY ansible-telegraf ANSIBLE ROLE ###

[global_tags]
{{ range $value := .Tags }}
    {{ $value.Name }} = {{ printf "%q" $value.Value }}
{{- end }}

# Configuration for telegraf agent
[agent]
    interval = "{{ .Interval }}s"
    debug = false
    hostname = ""
    round_interval = true
    flush_interval = "{{ .FlushInterval }}s"
    flush_jitter = "{{ .FlushJitter }}s"
    collection_jitter = "{{ .CollectionJitter }}s"
    metric_batch_size = {{ .MetricBatchSize }}
    metric_buffer_limit = {{ .MetricBufferLimit }}
    quiet = false
    logfile = {{ printf "%q" .Logfile }}
    logfile_rotation_max_size = "{{ .LogRotationMaxSize }}"
    logfile_rotation_max_archives = {{ .LogRotationMaxArchives }}
    omit_hostname = true

###############################################################################
#                                  OUTPUTS                                    #
###############################################################################

[[outputs.{{ .OutputType }}]]
    urls = [{{ printf "%q" .URL }}]
    database = "{{ .OutputDatabase }}"
    insecure_skip_verify = true

###############################################################################
#                                  INPUTS                                     #
###############################################################################
{{ range .Plugins }}
[[inputs.{{ .Plugin }}]]
{{ range .Config }}    {{ . }}
{{ end }}{{ end }}`

type tag struct{ Name, Value string }
type renderData struct {
	Tags                                                       []tag
	Logfile, URL                                               string
	Plugins                                                    []Plugin
	Interval, FlushInterval, FlushJitter, CollectionJitter     int
	MetricBatchSize, MetricBufferLimit, LogRotationMaxArchives int
	LogRotationMaxSize                                         string
	OutputType, OutputDatabase                                 string
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
	d := renderData{
		Logfile: logfile, URL: url, Plugins: selected,
		Interval: agentIntervalSeconds, FlushInterval: agentFlushIntervalSeconds,
		FlushJitter: agentFlushJitterSeconds, CollectionJitter: agentCollectionJitterSeconds,
		MetricBatchSize: agentMetricBatchSize, MetricBufferLimit: agentMetricBufferLimit,
		LogRotationMaxSize: agentLogRotationMaxSize, LogRotationMaxArchives: agentLogRotationMaxArchives,
		OutputType: outputType, OutputDatabase: outputDatabase,
	}
	for _, values := range tags {
		d.Tags = append(d.Tags, tag{Name: values["tag_name"], Value: values["tag_value"]})
	}
	t, err := template.New("telegraf").Parse(configTemplate)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, d); err != nil {
		return "", err
	}
	return out.String(), nil
}

// RegionWindowsLogfile preserves the Windows path used by region-side delivery.
func RegionWindowsLogfile() string { return regionWindowsLogfile }
