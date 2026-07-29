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
// See the License for the permissions and limitations under the License.

package container_device

import (
	"os"
	"path"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func hygonUseRemoteFS() bool {
	return options.HostOptions.EnableRemoteExecutor
}

func hygonPathExists(path string) bool {
	if hygonUseRemoteFS() {
		return procutils.RemotePathExists(path)
	}
	return fileutils2.Exists(path)
}

func hygonReadDir(dirname string) ([]os.FileInfo, error) {
	if hygonUseRemoteFS() {
		return procutils.RemoteReadDir(dirname)
	}
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	files := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, info)
	}
	return files, nil
}

func hygonReadlink(path string) (string, error) {
	if hygonUseRemoteFS() {
		return procutils.RemoteReadlink(path)
	}
	return os.Readlink(path)
}

func hygonRenderPathFromLink(linkPath string) string {
	return hygonRenderPathFromLinkWithRemote(linkPath, hygonUseRemoteFS())
}

func hygonRenderPathFromLinkWithRemote(linkPath string, remote bool) string {
	if remote {
		return linkPath
	}
	return path.Join("/dev/dri", path.Base(linkPath))
}
