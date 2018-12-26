package qemuimg

import "testing"

func TestGetQemuImgVersion(t *testing.T) {
	verStr := `qemu-img version 1.5.3, Copyright (c) 2004-2008 Fabrice Bellard`
	matches := qemuImgVersionRegexp.FindStringSubmatch(verStr)
	t.Logf("%s", matches[1])
}

func TestQcow2(t *testing.T) {
	img, err := NewQemuImage("test")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = img.CreateQcow2(1000, true, "")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Delete()
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = img.CreateQcow2(1000, false, "")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Qcow2(true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Qcow2(false)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Resize(2048)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Qcow2(true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Expand()
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Delete()
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = img.CreateQcow2(1000, true, "")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	img2, err := img.CloneQcow2("test2", true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img2, img2.IsSparse())
	err = img2.Delete()
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = img.Convert2Qcow2(false)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	img4, err := NewQemuImage("test_top")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = img4.CreateQcow2(0, true, img.Path)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	t.Logf("%s %v", img4, img4.IsSparse())
	err = img.Convert2Qcow2(true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	t.Logf("%s %v", img4, img4.IsSparse())
	img4.Delete()
	img.Delete()
}

func TestVmdk(t *testing.T) {
	img, err := NewQemuImage("test")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = img.CreateVmdk(1024, true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Delete()
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = img.CreateVmdk(1024, false)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Vmdk(true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	err = img.Convert2Vmdk(false)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Logf("%s %v", img, img.IsSparse())
	img.Delete()
}