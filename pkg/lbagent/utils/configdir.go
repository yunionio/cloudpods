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

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	aggrerrors "yunion.io/x/pkg/util/errors"
)

const (
	ConfigDirMode       = os.FileMode(0755)
	configDirFmt        = "20060102.150405.000"
	configDirFmtStaging = configDirFmt + ".staging"
)

var (
	configDirPat = regexp.MustCompile(`^\d{8}\.\d{6}\.\d{3}$`)
)

type ConfigDirManager struct {
	baseDir string
}

// TODO
//
//   - set active
func NewConfigDirManager(baseDir string) *ConfigDirManager {
	return &ConfigDirManager{
		baseDir: baseDir,
	}
}

type DirFunc func(string) error

// call f() inside 20180830.104050.333.lbcorpus.json.staging
// move to final name 20180830.104050.333.lbcorpus.json
func (m *ConfigDirManager) NewDir(f DirFunc) (finalDir string, err error) {
	stagingDir := m.stagingDir()
	if stagingDir == "" {
		err = fmt.Errorf("failed creating staging dir")
		return
	}
	defer func() {
		if err != nil {
			os.RemoveAll(stagingDir)
		}
	}()

	err = f(stagingDir)

	if err != nil {
		return
	}
	finalDir = DirStagingToFinal(stagingDir)
	err = os.Rename(stagingDir, finalDir)
	if err != nil {
		return
	}
	return
}

func (m *ConfigDirManager) subdirs() []string {
	dirs := []string{}
	fis, err := os.ReadDir(m.baseDir)
	if err != nil {
		return dirs
	}
	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		dirName := fi.Name()
		if !configDirPat.Match([]byte(dirName)) {
			continue
		}
		dir := filepath.Join(m.baseDir, dirName)
		dirs = append(dirs, dir)
	}
	return dirs
}

func (m *ConfigDirManager) MostRecentSubdir() string {
	dirs := m.subdirs()
	nDirs := len(dirs)
	if nDirs == 0 {
		return ""
	}
	return dirs[nDirs-1]
}

func (m *ConfigDirManager) Prune(nRetain int) error {
	if nRetain <= 0 {
		// zero means prune nothing
		return nil
	}
	dirs := m.subdirs()
	if len(dirs) > nRetain {
		// retain the most recent n
		dirs = dirs[:len(dirs)-nRetain]
		errs := []error{}
		for _, dir := range dirs {
			err := os.RemoveAll(dir)
			if os.IsNotExist(err) {
				continue
			}
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			return aggrerrors.NewAggregate(errs)
		}
	}
	return nil
}

// stagingDir creates a new staging dir and returns the full path
func (m *ConfigDirManager) stagingDir() string {
	for {
		now := time.Now()
		dn := now.Format(configDirFmtStaging)
		path := filepath.Join(m.baseDir, dn)
		_, err := os.Stat(path)
		if err != nil && os.IsNotExist(err) {
			os.MkdirAll(path, ConfigDirMode)
			return path
		}
		time.Sleep(time.Millisecond)
	}
}

func DirStagingToFinal(s string) string {
	if strings.HasSuffix(s, ".staging") {
		return s[:len(s)-8]
	}
	return s
}
