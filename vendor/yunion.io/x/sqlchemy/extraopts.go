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

package sqlchemy

type TableExtraOptions map[string]string

func (opts TableExtraOptions) Get(key string) string {
	return opts[key]
}

func (opts TableExtraOptions) Set(key string, val string) TableExtraOptions {
	opts[key] = val
	return opts
}

func (opts TableExtraOptions) Contains(key string) bool {
	if _, ok := opts[key]; ok {
		return true
	}
	return false
}

func (ts *STableSpec) GetExtraOptions() TableExtraOptions {
	return ts.extraOptions
}

func (ts *STableSpec) SetExtraOptions(opts TableExtraOptions) {
	if ts.extraOptions == nil {
		ts.extraOptions = opts
		return
	}
	for k := range opts {
		ts.extraOptions[k] = opts[k]
	}
}
