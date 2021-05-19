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

package options

import common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"

type AnsibleServerOptions struct {
	common_options.CommonOptions
	common_options.DBOptions
	KeepTmpdir          bool `help:"Whether to save the tmp directory" json:"keep_tmpdir"`
	PlaybookWorkerCount int  `help:"count of worker to run playbook" default:"5" json:"playbook_worker_count"`
	RolePublic          bool `help:"Download and use the role of the public directory"`
	Timeout             int  `help:"Ansible Override the connection timeout in seconds" default:"10"`
}

var (
	Options AnsibleServerOptions
)
