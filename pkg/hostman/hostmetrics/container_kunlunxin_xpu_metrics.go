package hostmetrics

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

type KunlunxinXpuProcessMetrics struct {
	Pid     string // Process ID
	PciAddr string
	MemUtil float64
	Mem     float64
	Idx     int
}

func GetKunlunxinXpuProcessMetrics() ([]KunlunxinXpuProcessMetrics, error) {
	cmd := "xpu-smi -q"
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, output)
	}

	return parseKunlunxinXpuProcessMetrics(bytes.NewReader(output)), nil
}

func util(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return used / total * 100
}

func parseNum(s string) float64 {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0 // N/A 等
	}
	return v
}

type devCtx struct {
	pciAddr  string
	idx      int
	memTotal float64
}

type procCtx struct {
	pid string
	mem float64
}

var deviceHeaderRe = regexp.MustCompile(`^XPU\s+([0-9a-fA-F]{4,8}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F])$`)

var kvRe = regexp.MustCompile(`^([^:]+?)\s*:\s*(.*)$`)

func parseKunlunxinXpuProcessMetrics(r io.Reader) []KunlunxinXpuProcessMetrics {
	var procs []KunlunxinXpuProcessMetrics

	var (
		curDev  *devCtx
		curProc *procCtx
		section string
	)

	emitProc := func() {
		if curProc == nil || curDev == nil {
			curProc = nil
			return
		}
		if curProc.pid != "" && curProc.pid != "N/A" {
			procs = append(procs, KunlunxinXpuProcessMetrics{
				Pid:     curProc.pid,
				PciAddr: curDev.pciAddr,
				Idx:     curDev.idx,
				Mem:     curProc.mem,
				MemUtil: util(curProc.mem, curDev.memTotal),
			})
		}
		curProc = nil
	}

	flushDev := func() {
		emitProc()
		curDev = nil
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		trimmed := strings.TrimSpace(strings.TrimRight(sc.Text(), " \t\r\n"))
		if trimmed == "" {
			continue
		}

		if m := deviceHeaderRe.FindStringSubmatch(trimmed); m != nil {
			flushDev()
			pciaddr := strings.ToLower(strings.TrimSpace(m[1]))
			curDev = &devCtx{pciAddr: pciaddr}
			section = ""
			continue
		}
		if curDev == nil {
			continue
		}

		if !strings.Contains(trimmed, ":") {
			section = trimmed
			continue
		}

		m := kvRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		key, val := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])

		switch key {
		case "Minor Number":
			curDev.idx = int(parseNum(val))
		case "Process ID":
			emitProc()
			curProc = &procCtx{pid: val}
		case "Used XPU Memory":
			if curProc != nil {
				curProc.mem = parseNum(val)
			}
		}

		if section == "Memory Usage" {
			switch key {
			case "Total":
				curDev.memTotal = parseNum(val)
			}
		}
	}
	flushDev()

	return procs
}
