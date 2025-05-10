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
	"encoding/json"
	"fmt"
	"io/ioutil"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

// VarsData represents the UEFI variables data
type VarsData struct {
	Version   int        `json:"version"`
	Variables []Variable `json:"variables"`
}

// Variable represents a UEFI variable
type Variable struct {
	Name string `json:"name"`
	GUID string `json:"guid"`
	Attr int    `json:"attr"`
	Data string `json:"data"`
}

// EFI_GLOBAL_VARIABLE_GUID is the GUID for EFI global variables
const EFI_GLOBAL_VARIABLE_GUID = "8be4df61-93ca-11d2-aa0d-00e098032b8c"

// ParseVarsJson parses UEFI variables from a JSON file
func ParseVarsJson(jsonPath string) ([]*BootEntry, []uint16, error) {
	// Read JSON file
	data, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read JSON file: %v", err)
	}

	// Parse JSON
	var varsData VarsData
	err = json.Unmarshal(data, &varsData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Parse boot entries and boot order
	var bootEntries []*BootEntry
	var bootOrder []uint16

	for _, v := range varsData.Variables {
		// Check if this is a boot entry
		if len(v.Name) >= 8 && v.Name[:4] == "Boot" && v.GUID == EFI_GLOBAL_VARIABLE_GUID {
			// Check if this is the boot order
			if v.Name == "BootOrder" {
				var err error
				bootOrder, err = ParseBootOrder(v.Data)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse boot order: %v", err)
				}
				continue
			}

			// Parse boot entry
			name, devPaths, err := ParseBootEntryData(v.Data)
			if err != nil {
				log.Errorf("failed to parse boot entry %s: %s", v.Name, err)
				continue
			}

			// Create boot entry
			entry := &BootEntry{
				ID:       v.Name,
				Name:     name,
				DevPaths: devPaths,
				RawData:  v.Data,
			}

			// Add entry to list
			bootEntries = append(bootEntries, entry)
		}
	}

	return bootEntries, bootOrder, nil
}

// UpdateBootOrderInJson updates the boot order in a UEFI variables JSON file
func UpdateBootOrderInJson(jsonPath string, bootOrder []uint16) error {
	// Read JSON file
	data, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %v", err)
	}

	// Parse JSON
	var varsData VarsData
	err = json.Unmarshal(data, &varsData)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	bootOrderHex := BuildBootOrderHex(bootOrder)
	bootOrderFound := false
	for i, v := range varsData.Variables {
		if v.Name == "BootOrder" && v.GUID == EFI_GLOBAL_VARIABLE_GUID {
			varsData.Variables[i].Data = bootOrderHex
			bootOrderFound = true
			break
		}
	}

	// Add boot order if not found
	if !bootOrderFound {
		varsData.Variables = append(varsData.Variables, Variable{
			Name: "BootOrder",
			GUID: EFI_GLOBAL_VARIABLE_GUID,
			Attr: 7, // NV+BS+RT
			Data: bootOrderHex,
		})
	}

	// Write updated JSON
	updatedData, err := json.MarshalIndent(varsData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	err = ioutil.WriteFile(jsonPath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	return nil
}

// ApplyJsonToVars applies a JSON file to OVMF_VARS.fd
func ApplyJsonToVars(jsonPath, inputVarsPath, outputVarsPath string) ([]byte, error) {
	// Execute virt-fw-vars to apply JSON
	output, err := procutils.NewCommand("virt-fw-vars", "-i", inputVarsPath, "-o", outputVarsPath, "--set-json", jsonPath).Output()
	if err != nil {
		return output, fmt.Errorf("failed to execute virt-fw-vars --set-json command: %v", err)
	}

	return output, nil
}
