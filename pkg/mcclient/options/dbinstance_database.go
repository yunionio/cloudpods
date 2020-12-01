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

import "yunion.io/x/jsonutils"

type DBInstanceDatabaseListOptions struct {
	BaseListOptions
	DBInstance string `help:"ID or Name of DBInstance" json:"dbinstance"`
}

func (opts *DBInstanceDatabaseListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type DBInstanceDatabaseIdOptions struct {
	ID string `help:"ID of DBInstancedatabase"`
}

func (opts *DBInstanceDatabaseIdOptions) GetId() string {
	return opts.ID
}

func (opts *DBInstanceDatabaseIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DBInstanceDatabaseCreateOptions struct {
	NAME         string
	DBINSTANCE   string `help:"ID or Name of DBInstance" json:"dbinstance"`
	CharacterSet string `help:"CharacterSet for database"`
}

func (opts *DBInstanceDatabaseCreateOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}
