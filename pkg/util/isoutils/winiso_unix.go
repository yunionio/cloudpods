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

//go:build linux
// +build linux

package isoutils

import (
	"fmt"
)

// DetectWindowsEdition reads sources/install.wim or sources/install.esd XML metadata
// to determine Windows edition/version. Prefer .wim, fall back to .esd.
func DetectWindowsEdition(r *ISOFileReader) (*ISOInfo, error) {
	var lastErr error
	for _, path := range []string{"sources/install.wim", "sources/install.esd"} {
		if !r.FileExists(path) {
			continue
		}
		f, err := r.GetFile(path)
		if err != nil {
			lastErr = fmt.Errorf("open %s: %w", path, err)
			continue
		}
		info, err := parseWimXmlMetadata(f.NewReader())
		if err != nil {
			lastErr = fmt.Errorf("parse %s: %w", path, err)
			continue
		}
		return info, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("sources/install.wim or sources/install.esd not found")
}
