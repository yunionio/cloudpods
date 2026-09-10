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

package fileutils2

import (
	"testing"
)

func TestCleanHostBindPath(t *testing.T) {
	got, err := CleanHostBindPath("/data/models/Qwen")
	if err != nil {
		t.Fatalf("valid path: %v", err)
	}
	if got != "/data/models/Qwen" {
		t.Fatalf("got %q", got)
	}
	got, err = CleanHostBindPath("/data/models/../models/Qwen")
	if err != nil {
		t.Fatalf("cleaned path: %v", err)
	}
	if got != "/data/models/Qwen" {
		t.Fatalf("got %q", got)
	}
	for _, p := range []string{"", "relative", "/", "/.", "/tmp/../.."} {
		if _, err := CleanHostBindPath(p); err == nil {
			t.Fatalf("expected error for %q", p)
		}
	}
}

func TestIsRestrictedHostBindPath(t *testing.T) {
	for _, p := range []string{"/etc", "/etc/shadow", "/root/.ssh", "/proc/1", "/var/lib/mysql", "/dev/sda"} {
		if !IsRestrictedHostBindPath(p) {
			t.Fatalf("%q should be restricted", p)
		}
	}
	for _, p := range []string{"/data/models", "/tmp/foo", "/opt/models", "/mnt/disk", "/home/admin", "/opt/cloud/workspace"} {
		if IsRestrictedHostBindPath(p) {
			t.Fatalf("%q should not be restricted", p)
		}
	}
}

func TestParseUnixFileMode(t *testing.T) {
	mode, canon, err := ParseUnixFileMode("755")
	if err != nil {
		t.Fatalf("755: %v", err)
	}
	if uint32(mode) != 0o755 || canon != "755" {
		t.Fatalf("mode=%o canon=%s", mode, canon)
	}
	_, canon, err = ParseUnixFileMode("0755")
	if err != nil {
		t.Fatalf("0755: %v", err)
	}
	if canon != "755" {
		t.Fatalf("canon=%s", canon)
	}
	for _, p := range []string{"", "755; id", "rwx", "999", "12", "0755 extra"} {
		if _, _, err := ParseUnixFileMode(p); err == nil {
			t.Fatalf("expected error for %q", p)
		}
	}
}

func TestCleanRelSubpath(t *testing.T) {
	got, err := CleanRelSubpath("foo/bar")
	if err != nil {
		t.Fatalf("valid: %v", err)
	}
	if got != "foo/bar" {
		t.Fatalf("got %q", got)
	}
	got, err = CleanRelSubpath("foo/./bar")
	if err != nil {
		t.Fatalf("dot: %v", err)
	}
	if got != "foo/bar" {
		t.Fatalf("got %q", got)
	}
	got, err = CleanRelSubpath(".id_rsa")
	if err != nil {
		t.Fatalf("dotfile: %v", err)
	}
	if got != ".id_rsa" {
		t.Fatalf("got %q", got)
	}
	for _, p := range []string{"", "/abs", "../etc", "foo/../../etc", "..", ".", "a\x00b"} {
		if _, err := CleanRelSubpath(p); err == nil {
			t.Fatalf("expected error for %q", p)
		}
	}
}

func TestJoinInside(t *testing.T) {
	got, err := JoinInside("/mnt/disk", "sub/dir")
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if got != "/mnt/disk/sub/dir" {
		t.Fatalf("got %q", got)
	}
	if _, err := JoinInside("/mnt/disk", "../etc"); err == nil {
		t.Fatal("expected escape to fail")
	}
	if _, err := JoinInside("/mnt/disk", "/etc"); err == nil {
		t.Fatal("expected absolute to fail")
	}
	base := t.TempDir()
	got, err = JoinInside(base, "a/b.txt")
	if err != nil {
		t.Fatalf("temp base: %v", err)
	}
	if !IsPathInside(base, got) {
		t.Fatalf("escaped: %q", got)
	}
	got, err = JoinInsideAll("/mnt/disk", "sub", "dir")
	if err != nil {
		t.Fatalf("join all: %v", err)
	}
	if got != "/mnt/disk/sub/dir" {
		t.Fatalf("got %q", got)
	}
	if _, err := JoinInsideAll("/mnt/disk", "sub", "../.."); err == nil {
		t.Fatal("expected join all escape to fail")
	}
}
