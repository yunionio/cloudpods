// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or authorized to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kvmpart

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetLocalPathDoesNotFollowSymlink(t *testing.T) {
	mount := t.TempDir()
	fs := NewLocalGuestFS(mount)
	if err := os.MkdirAll(filepath.Join(mount, "etc"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/", filepath.Join(mount, "attack")); err != nil {
		t.Fatal(err)
	}

	if p := fs.GetLocalPath("/attack/etc", false); p != "" {
		t.Fatalf("expected empty path, got %q", p)
	}
	if p := fs.GetLocalPath("/etc", false); p != filepath.Join(mount, "etc") {
		t.Fatalf("got %q", p)
	}
}

func TestFilePutContentsDoesNotFollowSymlink(t *testing.T) {
	mount := t.TempDir()
	fs := NewLocalGuestFS(mount)
	if err := os.MkdirAll(filepath.Join(mount, "etc"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/", filepath.Join(mount, "attack")); err != nil {
		t.Fatal(err)
	}

	if err := fs.FilePutContents("/attack/etc/pwn", "x", false, false); err == nil {
		t.Fatal("expected write through symlink prefix to fail")
	}

	marker := filepath.Join(t.TempDir(), "host-target")
	if err := os.WriteFile(marker, []byte("orig"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(marker, filepath.Join(mount, "etc", "hosts")); err != nil {
		t.Fatal(err)
	}
	if err := fs.FilePutContents("/etc/hosts", "pwn", false, false); err == nil {
		t.Fatal("expected write through symlink file to fail")
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "orig" {
		t.Fatalf("host file changed: %q", got)
	}

	if err := fs.FilePutContents("/etc/hostname", "vm1", false, false); err != nil {
		t.Fatalf("regular write: %v", err)
	}
	got, err = os.ReadFile(filepath.Join(mount, "etc", "hostname"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "vm1" {
		t.Fatalf("got %q", got)
	}
}

func TestMkdirDoesNotFollowSymlink(t *testing.T) {
	mount := t.TempDir()
	fs := NewLocalGuestFS(mount)
	if err := os.Symlink("/", filepath.Join(mount, "attack")); err != nil {
		t.Fatal(err)
	}
	if err := fs.Mkdir("/attack/etc/cron.d", 0755, false); err == nil {
		t.Fatal("expected mkdir through symlink to fail")
	}
	if err := fs.Mkdir("/etc/cron.d", 0755, false); err != nil {
		t.Fatalf("regular mkdir: %v", err)
	}
	if st, err := os.Stat(filepath.Join(mount, "etc", "cron.d")); err != nil || !st.IsDir() {
		t.Fatalf("cron.d not created: %v", err)
	}
}
