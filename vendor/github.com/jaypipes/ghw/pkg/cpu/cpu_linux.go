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
	"strconv"
	"strings"

	"github.com/jaypipes/ghw/pkg/context"
	"github.com/jaypipes/ghw/pkg/linuxpath"
	"github.com/jaypipes/ghw/pkg/util"
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

	// Build a set of physical processor IDs which represent the physical
	// package of the CPU
	setPhysicalIDs := make(map[int]bool)
	for _, attrs := range procAttrs {
		pid, err := strconv.Atoi(attrs["physical id"])
		if err != nil {
			continue
		}
		setPhysicalIDs[pid] = true
	}

	for pid := range setPhysicalIDs {
		p := &Processor{
			ID: pid,
		}
		// The indexes into the array of attribute maps for each logical
		// processor within the physical processor
		lps := make([]int, 0)
		for x := range procAttrs {
			lppid, err := strconv.Atoi(procAttrs[x]["physical id"])
			if err != nil {
				continue
			}
			if pid == lppid {
				lps = append(lps, x)
			}
		}
		first := procAttrs[lps[0]]
		p.Model = first["model name"]
		p.Vendor = first["vendor_id"]
		numCores, err := strconv.Atoi(first["cpu cores"])
		if err != nil {
			continue
		}
		p.NumCores = uint32(numCores)
		numThreads, err := strconv.Atoi(first["siblings"])
		if err != nil {
			continue
		}
		p.NumThreads = uint32(numThreads)

		// The flags field is a space-separated list of CPU capabilities
		p.Capabilities = strings.Split(first["flags"], " ")

		cores := make([]*ProcessorCore, 0)
		for _, lpidx := range lps {
			lpid, err := strconv.Atoi(procAttrs[lpidx]["processor"])
			if err != nil {
				continue
			}
			coreID, err := strconv.Atoi(procAttrs[lpidx]["core id"])
			if err != nil {
				continue
			}
			var core *ProcessorCore
			for _, c := range cores {
				if c.ID == coreID {
					c.LogicalProcessors = append(
						c.LogicalProcessors,
						lpid,
					)
					c.NumThreads = uint32(len(c.LogicalProcessors))
					core = c
				}
			}
			if core == nil {
				coreLps := make([]int, 1)
				coreLps[0] = lpid
				core = &ProcessorCore{
					ID:                coreID,
					Index:             len(cores),
					NumThreads:        1,
					LogicalProcessors: coreLps,
				}
				cores = append(cores, core)
			}
		}
		p.Cores = cores
		procs = append(procs, p)
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
