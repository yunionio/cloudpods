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

package azure

type ClassicVMSize struct {
	MemoryInMB    int
	NumberOfCores int8
	StorageSize   int
	MaxNic        int
}

var CLASSIC_VM_SIZES = map[string]ClassicVMSize{
	"ExtraSmall":      {MemoryInMB: 786, NumberOfCores: 1, StorageSize: 20, MaxNic: 1},
	"Small":           {MemoryInMB: 1.75 * 1024, NumberOfCores: 1, StorageSize: 225, MaxNic: 1},
	"Medium":          {MemoryInMB: 3.5 * 1024, NumberOfCores: 2, StorageSize: 490, MaxNic: 1},
	"Large":           {MemoryInMB: 7 * 1024, NumberOfCores: 4, StorageSize: 1000, MaxNic: 2},
	"ExtraLarge":      {MemoryInMB: 14 * 1024, NumberOfCores: 8, StorageSize: 2040, MaxNic: 4},
	"A5":              {MemoryInMB: 14 * 1024, NumberOfCores: 2, StorageSize: 490, MaxNic: 1},
	"A6":              {MemoryInMB: 28 * 1024, NumberOfCores: 4, StorageSize: 1000, MaxNic: 2},
	"A7":              {MemoryInMB: 56 * 1024, NumberOfCores: 8, StorageSize: 2040, MaxNic: 4},
	"A8*":             {MemoryInMB: 56 * 1024, NumberOfCores: 8, StorageSize: 1817, MaxNic: 2},
	"A9*":             {MemoryInMB: 112 * 1024, NumberOfCores: 16, StorageSize: 1817, MaxNic: 4},
	"A10":             {MemoryInMB: 56 * 1024, NumberOfCores: 8, StorageSize: 1817, MaxNic: 2},
	"A11":             {MemoryInMB: 112 * 1024, NumberOfCores: 16, StorageSize: 1817, MaxNic: 4},
	"Standard_A1_v2":  {MemoryInMB: 2 * 1024, NumberOfCores: 1, StorageSize: 10, MaxNic: 1},
	"Standard_A2_v2":  {MemoryInMB: 4 * 1024, NumberOfCores: 2, StorageSize: 20, MaxNic: 2},
	"Standard_A4_v2":  {MemoryInMB: 8 * 1024, NumberOfCores: 4, StorageSize: 40, MaxNic: 4},
	"Standard_A8_v2":  {MemoryInMB: 16 * 1024, NumberOfCores: 8, StorageSize: 80, MaxNic: 8},
	"Standard_A2m_v2": {MemoryInMB: 16 * 1024, NumberOfCores: 2, StorageSize: 20, MaxNic: 2},
	"Standard_A4m_v2": {MemoryInMB: 32 * 1024, NumberOfCores: 4, StorageSize: 40, MaxNic: 4},
	"Standard_A8m_v2": {MemoryInMB: 64 * 1024, NumberOfCores: 8, StorageSize: 80, MaxNic: 8},
	"Standard_D1":     {MemoryInMB: 3 * 1024, NumberOfCores: 1, StorageSize: 50, MaxNic: 1},
	"Standard_D2":     {MemoryInMB: 7 * 1024, NumberOfCores: 2, StorageSize: 100, MaxNic: 2},
	"Standard_D3":     {MemoryInMB: 14 * 1024, NumberOfCores: 4, StorageSize: 200, MaxNic: 4},
	"Standard_D4":     {MemoryInMB: 28 * 1024, NumberOfCores: 8, StorageSize: 400, MaxNic: 8},
	"Standard_D11":    {MemoryInMB: 14 * 1024, NumberOfCores: 2, StorageSize: 100, MaxNic: 2},
	"Standard_D12":    {MemoryInMB: 28 * 1024, NumberOfCores: 4, StorageSize: 200, MaxNic: 4},
	"Standard_D13":    {MemoryInMB: 56 * 1024, NumberOfCores: 8, StorageSize: 400, MaxNic: 8},
	"Standard_D14":    {MemoryInMB: 112 * 1024, NumberOfCores: 16, StorageSize: 800, MaxNic: 8},
	"Standard_D1_v2":  {MemoryInMB: 3 * 1024, NumberOfCores: 1, StorageSize: 50, MaxNic: 1},
	"Standard_D2_v2":  {MemoryInMB: 7 * 1024, NumberOfCores: 2, StorageSize: 100, MaxNic: 2},
	"Standard_D3_v2":  {MemoryInMB: 14 * 1024, NumberOfCores: 4, StorageSize: 200, MaxNic: 4},
	"Standard_D4_v2":  {MemoryInMB: 28 * 1024, NumberOfCores: 8, StorageSize: 400, MaxNic: 8},
	"Standard_D5_v2":  {MemoryInMB: 56 * 1024, NumberOfCores: 16, StorageSize: 800, MaxNic: 8},
	"Standard_D11_v2": {MemoryInMB: 14 * 1024, NumberOfCores: 2, StorageSize: 100, MaxNic: 2},
	"Standard_D12_v2": {MemoryInMB: 28 * 1024, NumberOfCores: 4, StorageSize: 200, MaxNic: 4},
	"Standard_D13_v2": {MemoryInMB: 56 * 1024, NumberOfCores: 8, StorageSize: 400, MaxNic: 8},
	"Standard_D14_v2": {MemoryInMB: 112 * 1024, NumberOfCores: 16, StorageSize: 800, MaxNic: 8},
	"Standard_D15_v2": {MemoryInMB: 140 * 1024, NumberOfCores: 20, StorageSize: 1, MaxNic: 8},
	"Standard_D2_v3":  {MemoryInMB: 8 * 1024, NumberOfCores: 2, StorageSize: 50, MaxNic: 2},
	"Standard_D4_v3":  {MemoryInMB: 16 * 1024, NumberOfCores: 4, StorageSize: 100, MaxNic: 2},
	"Standard_D8_v3":  {MemoryInMB: 32 * 1024, NumberOfCores: 8, StorageSize: 200, MaxNic: 4},
	"Standard_D16_v3": {MemoryInMB: 64 * 1024, NumberOfCores: 16, StorageSize: 400, MaxNic: 8},
	"Standard_D32_v3": {MemoryInMB: 128 * 1024, NumberOfCores: 32, StorageSize: 800, MaxNic: 8},
	"Standard_D64_v3": {MemoryInMB: 256 * 1024, NumberOfCores: 64, StorageSize: 1600, MaxNic: 8},
	"Standard_E2_v3":  {MemoryInMB: 16 * 1024, NumberOfCores: 2, StorageSize: 50, MaxNic: 2},
	"Standard_E4_v3":  {MemoryInMB: 32 * 1024, NumberOfCores: 4, StorageSize: 100, MaxNic: 2},
	"Standard_E8_v3":  {MemoryInMB: 64 * 1024, NumberOfCores: 8, StorageSize: 200, MaxNic: 4},
	"Standard_E16_v3": {MemoryInMB: 128 * 1024, NumberOfCores: 16, StorageSize: 400, MaxNic: 8},
	"Standard_E32_v3": {MemoryInMB: 256 * 1024, NumberOfCores: 32, StorageSize: 800, MaxNic: 8},
	"Standard_E64_v3": {MemoryInMB: 432 * 1024, NumberOfCores: 64, StorageSize: 1600, MaxNic: 8},
	"Standard_G1":     {MemoryInMB: 28 * 1024, NumberOfCores: 2, StorageSize: 384, MaxNic: 1},
	"Standard_G2":     {MemoryInMB: 56 * 1024, NumberOfCores: 4, StorageSize: 768, MaxNic: 2},
	"Standard_G3":     {MemoryInMB: 112 * 1024, NumberOfCores: 8, StorageSize: 1, MaxNic: 4},
	"Standard_G4":     {MemoryInMB: 224 * 1024, NumberOfCores: 16, StorageSize: 3, MaxNic: 8},
	"Standard_G5":     {MemoryInMB: 448 * 1024, NumberOfCores: 32, StorageSize: 6, MaxNic: 8},
}
