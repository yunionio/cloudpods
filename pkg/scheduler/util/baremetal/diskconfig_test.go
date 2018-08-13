package baremetal

import (
	"encoding/json"
	"reflect"
	"testing"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/scheduler/api"
)

func TestParseDiskConfig(t *testing.T) {
	type args struct {
		desc string
	}

	var tAda int = 2
	var zAda int = 0
	var tStrip64k int64 = 64
	var splits40 string = "40%, "
	var splits100_32 string = "100g,32g,"

	tests := []struct {
		name    string
		args    args
		wantBdc api.BaremetalDiskConfig
		wantErr bool
	}{
		{
			name: "rotate:[1-2,4-5]:MegaRaid",
			args: args{"rotate:[1-2,4-5]:MegaRaid"},
			wantBdc: api.BaremetalDiskConfig{
				Type:   DISK_TYPE_ROTATE,
				Conf:   DISK_CONF_NONE,
				Driver: DISK_DRIVER_MEGARAID,
				Count:  0,
				Range:  []int64{1, 2, 4, 5},
			},
			wantErr: false,
		},
		{
			name: "rotate:[1-2,4,6]:raid10:marvelraid",
			args: args{"rotate:[1-2,4,6]:raid10:marvelraid"},
			wantBdc: api.BaremetalDiskConfig{
				Type:   DISK_TYPE_ROTATE,
				Conf:   DISK_CONF_RAID10,
				Driver: DISK_DRIVER_MARVELRAID,
				Count:  0,
				Range:  []int64{1, 2, 4, 6},
			},
			wantErr: false,
		},
		{
			name: "rotate:[4,6]:raid10",
			args: args{"rotate:[4,6]:raid10"},
			wantBdc: api.BaremetalDiskConfig{
				Type:  DISK_TYPE_ROTATE,
				Conf:  DISK_CONF_RAID10,
				Count: 0,
				Range: []int64{4, 6},
			},
			wantErr: false,
		},
		{
			name: "rotate:[4]:raid10:(40%, )",
			args: args{"rotate:[4]:raid10:(40%, )"},
			wantBdc: api.BaremetalDiskConfig{
				Type:   DISK_TYPE_ROTATE,
				Conf:   DISK_CONF_RAID10,
				Count:  0,
				Splits: splits40,
				Range:  []int64{4},
			},
			wantErr: false,
		},
		{
			name: "[12-13]:raid1:(100g,32g,):adapter0:strip64k",
			args: args{"[12-13]:raid1:(100g,32g,):adapter0:strip64k"},
			wantBdc: api.BaremetalDiskConfig{
				Type:    DISK_TYPE_HYBRID,
				Conf:    DISK_CONF_RAID1,
				Count:   0,
				Splits:  splits100_32,
				Adapter: &zAda,
				Strip:   tStrip64k,
				Range:   []int64{12, 13},
			},
			wantErr: false,
		},
		{
			name: "6:raid5:adapter2",
			args: args{"6:raid5:adapter2"},
			wantBdc: api.BaremetalDiskConfig{
				Type:    DISK_TYPE_HYBRID,
				Conf:    DISK_CONF_RAID5,
				Count:   6,
				Adapter: &tAda,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBdc, err := ParseDiskConfig(tt.args.desc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDiskConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotBdc, tt.wantBdc) {
				t.Errorf("ParseDiskConfig() = %v, want %v", gotBdc, tt.wantBdc)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	type args struct {
		rangeStr string
	}
	tests := []struct {
		name    string
		args    args
		wantRet []int64
		wantErr bool
	}{
		{
			name:    "range number and , test",
			args:    args{"11-7,, 9-9, 10, 15, 1-2,4-5,,"},
			wantRet: []int64{1, 2, 4, 5, 7, 8, 9, 10, 11, 15},
			wantErr: false,
		},
		{
			name:    "range,range",
			args:    args{"1-2,4-5"},
			wantRet: []int64{1, 2, 4, 5},
			wantErr: false,
		},
		{
			name:    "numbers",
			args:    args{"1, 1"},
			wantRet: []int64{1},
			wantErr: false,
		},
		{
			name:    "numbersErr",
			args:    args{"1-, 1"},
			wantRet: nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRet, err := ParseRange(tt.args.rangeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRet, tt.wantRet) {
				t.Errorf("ParseRange() = %v, want %v", gotRet, tt.wantRet)
			}
		})
	}
}

var (
	testStorages = []*api.BaremetalStorage{
		// 0-11 disk on adapter2
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    2861056,
			Adapter: 2,
		},

		// 12-13 disk on adapter0
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    953344,
			Adapter: 0,
		},
		{
			Driver:  DISK_DRIVER_MEGARAID,
			Rotate:  true,
			Size:    953344,
			Adapter: 0,
		},
	}
	bitmainStorage = []*api.BaremetalStorage{
		{
			Driver:       DISK_DRIVER_MARVELRAID,
			Rotate:       true,
			Size:         3814912,
			Adapter:      0,
			Enclosure:    252,
			MaxStripSize: 1024,
			MinStripSize: 64,
			Model:        "K7GTE9JL HGST HUS726040ALE610 APGNT907",
			Slot:         0,
			Status:       "online",
		},
		{
			Driver:       DISK_DRIVER_MARVELRAID,
			Rotate:       true,
			Size:         3814912,
			Adapter:      0,
			Enclosure:    252,
			MaxStripSize: 1024,
			MinStripSize: 64,
			Model:        "K7GTE9JL HGST HUS726040ALE610 APGNT907",
			Slot:         1,
			Status:       "online",
		},
		{
			Driver:       DISK_DRIVER_MARVELRAID,
			Rotate:       true,
			Size:         3814912,
			Adapter:      0,
			Enclosure:    252,
			MaxStripSize: 1024,
			MinStripSize: 64,
			Model:        "K7GTE9JL HGST HUS726040ALE610 APGNT907",
			Slot:         2,
			Status:       "online",
		},
		{
			Driver:       DISK_DRIVER_MARVELRAID,
			Rotate:       true,
			Size:         3814912,
			Adapter:      0,
			Enclosure:    252,
			MaxStripSize: 1024,
			MinStripSize: 64,
			Model:        "K7GTE9JL HGST HUS726040ALE610 APGNT907",
			Slot:         3,
			Status:       "online",
		},
		{
			Driver:       DISK_DRIVER_PCIE,
			Rotate:       true,
			Size:         2200000,
			Adapter:      0,
			Enclosure:    252,
			MaxStripSize: 1024,
			MinStripSize: 64,
			Model:        "NVME test",
			Slot:         4,
			Status:       "online",
		},
		{
			Driver:       DISK_DRIVER_PCIE,
			Rotate:       true,
			Size:         2200000,
			Adapter:      0,
			Enclosure:    252,
			MaxStripSize: 1024,
			MinStripSize: 64,
			Model:        "NVME test",
			Slot:         5,
			Status:       "online",
		},
	}
)

func TestCalculateLayout(t *testing.T) {
	confs, err := NewBaremetalDiskConfigs(
		"[12-13]:raid1:(100g,32g,):adapter0",
		"6:raid5:adapter2",
		"6:raid5:adapter2",
	)
	if err != nil {
		t.Fatalf("NewDiskConfigs err: %v", err)
	}

	expectedLayoutJson := `
[
  {
    "disks": [
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 0,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 953344,
        "index": 12
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 0,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 953344,
        "index": 13
      }
    ],
    "conf": {
      "type": "hybrid",
      "conf": "raid1",
      "count": 0,
      "range": [
        12,
        13
      ],
      "splits": "100g,32g,",
      "adapter": 0,
      "cachedbadbbu": false,
      "strip": 0,
      "ra": false,
      "wt": false,
      "direct": false
    },
    "size": 953344
  },
  {
    "disks": [
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 0
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 1
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 2
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 3
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 4
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 5
      }
    ],
    "conf": {
      "type": "hybrid",
      "conf": "raid5",
      "count": 6,
      "range": null,
      "splits": "",
      "adapter": 2,
      "cachedbadbbu": false,
      "strip": 0,
      "ra": false,
      "wt": false,
      "direct": false
    },
    "size": 14305280
  },
  {
    "disks": [
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 6
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 7
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 8
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 9
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 10
      },
      {
        "slot": 0,
        "status": "",
        "rotate": true,
        "adapter": 2,
        "driver": "MegaRaid",
        "model": "",
        "enclousure": 0,
        "size": 2861056,
        "index": 11
      }
    ],
    "conf": {
      "type": "hybrid",
      "conf": "raid5",
      "count": 6,
      "range": null,
      "splits": "",
      "adapter": 2,
      "cachedbadbbu": false,
      "strip": 0,
      "ra": false,
      "wt": false,
      "direct": false
    },
    "size": 14305280
  }
]
`
	layout, err := CalculateLayout(confs, testStorages)
	if err != nil {
		t.Fatalf("CalculateLayout err: %v", err)
	}

	var expectedLayout []Layout
	err = json.Unmarshal([]byte(expectedLayoutJson), &expectedLayout)
	if err != nil {
		t.Fatalf("Unmarshal expectedLayoutJson err: %v", err)
	}
	if !reflect.DeepEqual(layout, expectedLayout) {
		t.Errorf("CalculateLayout() = %v, want %v", layout, expectedLayout)
	}
}

func TestCheckDisksAllocable(t *testing.T) {
	confs, err := NewBaremetalDiskConfigs(
		"[12-13]:raid1:(100g,32g,):adapter0",
		"6:raid5:adapter2",
		"6:raid5:adapter2",
	)
	bitmainConfs, err := NewBaremetalDiskConfigs("raid10:(60g,)")
	if err != nil {
		t.Fatalf("NewDiskConfigs err: %v", err)
	}

	layout, err := CalculateLayout(confs, testStorages)
	defaultLayout, err := CalculateLayout([]*api.BaremetalDiskConfig{&api.BaremetalDefaultDiskConfig}, testStorages[12:])
	if err != nil {
		t.Fatalf("CalculateLayout err: %v", err)
	}
	bitmainLayout, err := CalculateLayout(bitmainConfs, bitmainStorage)
	if err != nil {
		t.Fatalf("Calculate bitmain layout err: %v", err)
	}

	log.Debugf("defaultLayout: %s", defaultLayout)
	log.Debugf("layout: %s", layout)
	log.Debugf("Bitmain layout: %s", bitmainLayout)

	tdiskDefault := []*api.Disk{
		{Size: -1},
		{Size: -1},
	}

	tdisk1 := []*api.Disk{
		{Size: 960000},
		{Size: -1},
		{Size: -1},
		{Size: -1},
	}

	tdisk2 := []*api.Disk{
		{Size: -1},
		{Size: -1},
		{Size: -1},
		{Size: -1},
	}

	tdisk3 := []*api.Disk{
		{Size: 102398},
		{Size: -1},
		{Size: -1},
		{Size: -1},
	}

	btdisk1 := []*api.Disk{
		{Size: 61438},
		{Size: -1},
	}

	btdisk2 := []*api.Disk{
		{Size: 61440},
		{Size: -1},
	}

	btdisk3 := []*api.Disk{
		{Size: -1},
		{Size: -1},
	}

	btPcieDisk := []*api.Disk{
		{Size: 44440},
		{Size: -1},
		{Size: -1},
		{Size: -1},
		{Size: -1},
	}

	type args struct {
		layouts []Layout
		disks   []*api.Disk
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "default none type config should allocable",
			args: args{
				layouts: defaultLayout,
				disks:   tdiskDefault,
			},
			want: true,
		},
		{
			name: "should not allocable",
			args: args{
				layouts: layout,
				disks:   tdisk1,
			},
			want: false,
		},
		{
			name: "should allocable",
			args: args{
				layouts: layout,
				disks:   tdisk2,
			},
			want: true,
		},
		{
			name: "should allocable2",
			args: args{
				layouts: layout,
				disks:   tdisk3,
			},
			want: true,
		},
		{
			name: "Bitmain allocable 61438 should true",
			args: args{
				layouts: bitmainLayout,
				disks:   btdisk1,
			},
			want: true,
		},
		{
			name: "Bitmain allocable 61440 should false",
			args: args{
				layouts: bitmainLayout,
				disks:   btdisk2,
			},
			want: false,
		},
		{
			name: "Bitmain allocable autoextend should true",
			args: args{
				layouts: bitmainLayout,
				disks:   btdisk3,
			},
			want: true,
		},
		{
			name: "Bitmain allocable PCIE disk should true",
			args: args{
				layouts: bitmainLayout,
				disks:   btPcieDisk,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckDisksAllocable(tt.args.layouts, tt.args.disks); got != tt.want {
				t.Errorf("CheckDisksAllocable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDiskSpec(t *testing.T) {
	type args struct {
		storages []*api.BaremetalStorage
	}
	tests := []struct {
		name string
		args args
		want DiskSpec
	}{
		{
			name: "Only HDD",
			args: args{testStorages},
			want: map[string]SpecSizeCount{
				"HDD": map[string]int{
					"2861056": 12,
					"953344":  2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDiskSpec(tt.args.storages); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDiskSpec() = %v, want %v", got, tt.want)
			} else {
				log.Debugf("GetDiskSpec() = %v", got)
			}
		})
	}
}

var testStorages2 string = `
[
  {
    "adapter": 0,
    "driver": "MarvelRaid",
    "model": "SSDSCKJB120G7R",
    "rotate": true,
    "size": 114473,
    "slot": 0,
    "sn": "PHDW80440155150A"
  },
  {
    "adapter": 0,
    "driver": "MarvelRaid",
    "model": "SSDSCKJB120G7R",
    "rotate": true,
    "size": 114473,
    "slot": 1,
    "sn": "PHDW80440180150A"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "BTYM73040578960CGNSSDSC2KG960G7R SCV1DL56",
    "rotate": false,
    "size": 915200,
    "slot": 0,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "BTYM73040566960CGNSSDSC2KG960G7R SCV1DL56",
    "rotate": false,
    "size": 915200,
    "slot": 1,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "BTYM734403W7960CGNSSDSC2KG960G7R SCV1DL56",
    "rotate": false,
    "size": 915200,
    "slot": 2,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "BTYM734600B1960CGNSSDSC2KG960G7R SCV1DL56",
    "rotate": false,
    "size": 915200,
    "slot": 3,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A192FL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 4,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A1A4FL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 5,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A190FL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 6,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A15WFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 7,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A18CFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 8,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A1ABFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 9,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A1ACFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 10,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012840A23VFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 11,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012840A21SFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 12,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A191FL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 13,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A15UFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 14,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012840A23TFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 15,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A13GFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 16,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A13JFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 17,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A19EFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 18,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A19UFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 19,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A1BSFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 20,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A1B5FL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 21,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A16FFL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 22,
    "status": "offline"
  },
  {
    "adapter": 0,
    "driver": "MegaRaid",
    "enclosure": 64,
    "max_strip_size": 1024,
    "min_strip_size": 64,
    "model": "TOSHIBA AL14SEB18EQY EB012850A165FL7E",
    "rotate": true,
    "size": 1716352,
    "slot": 23,
    "status": "offline"
  }
]
`

func TestStorageLoad(t *testing.T) {
	ss := make([]*api.BaremetalStorage, 0)
	confs, err := NewBaremetalDiskConfigs(
		"2:raid1:MarvelRaid",
		"4:raid10:MegaRaid",
		"raid5",
	)
	if err != nil {
		t.Fatalf("NewDiskConfigs err: %v", err)
	}
	json.Unmarshal([]byte(testStorages2), &ss)
	layout, err := CalculateLayout(confs, ss)
	if err != nil {
		t.Fatalf("CalculateLayout err: %v", err)
	}
	log.Debugf("layout: %s", layout)
}
