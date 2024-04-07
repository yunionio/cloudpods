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

package megactl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseStorcliControllers(t *testing.T) {
	input := `
{
"Controllers":[
{
        "Command Status" : {
                "CLI Version" : "007.1907.0000.0000 Sep 13, 2021",
                "Operating system" : "Linux 5.14.0-4-arm64",
                "Controller" : 0,
                "Status" : "Success",
                "Description" : "Show Drive Information Succeeded."
        },
        "Response Data" : {
                "Drive Information" : [
                        {
                                "EID:Slt" : "65:0",
                                "DID" : 6,
                                "State" : "Onln",
                                "DG" : 0,
                                "Size" : "1.089 TB",
                                "Intf" : "SAS",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "ST1200MM0009    ",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:1",
                                "DID" : 8,
                                "State" : "Onln",
                                "DG" : 0,
                                "Size" : "1.089 TB",
                                "Intf" : "SAS",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "ST1200MM0009    ",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:2",
                                "DID" : 9,
                                "State" : "Onln",
                                "DG" : 0,
                                "Size" : "1.089 TB",
                                "Intf" : "SAS",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "ST1200MM0009    ",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:3",
                                "DID" : 7,
                                "State" : "Onln",
                                "DG" : 0,
                                "Size" : "1.089 TB",
                                "Intf" : "SAS",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "ST1200MM0009    ",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:4",
                                "DID" : 14,
                                "State" : "Onln",
                                "DG" : 0,
                                "Size" : "1.089 TB",
                                "Intf" : "SAS",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "ST1200MM0009    ",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:5",
                                "DID" : 13,
                                "State" : "Onln",
                                "DG" : 0,
                                "Size" : "1.089 TB",
                                "Intf" : "SAS",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "ST1200MM0009    ",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:6",
                                "DID" : 19,
                                "State" : "UGood",
                                "DG" : "F",
                                "Size" : "1.745 TB",
                                "Intf" : "SATA",
                                "Med" : "SSD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "INTEL SSDSC2KB019T8",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:7",
                                "DID" : 20,
                                "State" : "JBOD",
                                "DG" : "-",
                                "Size" : "1.746 TB",
                                "Intf" : "SATA",
                                "Med" : "SSD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "INTEL SSDSC2KB019T8",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:9",
                                "DID" : 15,
                                "State" : "Onln",
                                "DG" : 1,
                                "Size" : "3.637 TB",
                                "Intf" : "SATA",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "MG04ACA400N",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:10",
                                "DID" : 17,
                                "State" : "Onln",
                                "DG" : 1,
                                "Size" : "3.637 TB",
                                "Intf" : "SATA",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "MG04ACA400N",
                                "Sp" : "U",
                                "Type" : "-"
                        },
                        {
                                "EID:Slt" : "65:11",
                                "DID" : 16,
                                "State" : "Onln",
                                "DG" : 1,
                                "Size" : "3.637 TB",
                                "Intf" : "SATA",
                                "Med" : "HDD",
                                "SED" : "N",
                                "PI" : "N",
                                "SeSz" : "512B",
                                "Model" : "MG04ACA400N",
                                "Sp" : "U",
                                "Type" : "-"
                        }
                ]
        }
}
]
}
`
	info, err := parseStorcliControllers(input)
	if err != nil {
		t.Fatalf("parseStorcliControllers error: %v", err)
	}

	assert := assert.New(t)
	assert.Equal(1, len(info.Controllers))
	assert.Equal("ST1200MM0009    ", info.Controllers[0].ResponseData.Info[0].Model)
	assert.Equal("1.089 TB", info.Controllers[0].ResponseData.Info[0].Size)

	pds := info.Controllers[0].ResponseData.Info

	for _, pd := range pds {
		d, err := pd.toMegaraidDev()
		if err != nil {
			t.Fatalf("toMegaraidDev error: %v", err)
		}
		t.Logf("convert megaraid device: %#v", d)
	}
}

func TestStorcliLogicalVolumes_GetLogicalVolumes(t *testing.T) {
	input := `
{
"Controllers":[
{
	"Command Status" : {
		"CLI Version" : "007.1907.0000.0000 Sep 13, 2021",
		"Operating system" : "Linux 5.14.0-4-arm64",
		"Controller" : 0,
		"Status" : "Success",
		"Description" : "None"
	},
	"Response Data" : {
		"/c0/v0" : [
			{
				"DG/VD" : "0/0",
				"TYPE" : "RAID1",
				"State" : "Optl",
				"Access" : "RW",
				"Consist" : "No",
				"Cache" : "RWBD",
				"Cac" : "-",
				"sCC" : "ON",
				"Size" : "446.625 GB",
				"Name" : ""
			}
		],
		"PDs for VD 0" : [
			{
				"EID:Slt" : "8:0",
				"DID" : 16,
				"State" : "Onln",
				"DG" : 0,
				"Size" : "446.625 GB",
				"Intf" : "SATA",
				"Med" : "SSD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "SAMSUNG MZ7LH480HAHQ-00005",
				"Sp" : "U",
				"Type" : "-"
			},
			{
				"EID:Slt" : "8:1",
				"DID" : 17,
				"State" : "Onln",
				"DG" : 0,
				"Size" : "446.625 GB",
				"Intf" : "SATA",
				"Med" : "SSD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "SAMSUNG MZ7LH480HAHQ-00005",
				"Sp" : "U",
				"Type" : "-"
			}
		],
		"VD0 Properties" : {
			"Strip Size" : "256 KB",
			"Number of Blocks" : 936640512,
			"VD has Emulated PD" : "Yes",
			"Span Depth" : 1,
			"Number of Drives Per Span" : 2,
			"Write Cache(initial setting)" : "WriteBack",
			"Disk Cache Policy" : "Disk's Default",
			"Encryption" : "None",
			"Data Protection" : "Disabled",
			"Active Operations" : "None",
			"Exposed to OS" : "Yes",
			"OS Drive Name" : "/dev/sda",
			"Creation Date" : "08-06-2022",
			"Creation Time" : "02:46:23 PM",
			"Emulation type" : "default",
			"Cachebypass size" : "Cachebypass-64k",
			"Cachebypass Mode" : "Cachebypass Intelligent",
			"Is LD Ready for OS Requests" : "Yes",
			"SCSI NAA Id" : "600605b010e943702a3372bf4d29f6a7",
			"Unmap Enabled" : "N/A"
		},
		"/c0/v1" : [
			{
				"DG/VD" : "1/1",
				"TYPE" : "RAID1",
				"State" : "Optl",
				"Access" : "RW",
				"Consist" : "No",
				"Cache" : "RWBD",
				"Cac" : "-",
				"sCC" : "ON",
				"Size" : "558.406 GB",
				"Name" : ""
			}
		],
		"PDs for VD 1" : [
			{
				"EID:Slt" : "8:2",
				"DID" : 15,
				"State" : "Onln",
				"DG" : 1,
				"Size" : "558.406 GB",
				"Intf" : "SAS",
				"Med" : "HDD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "ST600MP0006     ",
				"Sp" : "U",
				"Type" : "-"
			},
			{
				"EID:Slt" : "8:3",
				"DID" : 12,
				"State" : "Onln",
				"DG" : 1,
				"Size" : "558.406 GB",
				"Intf" : "SAS",
				"Med" : "HDD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "ST600MP0006     ",
				"Sp" : "U",
				"Type" : "-"
			}
		],
		"VD1 Properties" : {
			"Strip Size" : "256 KB",
			"Number of Blocks" : 1171062784,
			"VD has Emulated PD" : "No",
			"Span Depth" : 1,
			"Number of Drives Per Span" : 2,
			"Write Cache(initial setting)" : "WriteBack",
			"Disk Cache Policy" : "Disk's Default",
			"Encryption" : "None",
			"Data Protection" : "Disabled",
			"Active Operations" : "None",
			"Exposed to OS" : "Yes",
			"OS Drive Name" : "/dev/sdb",
			"Creation Date" : "08-06-2022",
			"Creation Time" : "02:46:24 PM",
			"Emulation type" : "default",
			"Cachebypass size" : "Cachebypass-64k",
			"Cachebypass Mode" : "Cachebypass Intelligent",
			"Is LD Ready for OS Requests" : "Yes",
			"SCSI NAA Id" : "600605b010e943702a3372c04d2fb6b9",
			"Unmap Enabled" : "N/A"
		},
		"/c0/v2" : [
			{
				"DG/VD" : "2/2",
				"TYPE" : "RAID0",
				"State" : "Optl",
				"Access" : "RW",
				"Consist" : "Yes",
				"Cache" : "NRWTD",
				"Cac" : "-",
				"sCC" : "ON",
				"Size" : "1.090 TB",
				"Name" : ""
			}
		],
		"PDs for VD 2" : [
			{
				"EID:Slt" : "8:4",
				"DID" : 14,
				"State" : "Onln",
				"DG" : 2,
				"Size" : "1.090 TB",
				"Intf" : "SAS",
				"Med" : "HDD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "ST1200MM0129    ",
				"Sp" : "U",
				"Type" : "-"
			}
		],
		"VD2 Properties" : {
			"Strip Size" : "256 KB",
			"Number of Blocks" : 2343174144,
			"VD has Emulated PD" : "Yes",
			"Span Depth" : 1,
			"Number of Drives Per Span" : 1,
			"Write Cache(initial setting)" : "WriteThrough",
			"Disk Cache Policy" : "Disk's Default",
			"Encryption" : "None",
			"Data Protection" : "Disabled",
			"Active Operations" : "None",
			"Exposed to OS" : "Yes",
			"OS Drive Name" : "/dev/sdc",
			"Creation Date" : "08-06-2022",
			"Creation Time" : "02:46:24 PM",
			"Emulation type" : "default",
			"Cachebypass size" : "Cachebypass-64k",
			"Cachebypass Mode" : "Cachebypass Intelligent",
			"Is LD Ready for OS Requests" : "Yes",
			"SCSI NAA Id" : "600605b010e943702a3372c04d3ad143",
			"Unmap Enabled" : "N/A"
		},
		"/c0/v3" : [
			{
				"DG/VD" : "3/3",
				"TYPE" : "RAID0",
				"State" : "Optl",
				"Access" : "RW",
				"Consist" : "Yes",
				"Cache" : "NRWTD",
				"Cac" : "-",
				"sCC" : "ON",
				"Size" : "1.090 TB",
				"Name" : ""
			}
		],
		"PDs for VD 3" : [
			{
				"EID:Slt" : "8:5",
				"DID" : 10,
				"State" : "Onln",
				"DG" : 3,
				"Size" : "1.090 TB",
				"Intf" : "SAS",
				"Med" : "HDD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "ST1200MM0129    ",
				"Sp" : "U",
				"Type" : "-"
			}
		],
		"VD3 Properties" : {
			"Strip Size" : "256 KB",
			"Number of Blocks" : 2343174144,
			"VD has Emulated PD" : "Yes",
			"Span Depth" : 1,
			"Number of Drives Per Span" : 1,
			"Write Cache(initial setting)" : "WriteThrough",
			"Disk Cache Policy" : "Disk's Default",
			"Encryption" : "None",
			"Data Protection" : "Disabled",
			"Active Operations" : "None",
			"Exposed to OS" : "Yes",
			"OS Drive Name" : "/dev/sdd",
			"Creation Date" : "08-06-2022",
			"Creation Time" : "02:46:25 PM",
			"Emulation type" : "default",
			"Cachebypass size" : "Cachebypass-64k",
			"Cachebypass Mode" : "Cachebypass Intelligent",
			"Is LD Ready for OS Requests" : "Yes",
			"SCSI NAA Id" : "600605b010e943702a3372c14d48b67e",
			"Unmap Enabled" : "N/A"
		},
		"/c0/v4" : [
			{
				"DG/VD" : "4/4",
				"TYPE" : "RAID0",
				"State" : "Optl",
				"Access" : "RW",
				"Consist" : "Yes",
				"Cache" : "NRWTD",
				"Cac" : "-",
				"sCC" : "ON",
				"Size" : "1.090 TB",
				"Name" : ""
			}
		],
		"PDs for VD 4" : [
			{
				"EID:Slt" : "8:6",
				"DID" : 9,
				"State" : "Onln",
				"DG" : 4,
				"Size" : "1.090 TB",
				"Intf" : "SAS",
				"Med" : "HDD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "ST1200MM0129    ",
				"Sp" : "U",
				"Type" : "-"
			}
		],
		"VD4 Properties" : {
			"Strip Size" : "256 KB",
			"Number of Blocks" : 2343174144,
			"VD has Emulated PD" : "Yes",
			"Span Depth" : 1,
			"Number of Drives Per Span" : 1,
			"Write Cache(initial setting)" : "WriteThrough",
			"Disk Cache Policy" : "Disk's Default",
			"Encryption" : "None",
			"Data Protection" : "Disabled",
			"Active Operations" : "None",
			"Exposed to OS" : "Yes",
			"OS Drive Name" : "/dev/sde",
			"Creation Date" : "08-06-2022",
			"Creation Time" : "02:46:26 PM",
			"Emulation type" : "default",
			"Cachebypass size" : "Cachebypass-64k",
			"Cachebypass Mode" : "Cachebypass Intelligent",
			"Is LD Ready for OS Requests" : "Yes",
			"SCSI NAA Id" : "600605b010e943702a3372c24d57b796",
			"Unmap Enabled" : "N/A"
		},
		"/c0/v5" : [
			{
				"DG/VD" : "5/5",
				"TYPE" : "RAID0",
				"State" : "Optl",
				"Access" : "RW",
				"Consist" : "Yes",
				"Cache" : "NRWTD",
				"Cac" : "-",
				"sCC" : "ON",
				"Size" : "1.090 TB",
				"Name" : ""
			}
		],
		"PDs for VD 5" : [
			{
				"EID:Slt" : "8:7",
				"DID" : 11,
				"State" : "Onln",
				"DG" : 5,
				"Size" : "1.090 TB",
				"Intf" : "SAS",
				"Med" : "HDD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "ST1200MM0129    ",
				"Sp" : "U",
				"Type" : "-"
			}
		],
		"VD5 Properties" : {
			"Strip Size" : "256 KB",
			"Number of Blocks" : 2343174144,
			"VD has Emulated PD" : "Yes",
			"Span Depth" : 1,
			"Number of Drives Per Span" : 1,
			"Write Cache(initial setting)" : "WriteThrough",
			"Disk Cache Policy" : "Disk's Default",
			"Encryption" : "None",
			"Data Protection" : "Disabled",
			"Active Operations" : "None",
			"Exposed to OS" : "Yes",
			"OS Drive Name" : "/dev/sdf",
			"Creation Date" : "08-06-2022",
			"Creation Time" : "02:46:27 PM",
			"Emulation type" : "default",
			"Cachebypass size" : "Cachebypass-64k",
			"Cachebypass Mode" : "Cachebypass Intelligent",
			"Is LD Ready for OS Requests" : "Yes",
			"SCSI NAA Id" : "600605b010e943702a3372c34d665ecf",
			"Unmap Enabled" : "N/A"
		},
		"/c0/v6" : [
			{
				"DG/VD" : "6/6",
				"TYPE" : "RAID0",
				"State" : "Optl",
				"Access" : "RW",
				"Consist" : "Yes",
				"Cache" : "NRWTD",
				"Cac" : "-",
				"sCC" : "ON",
				"Size" : "1.090 TB",
				"Name" : ""
			}
		],
		"PDs for VD 6" : [
			{
				"EID:Slt" : "8:8",
				"DID" : 13,
				"State" : "Onln",
				"DG" : 6,
				"Size" : "1.090 TB",
				"Intf" : "SAS",
				"Med" : "HDD",
				"SED" : "N",
				"PI" : "N",
				"SeSz" : "512B",
				"Model" : "ST1200MM0129    ",
				"Sp" : "U",
				"Type" : "-"
			}
		],
		"VD6 Properties" : {
			"Strip Size" : "256 KB",
			"Number of Blocks" : 2343174144,
			"VD has Emulated PD" : "Yes",
			"Span Depth" : 1,
			"Number of Drives Per Span" : 1,
			"Write Cache(initial setting)" : "WriteThrough",
			"Disk Cache Policy" : "Disk's Default",
			"Encryption" : "None",
			"Data Protection" : "Disabled",
			"Active Operations" : "None",
			"Exposed to OS" : "Yes",
			"OS Drive Name" : "/dev/sdg",
			"Creation Date" : "08-06-2022",
			"Creation Time" : "02:46:28 PM",
			"Emulation type" : "default",
			"Cachebypass size" : "Cachebypass-64k",
			"Cachebypass Mode" : "Cachebypass Intelligent",
			"Is LD Ready for OS Requests" : "Yes",
			"SCSI NAA Id" : "600605b010e943702a3372c44d751bab",
			"Unmap Enabled" : "N/A"
		}
	}
}
]
}`
	parser, err := parseStorcliLVs(input)
	if err != nil {
		t.Fatalf("parseStorcliLVs: %v", err)
	}
	lvs, err := parser.GetLogicalVolumes(0)
	if err != nil {
		t.Fatalf("GetLogicalVolumes: %v", err)
	}

	assert := assert.New(t)
	assert.Equal(7, len(lvs), "Should 7 logical volumes")

	for i := range lvs {
		switch lvs[i].Index {
		case 0:
			first2SDDs := lvs[i].PDs
			assert.Equal("SSD", first2SDDs[0].Med)
			assert.Equal("446.625 GB", first2SDDs[0].Size)
			assert.Equal("SSD", first2SDDs[1].Med)
			assert.Equal("446.625 GB", first2SDDs[1].Size)
			assert.Equal(true, lvs[i].IsSSD())
			assert.Equal("/dev/sda", lvs[i].Properties.DeviceName)
		case 6:
			assert.Equal("/dev/sdg", lvs[i].Properties.DeviceName)
		}
	}
}
