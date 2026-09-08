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

package fileutils2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPathInside(t *testing.T) {
	if !IsPathInside("/mnt/guest", "/mnt/guest") {
		t.Fatal("self")
	}
	if !IsPathInside("/mnt/guest", "/mnt/guest/etc/hosts") {
		t.Fatal("child")
	}
	if IsPathInside("/mnt/guest", "/etc/hosts") {
		t.Fatal("outside")
	}
	if IsPathInside("/mnt/guest", "/mnt/guest/../etc") {
		t.Fatal("escape")
	}
}

func TestFilePutContentsNoFollow(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("orig"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := FilePutContentsNoFollow(link, "pwn", false); err == nil {
		t.Fatal("expected write through symlink to fail")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "orig" {
		t.Fatalf("target changed: %q", got)
	}
	regular := filepath.Join(dir, "file")
	if err := FilePutContentsNoFollow(regular, "ok", false); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(regular)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ok" {
		t.Fatalf("got %q", got)
	}
}

func TestCleanGuestDeployPath(t *testing.T) {
	got, err := CleanGuestDeployPath("/etc/hosts")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/etc/hosts" {
		t.Fatalf("got %q", got)
	}
	got, err = CleanGuestDeployPath("/etc/../root/.ssh/authorized_keys")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/root/.ssh/authorized_keys" {
		t.Fatalf("got %q", got)
	}
	for _, p := range []string{"", "relative", "/", "/."} {
		if _, err := CleanGuestDeployPath(p); err == nil {
			t.Fatalf("expected error for %q", p)
		}
	}
}
