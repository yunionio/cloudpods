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

package qemuimg

import "testing"

func TestGetQemuImgVersion(t *testing.T) {
	verStr := `qemu-img version 1.5.3, Copyright (c) 2004-2008 Fabrice Bellard`
	matches := qemuImgVersionRegexp.FindStringSubmatch(verStr)
	t.Logf("%s", matches[1])
}

// TODO: rewrite TestQcow2
/*
func TestQcow2(t *testing.T) {
	img, err := NewQemuImage("test")
	if err != nil {
		t.Fatal(err)
	}
	err = img.CreateQcow2(1000, true, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Delete()
	if err != nil {
		t.Fatal(err)
	}
	err = img.CreateQcow2(1000, false, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Qcow2(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Qcow2(false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Resize(2048)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Qcow2(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Expand()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Delete()
	if err != nil {
		t.Fatal(err)
	}
	err = img.CreateQcow2(1000, true, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	img2, err := img.CloneQcow2("test2", true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img2, img2.IsSparse())
	err = img2.Delete()
	if err != nil {
		t.Fatal(err)
	}
	err = img.Convert2Qcow2(false)
	if err != nil {
		t.Fatal(err)
	}
	img4, err := NewQemuImage("test_top")
	if err != nil {
		t.Fatal(err)
	}
	err = img4.CreateQcow2(0, true, img.Path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	t.Logf("%s %v", img4, img4.IsSparse())
	err = img.Convert2Qcow2(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	t.Logf("%s %v", img4, img4.IsSparse())
	img4.Delete()
	img.Delete()
}

func TestVhd(t *testing.T) {
	img, err := NewQemuImage("test")
	if err != nil {
		t.Fatal(err)
	}
	err = img.CreateVhd(1024)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Delete()
	if err != nil {
		t.Fatal(err)
	}
	err = img.CreateVhd(1024)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Vhd()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Vhd()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	img.Delete()
}

func TestVmdk(t *testing.T) {
	img, err := NewQemuImage("test")
	if err != nil {
		t.Fatal(err)
	}
	err = img.CreateVmdk(1024, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Delete()
	if err != nil {
		t.Fatal(err)
	}
	err = img.CreateVmdk(1024, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Vmdk(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Vmdk(false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %v", img, img.IsSparse())
	img.Delete()
}*/

func TestParseBackingFile(t *testing.T) {
	in := `json:{"driver":"qcow2","file":{"driver":"file","filename":"/opt/cloud/workspace/disks/snapshots/72a2383d-e980-486f-816c-6c562e1757f3_snap/f39f225a-921f-492e-8fb6-0a4167d6ed91"}}`
	want := "/opt/cloud/workspace/disks/snapshots/72a2383d-e980-486f-816c-6c562e1757f3_snap/f39f225a-921f-492e-8fb6-0a4167d6ed91"
	path, err := parseBackingFilepath(in)
	if err != nil {
		t.Errorf("parseBackingFilepath: %s", err)
	} else if path != want {
		t.Errorf("want: %s got: %s", want, path)
	}
}
