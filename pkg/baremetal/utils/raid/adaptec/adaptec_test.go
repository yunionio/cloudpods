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

package adaptec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getAdaptorIndex(t *testing.T) {
	tests := []struct {
		name string
		args string
		want int
	}{
		{
			name: "Controllers found: 1",
			args: "Controllers found: 1",
			want: -1,
		},
		{
			name: "   Controller 1:             : Optimal, Slot 4, RAID (Expose RAW), Adaptec ASR8805, 6A2263667CA, 50000D1701B47780",
			args: "   Controller 1:             : Optimal, Slot 4, RAID (Expose RAW), Adaptec ASR8805, 6A2263667CA, 50000D1701B47780",
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAdaptorIndex(tt.args); got != tt.want {
				t.Errorf("getAdaptorIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAdaptorInfo(t *testing.T) {
	input := `Controllers found: 1
----------------------------------------------------------------------
Controller information
----------------------------------------------------------------------
   Controller Status                        : Optimal
   Controller Mode                          : RAID (Expose RAW)
   Channel description                      : SAS/SATA
   Controller Model                         : Adaptec ASR8805
   Controller Serial Number                 : 6A2263667CA
   Controller World Wide Name               : 50000D1701B47780
   Controller Alarm                         : Disabled
   Physical Slot                            : 4
   Temperature                              : 42 C/ 107 F (Normal)
   Installed memory                         : 1024 MB
   Host bus type                            : PCIe
   Host bus speed                           : 8000 MHz
   Host bus link width                      : 8 bit(s)/link(s)
   Global task priority                     : High
   Performance Mode                         : Default/Dynamic
   PCI Device ID                            : 653
   Stayawake period                         : Disabled
   Spinup limit internal drives             : 0
   Spinup limit external drives             : 0
   Defunct disk drive count                 : 0
   NCQ status                               : Enabled
   Statistics data collection mode          : Disabled
   Monitor Log Severity Level               : Informational
   Global Max SAS Phy Link Rate             : 12 Gbps
   Verify Write Setting                     : Not Applicable
   Save Custom Defaults Setting             : Disabled
   Smart Poll                               : Enabled
   Error Tunable Profile                    : Normal
   --------------------------------------------------------
   Cache Properties
   --------------------------------------------------------
   Controller Cache Preservation            : Disabled
   Global Physical Device Write Cache Policy: Drive Specific
   --------------------------------------------------------
   RAID Properties
   --------------------------------------------------------
   Logical devices/Failed/Degraded          : 0/0/0
   Copyback                                 : Disabled
   Automatic Failover                       : Enabled
   Background consistency check             : Disabled
   Background consistency check period      : 0
   --------------------------------------------------------
   Controller BIOS Setting Information
   --------------------------------------------------------
   Runtime BIOS                             : Enabled
   Array BBS Support                        : Disabled
   Physical Drives Displayed during POST    : Disabled
   Backplane Mode                           : IBPI
   BIOS Halt on Missing Drive Count         : 255
   --------------------------------------------------------
   Controller Version Information
   --------------------------------------------------------
   BIOS                                     : 7.18-0 (33556)
   Firmware                                 : 7.18-0 (33556)
   Driver                                   : 1.2-1 (50983)
   Boot Flash                               : 7.18-0 (33556)
   CPLD (Load version/ Flash version)       : 8/ 11
   SEEPROM (Load version/ Flash version)    : 1/ 1
   FCT Custom Init String Version           : 0x0

   --------------------------------------------------------
   Controller Cache Backup Unit Information
   --------------------------------------------------------

    Overall Backup Unit Status              : Not Present
   --------------------------------------------------------
   Connector information

   --------------------------------------------------------
   Connector #0
      Connector Name                        : CN0

      --------------------------------------
   Lane Information
      --------------------------------------
      Lane #0
         Channel ID                         : 0
         Device ID                          : 0
         SAS Address                        : 50000D1701B47780
         PHY Identifier                     : 3
         -----------------------------------
         Lane SAS Phy Information
         -----------------------------------
            SAS Address                     : 50000D1701B47780

      Lane #1
         Channel ID                         : 0
         Device ID                          : 1
         SAS Address                        : 50000D1701B47780
         PHY Identifier                     : 2
         -----------------------------------
         Lane SAS Phy Information
         -----------------------------------
            SAS Address                     : 50000D1701B47780

      Lane #2
         Channel ID                         : 0
         Device ID                          : 2
         SAS Address                        : 50000D1701B47780
         PHY Identifier                     : 0
         -----------------------------------
         Lane SAS Phy Information
         -----------------------------------
            SAS Address                     : 50000D1701B47780

      Lane #3
         Channel ID                         : 0
         Device ID                          : 3
         SAS Address                        : 50000D1701B47780
         PHY Identifier                     : 1
         -----------------------------------
         Lane SAS Phy Information
         -----------------------------------
            SAS Address                     : 50000D1701B47780


   Connector #1
      Connector Name                        : CN1

      --------------------------------------
   Lane Information
      --------------------------------------
      Lane #0
         Channel ID                         : 0
         Device ID                          : 4
         SAS Address                        : 50000D1701B47780
         PHY Identifier                     : 5
         -----------------------------------
         Lane SAS Phy Information
         -----------------------------------
            SAS Address                     : 50000D1701B47780

      Lane #1
         Channel ID                         : 0
         Device ID                          : 5
         SAS Address                        : 50000D1701B47780
         PHY Identifier                     : 6
         -----------------------------------
         Lane SAS Phy Information
         -----------------------------------
            SAS Address                     : 50000D1701B47780

      Lane #2
         Channel ID                         : 0
         Device ID                          : 6
         SAS Address                        : 50000D1701B47780
         PHY Identifier                     : 4
         -----------------------------------
         Lane SAS Phy Information
         -----------------------------------
            SAS Address                     : 50000D1701B47780
            Attached PHY Identifier         : 0
            Attached SAS Address            : 5000C50077893141
            Negotiated Logical Link Rate    : PHY enabled - 6 Gbps

      Lane #3
         Channel ID                         : 0
         Device ID                          : 7
         SAS Address                        : 50000D1701B47780
         PHY Identifier                     : 7
         -----------------------------------
         Lane SAS Phy Information
         -----------------------------------
            SAS Address                     : 50000D1701B47780
            Attached PHY Identifier         : 0
            Attached SAS Address            : 50000395B839D9AE
            Negotiated Logical Link Rate    : PHY enabled - 6 Gbps




Command completed successfully.`
	lines := strings.Split(input, "\n")
	info, err := getAdaptorInfo(lines)
	if err != nil {
		t.Errorf("getAdaptorInfo: %v", err)
	}
	assert := assert.New(t)
	assert.Equal("Optimal", info.status)
	assert.Equal("RAID (Expose RAW)", info.mode)
	assert.Equal(true, info.isRaidExposeRawMode(), "should be raid expose raw mode")
	assert.Equal("Adaptec ASR8805", info.name)
	assert.Equal("6A2263667CA", info.sn)
	assert.Equal("50000D1701B47780", info.wwn)
	assert.Equal("4", info.slot)
}

func Test_getPhyDevices(t *testing.T) {
	input := `Controllers found: 1
----------------------------------------------------------------------
Physical Device information
----------------------------------------------------------------------
      Device #0
         Device is a Hard drive
         State                              : Online
         Block Size                         : 512 Bytes
         Supported                          : Yes
         Programmed Max Speed               : SAS 6.0 Gb/s
         Transfer Speed                     : SAS 6.0 Gb/s
         Reported Channel,Device(T:L)       : 0,6(6:0)
         Reported Location                  : Connector 1, Device 2
         Vendor                             : SEAGATE
         Model                              : ST300MM0006
         Firmware                           : LS0A
         Serial number                      : S0K30A4N
         World-wide name                    : 5000C50077893140
         Reserved Size                      : 415982 KB
         Used Size                          : 285696 MB
         Unused Size                        : 64 KB
         Total Size                         : 286102 MB
         Write Cache                        : Disabled (write-through)
         FRU                                : None
         S.M.A.R.T.                         : No
         S.M.A.R.T. warnings                : 0
         Power State                        : Full rpm
         Supported Power States             : Full rpm,Powered off
         SSD                                : No
         Temperature                        : 34 C/ 93 F
      ----------------------------------------------------------------
      Device Phy Information
      ----------------------------------------------------------------
         Phy #0
            PHY Identifier                  : 0
            SAS Address                     : 5000C50077893141
            Attached PHY Identifier         : 4
            Attached SAS Address            : 50000D1701B47780
         Phy #1
            PHY Identifier                  : 1
            SAS Address                     : 5000C50077893142
      ----------------------------------------------------------------
      Runtime Error Counters
      ----------------------------------------------------------------
         Hardware Error Count               : 0
         Medium Error Count                 : 0
         Parity Error Count                 : 0
         Link Failure Count                 : 0
         Aborted Command Count              : 0
         SMART Warning Count                : 0

      Device #1
         Device is a Hard drive
         State                              : Online
         Block Size                         : 512 Bytes
         Supported                          : Yes
         Programmed Max Speed               : SAS 6.0 Gb/s
         Transfer Speed                     : SAS 6.0 Gb/s
         Reported Channel,Device(T:L)       : 0,7(7:0)
         Reported Location                  : Connector 1, Device 3
         Vendor                             : TOSHIBA
         Model                              : AL13SEB300
         Firmware                           : DE0D
         Serial number                      : 84T0A31SFRD6
         World-wide name                    : 50000395B839D9AC
         Reserved Size                      : 415982 KB
         Used Size                          : 285696 MB
         Unused Size                        : 64 KB
         Total Size                         : 286102 MB
         Write Cache                        : Disabled (write-through)
         FRU                                : None
         S.M.A.R.T.                         : No
         S.M.A.R.T. warnings                : 0
         Power State                        : Full rpm
         Supported Power States             : Full rpm,Powered off
         SSD                                : No
         Temperature                        : 33 C/ 91 F
      ----------------------------------------------------------------
      Device Phy Information
      ----------------------------------------------------------------
         Phy #0
            PHY Identifier                  : 0
            SAS Address                     : 50000395B839D9AE
            Attached PHY Identifier         : 7
            Attached SAS Address            : 50000D1701B47780
         Phy #1
            PHY Identifier                  : 1
            SAS Address                     : 50000395B839D9AF
      ----------------------------------------------------------------
      Runtime Error Counters
      ----------------------------------------------------------------
         Hardware Error Count               : 0
         Medium Error Count                 : 0
         Parity Error Count                 : 0
         Link Failure Count                 : 0
         Aborted Command Count              : 0
         SMART Warning Count                : 0



Command completed successfully.`
	lines := strings.Split(input, "\n")
	devs := getPhyDevices(1, lines)
	assert := assert.New(t)
	assert.Equal(2, len(devs), "physical devices should be 2")
	dev1 := devs[0]
	assert.Equal("0", dev1.channelId)
	assert.Equal("6", dev1.deviceId)
	assert.Equal("Online", dev1.Status)
	assert.Equal("ST300MM0006", dev1.Model)
	assert.Equal(true, dev1.Rotate.Bool())

	dev2 := devs[1]
	assert.Equal("0", dev2.channelId)
	assert.Equal("7", dev2.deviceId)
	assert.Equal("Online", dev2.Status)
	assert.Equal("AL13SEB300", dev2.Model)
	assert.Equal(true, dev2.Rotate.Bool())
}

func Test_getLogicalVolums(t *testing.T) {
	input := `Controllers found: 1
----------------------------------------------------------------------
Logical device information
----------------------------------------------------------------------
Logical Device number 0
   Logical Device name                      : LogicalDrv 0
   Block Size of member drives              : 512 Bytes
   RAID level                               : 1
   Unique Identifier                        : 7D30DCD4
   Status of Logical Device                 : Optimal
   Additional details                       : Quick initialized
   Size                                     : 285686 MB
   Parity space                             : 285696 MB
   Interface Type                           : Serial Attached SCSI
   Device Type                              : HDD
   Read-cache setting                       : Enabled
   Read-cache status                        : On
   Write-cache setting                      : Enabled
   Write-cache status                       : On
   Partitioned                              : No
   Protected by Hot-Spare                   : No
   Bootable                                 : Yes
   Failed stripes                           : No
   Power settings                           : Disabled
   --------------------------------------------------------
   Logical Device segment information
   --------------------------------------------------------
   Segment 0                                : Present (286102MB, SAS, HDD, Connector:1, Device:2)             S0K30A4N
   Segment 1                                : Present (286102MB, SAS, HDD, Connector:1, Device:3)         84T0A31SFRD6



Command completed successfully.`
	lines := strings.Split(input, "\n")
	lvs, err := getLogicalVolumes(1, lines)
	if err != nil {
		t.Errorf("getLogicalVolumes: %v", err)
	}
	assert := assert.New(t)
	assert.Equal(1, len(lvs), "logical volume len should be 1")
	lv := lvs[0]
	assert.Equal(1, lv.Adapter)
	assert.Equal(0, lv.Index)

	inputNoLV := `Controllers found: 1
----------------------------------------------------------------------
Logical device information
----------------------------------------------------------------------
   No logical devices configured


Command completed successfully.`
	lines = strings.Split(inputNoLV, "\n")
	lvs, err = getLogicalVolumes(1, lines)
	if err != nil {
		t.Errorf("getLogicalVolumes: %v", err)
	}
	assert.Equal(0, len(lvs), "logical volume len should be 0")
}
