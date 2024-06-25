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
	"os"
	"runtime"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type sFileStat struct {
	FileSize  int64     `json:"file_size"`
	FileType  string    `json:"file_type"`
	FileName  string    `json:"file_name"`
	LastModAt time.Time `json:"last_mod_at"`
}

func (s *sFileStat) Name() string {
	return s.FileName
}

func (s *sFileStat) Size() int64 {
	return s.FileSize
}

func (s *sFileStat) Mode() os.FileMode {
	if s.IsDir() {
		return os.ModeDir
	}
	return os.FileMode(0)
}

func (s *sFileStat) ModTime() time.Time {
	return s.LastModAt
}

func (s *sFileStat) IsDir() bool {
	return s.FileType == "directory"
}

func (s *sFileStat) Sys() interface{} {
	return nil
}

func RemoteStat(filename string) (os.FileInfo, error) {
	args := []string{}
	switch runtime.GOOS {
	case "darwin":
		args = []string{"-f", `{"file_size":%z,"file_name":"%N","file_type":"%T"}`, filename}
	default:
		args = []string{"-c", `{"file_size":%s,"file_name":"%n","file_type":"%F"}`, filename}
	}
	output, err := NewRemoteCommandAsFarAsPossible("stat", args...).Output()
	if err != nil {
		if strings.Contains(strings.ToLower(string(output)), "no such file or directory") {
			return nil, os.ErrNotExist
		}
		return nil, errors.Wrapf(err, "NewRemoteCommandAsFarAsPossible with stat %v: %s", args, output)
	}
	json, err := jsonutils.Parse(output)
	if err != nil {
		return nil, errors.Error(output)
	}
	fs := &sFileStat{}
	err = json.Unmarshal(fs)
	if err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	return fs, nil
}
