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

package host

import "yunion.io/x/onecloud/pkg/apis"

type ContainerSpec struct {
	apis.ContainerSpec
	Devices []*ContainerDevice `json:"devices"`
}

type ContainerDevice struct {
	Type           apis.ContainerDeviceType `json:"type"`
	ContainerPath  string                   `json:"container_path"`
	Permissions    string                   `json:"permissions"`
	IsolatedDevice *ContainerIsolatedDevice `json:"isolated_device"`
	Host           *ContainerHostDevice     `json:"host"`
	Disk           *ContainerDiskDevice     `json:"disk"`
}

type ContainerIsolatedDevice struct {
	Id         string `json:"id"`
	Addr       string `json:"addr"`
	Path       string `json:"path"`
	DeviceType string `json:"device_type"`
}

type ContainerHostDevice struct {
	// Path of the device on the host.
	HostPath string `json:"host_path"`
}

type ContainerDiskDevice struct {
	Id string `json:"id"`
}

type ContainerCreateInput struct {
	Name    string         `json:"name"`
	GuestId string         `json:"guest_id"`
	Spec    *ContainerSpec `json:"spec"`
}

type ContainerPullImageAuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	ServerAddress string `json:"server_address,omitempty"`
	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `json:"identity_token,omitempty"`
	// RegistryToken is a bearer token to be sent to a registry
	RegistryToken string `json:"registry_token,omitempty"`
}

type ContainerPullImageInput struct {
	Image      string                        `json:"image"`
	PullPolicy apis.ImagePullPolicy          `json:"pull_policy"`
	Auth       *ContainerPullImageAuthConfig `json:"auth"`
}
