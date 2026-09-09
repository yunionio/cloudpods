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
	"os"
	"path/filepath"
	"testing"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
)

func TestHostLocalGetRuntimeMountHostPath(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "f")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	h := hostLocal{}
	got, err := h.GetRuntimeMountHostPath(nil, "", &hostapi.ContainerVolumeMount{
		HostPath: &apis.ContainerVolumeMountHostPath{
			Type: apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
			Path: filePath,
		},
	})
	if err != nil {
		t.Fatalf("existing file: %v", err)
	}
	if got != filePath {
		t.Fatalf("got %q", got)
	}

	if _, err := h.GetRuntimeMountHostPath(nil, "", &hostapi.ContainerVolumeMount{
		HostPath: &apis.ContainerVolumeMountHostPath{
			Type: apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
			Path: "/",
		},
	}); err == nil {
		t.Fatal("expected / to fail")
	}

	if _, err := h.GetRuntimeMountHostPath(nil, "", &hostapi.ContainerVolumeMount{
		HostPath: &apis.ContainerVolumeMountHostPath{
			Type: apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
			Path: filepath.Join(dir, "missing"),
		},
	}); err == nil {
		t.Fatal("expected missing file without auto_create to fail")
	}
}
