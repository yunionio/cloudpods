package fileutils2

import (
	"reflect"
	"testing"
)

var (
	sdaOut = []string{
		"Model: QEMU QEMU HARDDISK (scsi)\n",
		"Disk /dev/sda: 204800000s\n",
		"Sector size (logical/physical): 512B/512B\n",
		"Partition Table: msdos\n",
		"Disk Flags:",
		"",
		"Number  Start       End         Size        File system     Name                  Flags\n",
		" 1      63s    204796619s  204796557s  primary\n",
	}
	sdbOut = []string{
		"Model: ATA PH5-SE128G+ (scsi)\n",
		"Disk /dev/sdb: 250069680s\n",
		"Sector size (logical/physical): 512B/512B\n",
		"Partition Table: gpt\n",
		"Disk Flags:",
		"",
		"Number  Start       End         Size        File system     Name                  Flags\n",
		" 1      2048s       1023999s    1021952s    ntfs         Basic data partition          hidden, diag\n",
		" 2      1024000s    1226751s    202752s     fat32        EFI          boot\n",
		" 3      1226752s    1259519s    32768s                   Microsoft reserved partition  msftres\n",
		" 4      1259520s    105734143s  104474624s  ntfs         Basic data partition          msftdata\n",
		" 5      105734144s  250069646s  144335503s  xfs          Linux filesystem\n",
	}
)

func TestParseDiskPartitions(t *testing.T) {
	type args struct {
		dev   string
		lines []string
	}
	tests := []struct {
		name  string
		args  args
		want  []Partition
		want1 string
	}{
		{
			name: "sdaInput",
			args: args{
				dev:   "/dev/sda",
				lines: sdaOut,
			},
			want: []Partition{
				NewPartition(1, false, 63, 204796619, 204796557, "primary", "", "/dev/sda1"),
			},
			want1: "msdos",
		},
		{
			name: "sdbInput",
			args: args{
				dev:   "/dev/sdb",
				lines: sdbOut,
			},
			want: []Partition{
				NewPartition(1, false, 2048, 1023999, 1021952, "Basic", "ntfs", "/dev/sdb1"),
				NewPartition(2, true, 1024000, 1226751, 202752, "EFI", "fat32", "/dev/sdb2"),
				NewPartition(3, false, 1226752, 1259519, 32768, "Microsoft", "", "/dev/sdb3"),
				NewPartition(4, false, 1259520, 105734143, 104474624, "Basic", "ntfs", "/dev/sdb4"),
				NewPartition(5, false, 105734144, 250069646, 144335503, "Linux", "xfs", "/dev/sdb5"),
			},
			want1: "gpt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ParseDiskPartitions(tt.args.dev, tt.args.lines)
			for i, gotD := range got {
				if !reflect.DeepEqual(gotD, tt.want[i]) {
					t.Errorf("ParseDiskPartitions() got[%d] = %v, want %v", i, gotD, tt.want[i])
				}
			}
			if got1 != tt.want1 {
				t.Errorf("ParseDiskPartitions() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
