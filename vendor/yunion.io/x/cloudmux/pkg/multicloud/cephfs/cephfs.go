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

package cephfs

type SCephFS struct {
	Id     string
	Mdsmap struct {
		Epoch                     int
		Flags                     int
		EverAllowedFeatures       int
		ExplicitlyAllowedFeatures int
		SessionTimeout            int
		SessionAutoclose          int
		MaxFileSize               int64
		Created                   string
		Enabled                   bool
		FsName                    string
	}
}

func (cli *SCephFSClient) GetCephFSs() ([]SCephFS, error) {
	resp, err := cli.list("cephfs", nil)
	if err != nil {
		return nil, err
	}
	ret := []SCephFS{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
