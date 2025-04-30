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
	"yunion.io/x/onecloud/pkg/apis"
)

type HostFileType string

const (
	PlainFile       HostFileType = "plain"
	ScriptFile      HostFileType = "script"
	TelegrafConf    HostFileType = "telegraf"
	ApparmorProfile HostFileType = "apparmor"
)

type HostFileCreateInput struct {
	apis.InfrasResourceBaseCreateInput

	Path    string       `json:"path" help:"Path of the host file"`
	Content string       `json:"content" help:"Content of the host file"`
	Type    HostFileType `json:"type" help:"Type of the host file"`
}

type HostFileUpdateInput struct {
	apis.InfrasResourceBaseUpdateInput

	Path    string `json:"path"`
	Content string `json:"content"`
}

type HostFileListInput struct {
	apis.InfrasResourceBaseListInput

	Type []string `json:"type"`
	Path string   `json:"path"`
}

type HostSetHostFilesInput struct {
	HostFiles []string `json:"host_files"`
}

type HostFileDetails struct {
	apis.InfrasResourceBaseDetails

	Hosts []string `json:"hosts"`

	SHostFile
}
