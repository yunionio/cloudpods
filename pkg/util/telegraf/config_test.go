package telegraf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfilesKeepApprovedDifferences(t *testing.T) {
	profiles := Profiles()
	contains := func(profile, name string) bool {
		for _, p := range profiles[profile] {
			if p.Plugin == name {
				return true
			}
		}
		return false
	}
	for profile, name := range map[string]string{
		"guest_linux":   "nvidia_smi",
		"guest_windows": "swap",
		"baremetal":     "kernel",
	} {
		if !contains(profile, name) {
			t.Errorf("%s profile missing %s", profile, name)
		}
	}
	for _, name := range []string{"kernel", "kernel_vmstat", "brsmi"} {
		if !contains("baremetal", name) {
			t.Errorf("baremetal profile missing %s", name)
		}
	}
	for _, name := range []string{"sensors", "smart"} {
		if contains("baremetal", name) {
			t.Errorf("optional plugin %s must be selected after probing", name)
		}
	}
}

func TestRenderAndExportAnsibleVars(t *testing.T) {
	conf, err := Render("guest_windows", []map[string]string{{"tag_name": "host", "tag_value": "node-1"}}, "http://127.0.0.1:8086", `C:\\Program Files\\Telegraf\\telegraf.log`)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"host = \"node-1\"", "[[inputs.swap]]", "[[inputs.nvidia_smi]]", "logfile_rotation_max_size = \"10MB\""} {
		if !strings.Contains(conf, expected) {
			t.Errorf("rendered config missing %q", expected)
		}
	}

	path := filepath.Join(t.TempDir(), "vars.json")
	if err := WriteAnsibleVars(path); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var vars map[string]interface{}
	if err := json.Unmarshal(b, &vars); err != nil {
		t.Fatal(err)
	}
	if vars["cloudpods_telegraf_profiles"] == nil {
		t.Fatal("shared profiles not exported")
	}
	if vars["telegraf_agent_logfile"] != "" {
		t.Fatalf("ansibleserver logfile should be empty, got %v", vars["telegraf_agent_logfile"])
	}
}
