// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package linuxpath

import (
	"fmt"
	"path/filepath"

	"github.com/jaypipes/ghw/pkg/context"
)

type Paths struct {
	VarLog                 string
	ProcMeminfo            string
	ProcCpuinfo            string
	SysKernelMMHugepages   string
	EtcMtab                string
	SysBlock               string
	SysDevicesSystemNode   string
	SysDevicesSystemMemory string
	SysBusPciDevices       string
	SysClassDRM            string
	SysClassDMI            string
	SysClassNet            string
	RunUdevData            string
}

// New returns a new Paths struct containing filepath fields relative to the
// supplied Context
func New(ctx *context.Context) *Paths {
	return &Paths{
		VarLog:                 filepath.Join(ctx.Chroot, "var", "log"),
		ProcMeminfo:            filepath.Join(ctx.Chroot, "proc", "meminfo"),
		ProcCpuinfo:            filepath.Join(ctx.Chroot, "proc", "cpuinfo"),
		SysKernelMMHugepages:   filepath.Join(ctx.Chroot, "sys", "kernel", "mm", "hugepages"),
		EtcMtab:                filepath.Join(ctx.Chroot, "etc", "mtab"),
		SysBlock:               filepath.Join(ctx.Chroot, "sys", "block"),
		SysDevicesSystemNode:   filepath.Join(ctx.Chroot, "sys", "devices", "system", "node"),
		SysDevicesSystemMemory: filepath.Join(ctx.Chroot, "sys", "devices", "system", "memory"),
		SysBusPciDevices:       filepath.Join(ctx.Chroot, "sys", "bus", "pci", "devices"),
		SysClassDRM:            filepath.Join(ctx.Chroot, "sys", "class", "drm"),
		SysClassDMI:            filepath.Join(ctx.Chroot, "sys", "class", "dmi"),
		SysClassNet:            filepath.Join(ctx.Chroot, "sys", "class", "net"),
		RunUdevData:            filepath.Join(ctx.Chroot, "run", "udev", "data"),
	}
}

func (p *Paths) NodeCPU(nodeID int, lpID int) string {
	return filepath.Join(
		p.SysDevicesSystemNode,
		fmt.Sprintf("node%d", nodeID),
		fmt.Sprintf("cpu%d", lpID),
	)
}

func (p *Paths) NodeCPUCache(nodeID int, lpID int) string {
	return filepath.Join(
		p.NodeCPU(nodeID, lpID),
		"cache",
	)
}

func (p *Paths) NodeCPUCacheIndex(nodeID int, lpID int, cacheIndex int) string {
	return filepath.Join(
		p.NodeCPUCache(nodeID, lpID),
		fmt.Sprintf("index%d", cacheIndex),
	)
}
