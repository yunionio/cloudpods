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

package device

import (
	"context"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestHostDeviceValidateCreateData(t *testing.T) {
	ctx := context.Background()
	drv := newHostDevice()
	tenant := &mcclient.SSimpleToken{User: "u", Project: "p", Roles: "user"}
	admin := &mcclient.SSimpleToken{User: "admin", Project: "system", Roles: "admin"}

	dev := &api.ContainerDevice{
		Type: "host",
		Host: &api.ContainerHostDevice{
			HostPath:      "/dev/sda",
			ContainerPath: "/dev/sda",
			Permissions:   "rwm",
		},
	}
	if _, err := drv.ValidateCreateData(ctx, tenant, nil, dev); err == nil {
		t.Fatal("expected tenant host device to fail")
	}
	if _, err := drv.ValidateCreateData(ctx, admin, nil, dev); err != nil {
		t.Fatalf("admin host device: %v", err)
	}

	rel := &api.ContainerDevice{
		Type: "host",
		Host: &api.ContainerHostDevice{
			HostPath:      "sda",
			ContainerPath: "/dev/sda",
			Permissions:   "rwm",
		},
	}
	if _, err := drv.ValidateCreateData(ctx, admin, nil, rel); err == nil {
		t.Fatal("expected relative host_path to fail")
	}
}
