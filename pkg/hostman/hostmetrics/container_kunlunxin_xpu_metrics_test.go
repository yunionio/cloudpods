package hostmetrics

import (
	"strings"
	"testing"
)

const xpuSmiOutput = `
==============XPUSMI LOG==============

Timestamp                                 : Sat Oct 10 07:46:41 2026
Driver Version                            : 5.19.0.0
XPU-RT Version                            : 10.2

Attached XPUs                             : 8
XPU 00000000:03:00.0
    Product Name                          : P800 OAM
    Product Brand                         : KUNLUNXIN
    Product Architecture                 : KL3
    Serial Number                         : 02K15K6252V00115
    XPU UUID                              : GPU-a0ac042a-5a14-5481-bf7e-dd8e523792cd
    Minor Number                          : 0
    OAM Id                                : 3
    Firmware Version
        PBL Version                       : 1.0
        PCIE Version                      : 2.14
    PCI
        Bus                               : 0x03
        Device                            : 0x00
        Function                          : 0x0
        Domain                            : 0x0000
        Device Id                         : 0x36881D22
        Bus Id                            : 00000000:03:00.0
        Sub System Id                     : 0x00010001
        XPU Link Info
            PCIe Generation
                Max                       : 4
                Current                   : 4
            Link Width
                Max                       : 16x
                Current                   : 16x
    Memory Usage
        Total                             : 98304 MiB
        Reserved                          : 0 MiB
        Used                              : 26 MiB
        Free                              : 98278 MiB
   L3 Usage
       Total                             : 96 MiB
       Reserved                          : 0 MiB
       Used                              : 0 MiB
       Free                              : 96 MiB
   Utilization
       Xpu                               : 0 %
   Temperature
       XPU Current Temp                  : 36 C
   Power Readings
       Enforced Power Limit              : 400.00 W
       Power Draw                        : 87.00 W
   Clocks
       Cluster                           : 1450 MHz
       CDNN                              : 1450 MHz
   Processes
       XPU instance ID                   : N/A
       Compute instance ID               : N/A
       Process ID                        : 2501352
           Type                          : C
           Name                          : ./test_cuda
           Used XPU Memory               : 14 MiB
       XPU instance ID                   : N/A
       Compute instance ID               : N/A
       Process ID                        : 2501354
           Type                          : C
           Name                          : ./test_cuda
           Used XPU Memory               : 12 MiB

XPU 00000000:05:00.0
    Product Name                          : P800 OAM
    Product Brand                         : KUNLUNXIN
    Minor Number                          : 1
    OAM Id                                : 2
    PCI
        Bus Id                            : 00000000:05:00.0
    Memory Usage
        Total                             : 98304 MiB
        Reserved                          : 0 MiB
        Used                              : 26 MiB
        Free                              : 98278 MiB
   L3 Usage
       Total                             : 96 MiB
   Processes
       XPU instance ID                   : N/A
       Compute instance ID               : N/A
       Process ID                        : 2501999
           Type                          : C
           Name                          : ./train
           Used XPU Memory               : 26 MiB
`

func TestParseKunlunxinXpuProcessMetrics(t *testing.T) {
	got := parseKunlunxinXpuProcessMetrics(strings.NewReader(xpuSmiOutput))

	want := []KunlunxinXpuProcessMetrics{
		{Pid: "2501352", PciAddr: "00000000:03:00.0", MemUtil: 14.0 / 98304 * 100, Mem: 14, Idx: 0},
		{Pid: "2501354", PciAddr: "00000000:03:00.0", MemUtil: 12.0 / 98304 * 100, Mem: 12, Idx: 0},
		{Pid: "2501999", PciAddr: "00000000:05:00.0", MemUtil: 26.0 / 98304 * 100, Mem: 26, Idx: 1},
	}

	if len(got) != len(want) {
		t.Fatalf("进程数量不符: got %d, want %d\n got=%+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 个进程不符:\n got=%+v\nwant=%+v", i, got[i], want[i])
		}
	}
}

func TestParseKunlunxinXpuProcessMetricsMemTotalFromMemoryUsage(t *testing.T) {
	in := `XPU 00000000:03:00.0
    Minor Number                          : 3
    Memory Usage
        Total                             : 98304 MiB
   L3 Usage
       Total                             : 96 MiB
   Processes
       Process ID                        : 1001
           Used XPU Memory               : 98304 MiB
`
	got := parseKunlunxinXpuProcessMetrics(strings.NewReader(in))

	if len(got) != 1 {
		t.Fatalf("got %d processes, want 1: %+v", len(got), got)
	}
	if got[0].MemUtil != 100 {
		t.Errorf("MemUtil = %v, want 100（应按 98304 MiB 而非 L3 的 96 MiB 计算）", got[0].MemUtil)
	}
	if got[0].Idx != 3 || got[0].PciAddr != "00000000:03:00.0" || got[0].Pid != "1001" || got[0].Mem != 98304 {
		t.Errorf("got=%+v, want{Pid:1001 PciAddr:00000000:03:00.0 MemUtil:100 Mem:98304 Idx:3}", got[0])
	}
}

func TestParseKunlunxinXpuProcessMetricsSkipEmpty(t *testing.T) {
	in := `==============XPUSMI LOG==============
Timestamp                                 : Sat Oct 10 07:46:41 2026
Attached XPUs                             : 8
XPU 00000000:8A:00.0
    Minor Number                          : 7
    Memory Usage
        Total                             : 49152 MiB
XPU 00000000:8B:00.0
    Minor Number                          : 8
    Memory Usage
        Total                             : 49152 MiB
   Processes
       Process ID                        : N/A
           Used XPU Memory               : N/A
`
	got := parseKunlunxinXpuProcessMetrics(strings.NewReader(in))
	if len(got) != 0 {
		t.Fatalf("want 0 processes, got %d: %+v", len(got), got)
	}
}

func TestParseKunlunxinXpuProcessMetricsEmpty(t *testing.T) {
	for name, in := range map[string]string{
		"empty":      "",
		"headerOnly": "==============XPUSMI LOG==============\n\nTimestamp                                 : Sat Oct 10 07:46:41 2026\n",
	} {
		if got := parseKunlunxinXpuProcessMetrics(strings.NewReader(in)); len(got) != 0 {
			t.Errorf("%s: want 0 processes, got %+v", name, got)
		}
	}
}
