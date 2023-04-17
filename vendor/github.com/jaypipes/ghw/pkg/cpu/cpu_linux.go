// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package cpu

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw/pkg/context"
	"github.com/jaypipes/ghw/pkg/linuxpath"
	"github.com/jaypipes/ghw/pkg/util"
)

var (
	regexForCpulCore = regexp.MustCompile("^cpu([0-9]+)$")
)

func (i *Info) load() error {
	i.Processors = processorsGet(i.ctx)
	var totCores uint32
	var totThreads uint32
	for _, p := range i.Processors {
		totCores += p.NumCores
		totThreads += p.NumThreads
	}
	i.TotalCores = totCores
	i.TotalThreads = totThreads
	return nil
}

func ProcByID(procs []*Processor, id int) *Processor {
	for pid := range procs {
		if procs[pid].ID == id {
			return procs[pid]
		}
	}
	return nil
}

func CoreByID(cores []*ProcessorCore, id int) *ProcessorCore {
	for cid := range cores {
		if cores[cid].Index == id {
			return cores[cid]
		}
	}
	return nil
}

func processorsGet(ctx *context.Context) []*Processor {
	procs := make([]*Processor, 0)
	paths := linuxpath.New(ctx)

	r, err := os.Open(paths.ProcCpuinfo)
	if err != nil {
		return nil
	}
	defer util.SafeClose(r)

	// An array of maps of attributes describing the logical processor
	procAttrs := make([]map[string]string, 0)
	curProcAttrs := make(map[string]string)

	// Parse /proc/cpuinfo
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			// Output of /proc/cpuinfo has a blank newline to separate logical
			// processors, so here we collect up all the attributes we've
			// collected for this logical processor block
			procAttrs = append(procAttrs, curProcAttrs)
			// Reset the current set of processor attributes...
			curProcAttrs = make(map[string]string)
			continue
		}
		parts := strings.Split(line, ":")
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		curProcAttrs[key] = value
	}

	// Iterate on /sys/devices/system/cpu/cpuN, not on /proc/cpuinfo
	Entries, err := ioutil.ReadDir(paths.SysDevicesSystemCPU)
	if err != nil {
		return nil
	}
	for _, lcore := range Entries {
		matches := regexForCpulCore.FindStringSubmatch(lcore.Name())
		if len(matches) < 2 {
			continue
		}

		lcoreID, error := strconv.Atoi(matches[1])
		if error != nil {
			continue
		}

		// Fetch CPU ID
		physIdPath := filepath.Join(paths.SysDevicesSystemCPU, fmt.Sprintf("cpu%d", lcoreID), "topology", "physical_package_id")
		cpuID := util.SafeIntFromFile(ctx, physIdPath)

		proc := ProcByID(procs, cpuID)
		if proc == nil {
			proc = &Processor{ID: cpuID}
			// Assumes /proc/cpuinfo is in order of logical cpu id, then
			// procAttrs[lcoreID] describes logical cpu `lcoreID`.
			// Once got a more robust way of fetching the following info,
			// can we drop /proc/cpuinfo.
			if len(procAttrs[lcoreID]["flags"]) != 0 { // x86
				proc.Capabilities = strings.Split(procAttrs[lcoreID]["flags"], " ")
			} else if len(procAttrs[lcoreID]["Features"]) != 0 { // ARM64
				proc.Capabilities = strings.Split(procAttrs[lcoreID]["Features"], " ")
			}
			if len(procAttrs[lcoreID]["model name"]) != 0 {
				proc.Model = procAttrs[lcoreID]["model name"]
			} else if len(procAttrs[lcoreID]["uarch"]) != 0 { // SiFive
				proc.Model = procAttrs[lcoreID]["uarch"]
			}
			if len(procAttrs[lcoreID]["vendor_id"]) != 0 {
				proc.Vendor = procAttrs[lcoreID]["vendor_id"]
			} else if len(procAttrs[lcoreID]["isa"]) != 0 { // RISCV64
				proc.Vendor = procAttrs[lcoreID]["isa"]
			}
			procs = append(procs, proc)
		}

		// Fetch Core ID
		coreIdPath := filepath.Join(paths.SysDevicesSystemCPU, fmt.Sprintf("cpu%d", lcoreID), "topology", "core_id")
		coreID := util.SafeIntFromFile(ctx, coreIdPath)
		core := CoreByID(proc.Cores, coreID)
		if core == nil {
			core = &ProcessorCore{Index: coreID, NumThreads: 1}
			proc.Cores = append(proc.Cores, core)
			proc.NumCores += 1
		} else {
			core.NumThreads += 1
		}
		proc.NumThreads += 1
		core.LogicalProcessors = append(core.LogicalProcessors, lcoreID)
	}
	return procs
}

func CoresForNode(ctx *context.Context, nodeID int) ([]*ProcessorCore, error) {
	// The /sys/devices/system/node/nodeX directory contains a subdirectory
	// called 'cpuX' for each logical processor assigned to the node. Each of
	// those subdirectories contains a topology subdirectory which has a
	// core_id file that indicates the 0-based identifier of the physical core
	// the logical processor (hardware thread) is on.
	paths := linuxpath.New(ctx)
	path := filepath.Join(
		paths.SysDevicesSystemNode,
		fmt.Sprintf("node%d", nodeID),
	)
	cores := make([]*ProcessorCore, 0)

	findCoreByID := func(coreID int) *ProcessorCore {
		for _, c := range cores {
			if c.ID == coreID {
				return c
			}
		}

		c := &ProcessorCore{
			ID:                coreID,
			Index:             len(cores),
			LogicalProcessors: make([]int, 0),
		}
		cores = append(cores, c)
		return c
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		filename := file.Name()
		if !strings.HasPrefix(filename, "cpu") {
			continue
		}
		if filename == "cpumap" || filename == "cpulist" {
			// There are two files in the node directory that start with 'cpu'
			// but are not subdirectories ('cpulist' and 'cpumap'). Ignore
			// these files.
			continue
		}
		// Grab the logical processor ID by cutting the integer from the
		// /sys/devices/system/node/nodeX/cpuX filename
		cpuPath := filepath.Join(path, filename)
		procID, err := strconv.Atoi(filename[3:])
		if err != nil {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"failed to determine procID from %s. Expected integer after 3rd char.",
				filename,
			)
			continue
		}
		coreIDPath := filepath.Join(cpuPath, "topology", "core_id")
		coreID := util.SafeIntFromFile(ctx, coreIDPath)
		core := findCoreByID(coreID)
		core.LogicalProcessors = append(
			core.LogicalProcessors,
			procID,
		)
	}

	for _, c := range cores {
		c.NumThreads = uint32(len(c.LogicalProcessors))
	}

	return cores, nil
}
