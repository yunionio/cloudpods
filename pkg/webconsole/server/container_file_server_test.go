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

package server

import (
	"testing"
	"time"
)

func TestContainerLsCommand(t *testing.T) {
	cmd := containerLsCommand("/tmp/a'b")
	if len(cmd) != 3 || cmd[0] != "sh" || cmd[1] != "-c" {
		t.Fatalf("command = %#v", cmd)
	}
	want := `LC_ALL=C ls -la '/tmp/a'\''b'`
	if cmd[2] != want {
		t.Fatalf("script = %q, want %q", cmd[2], want)
	}
}

func TestParseContainerLs(t *testing.T) {
	out := "" +
		"total 16\r\n" +
		"drwxr-xr-x 3 root root 4096 Oct  8 17:11 .\n" +
		"drwxr-xr-x 3 root root 4096 Oct  8 17:10 ..\n" +
		"-rw-r--r-- 1 root root  123 Oct  8 17:11 file name.txt\n" +
		"drwxr-xr-x 2 root root 4096 Jan  2  2024 sub dir\n" +
		"lrwxrwxrwx 1 root root   11 Oct  8 17:11 link -> /tmp/target\n"
	files := parseContainerLs("/data", out)
	if len(files) != 3 {
		t.Fatalf("len = %d, files = %#v", len(files), files)
	}

	f := files[0]
	if f.Name != "file name.txt" || f.Path != "/data/file name.txt" || f.Size != 123 || f.IsDir || !f.IsRegular {
		t.Fatalf("file = %#v", f)
	}
	if f.Mode != "-rw-r--r--" || f.ModeNum != 0644 {
		t.Fatalf("mode = %s %o", f.Mode, f.ModeNum)
	}
	if f.ModTime.Month() != time.October || f.ModTime.Day() != 8 || f.ModTime.Hour() != 17 || f.ModTime.Minute() != 11 {
		t.Fatalf("mtime = %s", f.ModTime)
	}

	d := files[1]
	if d.Name != "sub dir" || !d.IsDir || d.IsRegular || d.Path != "/data/sub dir" || d.Size != 4096 {
		t.Fatalf("dir = %#v", d)
	}
	if d.ModTime.Year() != 2024 || d.ModTime.Month() != time.January || d.ModTime.Day() != 2 {
		t.Fatalf("dir mtime = %s", d.ModTime)
	}

	l := files[2]
	if l.Name != "link" || l.IsDir || l.IsRegular || l.LinkFile == nil || l.LinkFile.Name != "/tmp/target" || l.LinkFile.Path != "/tmp/target" {
		t.Fatalf("link = %#v", l)
	}
}
