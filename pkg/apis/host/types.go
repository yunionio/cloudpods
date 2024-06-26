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

import (
	"github.com/jaypipes/ghw/pkg/cpu"
	"github.com/jaypipes/ghw/pkg/topology"
)

type ServerCloneDiskFromStorageResponse struct {
	TargetAccessPath string `json:"target_access_path"`
	TargetFormat     string `json:"target_format"`
}

type HostNodeHugepageNr struct {
	NodeId     int `json:"node_id"`
	HugepageNr int `json:"hugepage_nr"`
}

type HostTopology struct {
	*topology.Info
}

type HostCPUInfo struct {
	*cpu.Info
}

type GuestSetPasswordRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Crypted  bool   `json:"crypted"`
}
