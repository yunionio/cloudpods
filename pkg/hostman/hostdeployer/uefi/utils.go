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

package uefi

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

func DumpOvmfVarsToJson(ovmfVarsPath string) (string, error) {
	// Create temporary file for JSON output
	jsonFile, err := ioutil.TempFile("", "ovmf-vars-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	jsonPath := jsonFile.Name()
	jsonFile.Close()

	output, err := procutils.NewCommand("virt-fw-vars", "-i", ovmfVarsPath, "--output-json", jsonPath).Output()
	if err != nil {
		os.Remove(jsonPath)
		return "", errors.Wrapf(err, "virt-fw-vars failed dump to json %s", output)
	}
	return jsonPath, nil
}

func ParseUefiVars(ovmfVarsPath string) ([]*BootEntry, []uint16, string, error) {
	jsonPath, err := DumpOvmfVarsToJson(ovmfVarsPath)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "DumpOvmfVarsToJson")
	}

	bootEntry, bootOrder, err := ParseVarsJson(jsonPath)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "ParseVarsJson")
	}
	sort.Slice(bootEntry, func(i, j int) bool {
		return bootEntry[i].ID < bootEntry[j].ID
	})
	return bootEntry, bootOrder, jsonPath, nil
}
