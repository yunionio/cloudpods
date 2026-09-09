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

package volume_mount

import (
	"context"
	"testing"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestHostLocalValidatePodCreateData(t *testing.T) {
	ctx := context.Background()
	drv := newHostLocal()
	tenant := &mcclient.SSimpleToken{User: "u", Project: "p", Roles: "user"}
	admin := &mcclient.SSimpleToken{User: "admin", Project: "system", Roles: "admin"}

	vm := func(path string) *apis.ContainerVolumeMount {
		return &apis.ContainerVolumeMount{
			HostPath: &apis.ContainerVolumeMountHostPath{
				Type: apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
				Path: path,
			},
		}
	}

	if err := drv.ValidatePodCreateData(ctx, tenant, vm("/data/models/Qwen3-8B"), nil); err != nil {
		t.Fatalf("tenant data path: %v", err)
	}
	if err := drv.ValidatePodCreateData(ctx, tenant, vm("/"), nil); err == nil {
		t.Fatal("expected / to fail")
	}
	if err := drv.ValidatePodCreateData(ctx, tenant, vm("/etc/shadow"), nil); err == nil {
		t.Fatal("expected tenant /etc/shadow to fail")
	}
	if err := drv.ValidatePodCreateData(ctx, admin, vm("/etc/shadow"), nil); err != nil {
		t.Fatalf("admin /etc/shadow: %v", err)
	}

	badPerm := vm("/data/models/x")
	badPerm.HostPath.AutoCreate = true
	badPerm.HostPath.AutoCreateConfig = &apis.ContainerVolumeMountHostPathAutoCreateConfig{
		Permissions: "755; id",
	}
	if err := drv.ValidatePodCreateData(ctx, tenant, badPerm, nil); err == nil {
		t.Fatal("expected tenant auto_create to fail")
	}
	if err := drv.ValidatePodCreateData(ctx, admin, badPerm, nil); err == nil {
		t.Fatal("expected invalid permissions to fail")
	}

	okPerm := vm("/data/models/x")
	okPerm.HostPath.AutoCreate = true
	okPerm.HostPath.AutoCreateConfig = &apis.ContainerVolumeMountHostPathAutoCreateConfig{
		Permissions: "0755",
	}
	if err := drv.ValidatePodCreateData(ctx, admin, okPerm, nil); err != nil {
		t.Fatalf("admin auto_create: %v", err)
	}
	if okPerm.HostPath.AutoCreateConfig.Permissions != "755" {
		t.Fatalf("canon permissions %q", okPerm.HostPath.AutoCreateConfig.Permissions)
	}
}

func TestDiskRelPathsAndOverlay(t *testing.T) {
	ctx := context.Background()
	tenant := &mcclient.SSimpleToken{User: "u", Project: "p", Roles: "user"}
	d := newDisk().(*disk)
	idx := 0

	_, err := d.validateDiskCreateData(ctx, tenant, &apis.ContainerVolumeMountDisk{
		Index:        &idx,
		SubDirectory: "../../etc",
	})
	if err == nil {
		t.Fatal("expected sub_directory escape to fail")
	}

	got, err := d.validateDiskCreateData(ctx, tenant, &apis.ContainerVolumeMountDisk{
		Index:        &idx,
		SubDirectory: "foo/./bar",
	})
	if err != nil {
		t.Fatalf("valid sub_directory: %v", err)
	}
	if got.SubDirectory != "foo/bar" {
		t.Fatalf("got %q", got.SubDirectory)
	}

	ov := diskOverlayDir{}
	input := &apis.ContainerVolumeMountDiskOverlay{LowerDir: []string{"/"}}
	if err := ov.validateCommonCreateData(ctx, tenant, input); err == nil {
		t.Fatal("expected overlay / to fail")
	}
	input = &apis.ContainerVolumeMountDiskOverlay{LowerDir: []string{"/etc/shadow"}}
	if err := ov.validateCommonCreateData(ctx, tenant, input); err == nil {
		t.Fatal("expected overlay /etc/shadow to fail")
	}
	input = &apis.ContainerVolumeMountDiskOverlay{LowerDir: []string{"/data/models"}}
	if err := ov.validateCommonCreateData(ctx, tenant, input); err != nil {
		t.Fatalf("overlay data path: %v", err)
	}

	pov := newDiskPostOverlayHostPath()
	if err := pov.validateData(ctx, tenant, &apis.ContainerVolumeMountDiskPostOverlay{
		HostLowerDir:       []string{"/etc/shadow"},
		ContainerTargetDir: "/models",
	}); err == nil {
		t.Fatal("expected host_lower_dir /etc/shadow to fail")
	}
	if err := pov.validateData(ctx, tenant, &apis.ContainerVolumeMountDiskPostOverlay{
		HostLowerDir:       []string{"/data/models"},
		ContainerTargetDir: "/models",
	}); err != nil {
		t.Fatalf("host_lower_dir data path: %v", err)
	}
}
