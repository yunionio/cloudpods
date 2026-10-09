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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRdpDrivePath(t *testing.T) {
	tmp := t.TempDir()
	oldPrefix := rdpDriveRootPrefix
	rdpDriveRootPrefix = tmp
	defer func() { rdpDriveRootPrefix = oldPrefix }()

	ownerId := "test-user-jail"
	root := rdpDriveRoot(ownerId)
	defer os.RemoveAll(root)

	wantRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}

	abs, err := resolveRdpDrivePath(ownerId, "/")
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	if abs != wantRoot {
		t.Fatalf("root path = %s, want %s", abs, wantRoot)
	}

	abs, err = resolveRdpDrivePath(ownerId, "/foo/bar")
	if err != nil {
		t.Fatalf("nested: %v", err)
	}
	want := filepath.Join(wantRoot, "foo", "bar")
	if abs != want {
		t.Fatalf("nested = %s, want %s", abs, want)
	}
	if !strings.HasPrefix(abs, wantRoot+string(os.PathSeparator)) {
		t.Fatalf("nested not under root: %s", abs)
	}

	// path.Clean("/../../etc/passwd") => "/etc/passwd" then join under root → still jailed
	abs, err = resolveRdpDrivePath(ownerId, "/../../etc/passwd")
	if err != nil {
		t.Fatalf("traversal should stay jailed: %v", err)
	}
	if abs != filepath.Join(wantRoot, "etc", "passwd") {
		t.Fatalf("traversal resolved unexpectedly: %s", abs)
	}
	if logicalRdpPath(ownerId, abs) != "/etc/passwd" {
		t.Fatalf("logical path = %s", logicalRdpPath(ownerId, abs))
	}
}
