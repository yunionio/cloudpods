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

package procutils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

var (
	remoteTmpDir = "/var/run/onecloud/files"
)

func GetRemoteTempDir() string {
	return remoteTmpDir
}

func SetRemoteTempDir(dir string) {
	remoteTmpDir = dir
}

func EnsureDir(dir string) error {
	if dir != "" && dir != "." {
		mkdirCmd := NewRemoteCommandAsFarAsPossible("mkdir", "-p", dir)
		if err := mkdirCmd.Run(); err != nil {
			return errors.Wrapf(err, "mkdir -p %s", dir)
		}
	}
	return nil
}

func FilePutContents(filename string, content string) error {
	// Generate temp filename: replace / with _ and add timestamp
	// Example: /etc/abc.txt -> etc_abc.txt.1234567890
	tempName := strings.TrimPrefix(filename, "/")
	tempName = strings.ReplaceAll(tempName, "/", "_")
	timestamp := time.Now().Unix()
	tempPath := fmt.Sprintf("%s/%s.%d", GetRemoteTempDir(), tempName, timestamp)

	// Ensure tempPath dir
	if err := EnsureDir(filepath.Dir(tempPath)); err != nil {
		return errors.Wrapf(err, "EnsureDir %s", filepath.Dir(tempPath))
	}

	// Write temp file using Go native function
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return errors.Wrapf(err, "write file %s", tempPath)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(filename)
	if err := EnsureDir(targetDir); err != nil {
		// Clean up temp file
		os.Remove(tempPath)
		return errors.Wrapf(err, "EnsureDir targetDir %s", targetDir)
	}

	// Move temp file to target location
	mvCmd := NewRemoteCommandAsFarAsPossible("mv", tempPath, filename)
	if err := mvCmd.Run(); err != nil {
		// Clean up temp file
		os.Remove(tempPath)
		return errors.Wrapf(err, "mv %s %s", tempPath, filename)
	}

	return nil
}

func FileGetContents(filename string) ([]byte, error) {
	cmd := NewRemoteCommandAsFarAsPossible("cat", filename)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "StdoutPipe")
	}
	err = cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, "Run")
	}
	contentChan := make(chan []byte)
	go func() {
		defer stdout.Close()
		content, err := io.ReadAll(stdout)
		if err != nil {
			log.Errorf("ReadAll: %v", err)
		}
		contentChan <- content
	}()
	content := <-contentChan
	return content, nil
}
