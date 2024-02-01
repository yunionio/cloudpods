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

package compute

import (
	"reflect"
	"testing"

	"yunion.io/x/onecloud/pkg/apis"
)

func Test_parseContainerVolumeMount(t *testing.T) {
	index0 := 0
	tests := []struct {
		args    string
		want    *apis.ContainerVolumeMount
		wantErr bool
	}{
		{
			args: "readonly=true,mount_path=/data,disk_index=0",
			want: &apis.ContainerVolumeMount{
				ReadOnly:  true,
				MountPath: "/data",
				Disk:      &apis.ContainerVolumeMountDisk{Index: &index0},
				Type:      apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			},
		},
		{
			args: "readonly=true,mount_path=/data,disk_index=0,disk_subdir=data",
			want: &apis.ContainerVolumeMount{
				ReadOnly:  true,
				MountPath: "/data",
				Disk:      &apis.ContainerVolumeMountDisk{Index: &index0, SubDirectory: "data"},
				Type:      apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			},
		},
		{
			args: "readonly=true,mount_path=/storage_size,disk_index=0,disk_ssf=storage_size",
			want: &apis.ContainerVolumeMount{
				ReadOnly:  true,
				MountPath: "/storage_size",
				Disk: &apis.ContainerVolumeMountDisk{
					Index:           &index0,
					StorageSizeFile: "storage_size",
				},
				Type: apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			},
		},
		{
			args:    "disk_id=abc,mount_path=/data",
			wantErr: false,
			want: &apis.ContainerVolumeMount{
				Type: apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
				Disk: &apis.ContainerVolumeMountDisk{
					Id: "abc",
				},
				MountPath: "/data",
			},
		},
		{
			args: "host_path=/hostpath/abc,mount_path=/data",
			want: &apis.ContainerVolumeMount{
				Type:      apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
				HostPath:  &apis.ContainerVolumeMountHostPath{Path: "/hostpath/abc"},
				MountPath: "/data",
			},
		},
		{
			args: "read_only=True,mount_path=/test",
			want: &apis.ContainerVolumeMount{
				ReadOnly:  true,
				MountPath: "/test",
			},
		},
		{
			args:    "vm1,read_only=True,mount_path=/test",
			want:    nil,
			wantErr: true,
		},
		{
			args:    "read_only=True,mount_path=/test,disk_index=one",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			got, err := parseContainerVolumeMount(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseContainerVolumeMount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseContainerVolumeMount() got = %v, want %v", got, tt.want)
			}
		})
	}
}
