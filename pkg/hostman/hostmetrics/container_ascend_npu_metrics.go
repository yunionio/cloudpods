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

// AscendNpuProcessMetrics 单个昇腾 NPU 进程指标。
type AscendNpuProcessMetrics struct {
	Pid     string
	PciAddr string
	MemUtil float64
	Mem     float64
	Idx     int
}

type ascendDevCtx struct {
	pciAddr  string
	memTotal float64
}

type ascendRawProc struct {
	idx int
	pid string
	mem float64
}

var (
	ascendDevIdxRe = regexp.MustCompile(`^(\d+)\s+\S+`)
	ascendBusIdRe  = regexp.MustCompile(`^[0-9a-fA-F]{4,8}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.\d+$`)
	ascendPairRe   = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*/\s*(\d+(?:\.\d+)?)\s*$`)
)

const (
	ascendDevHdr     = "Bus-Id"
	ascendProcHdr    = "Process id"
	ascendNoProcLine = "No running proc"
)

func GetContainerAscendNpuMetrics() ([]AscendNpuProcessMetrics, error) {
	cmd := "npu-smi info"
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, output)
	}
	return parseContainerAscendNpuMetrics(bytes.NewReader(output)), nil
}

func parseContainerAscendNpuMetrics(r io.Reader) []AscendNpuProcessMetrics {
	devs := make(map[int]*ascendDevCtx)
	var raw []ascendRawProc

	var (
		inProcTable bool
		curIdx      int
	)

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.HasPrefix(line, "|") {
			continue
		}

		if strings.Contains(line, ascendProcHdr) {
			inProcTable = true
			continue
		}
		if strings.Contains(line, ascendDevHdr) {
			inProcTable = false
			continue
		}
		if strings.Contains(line, ascendNoProcLine) {
			continue
		}

		cells := splitAscendRow(line)
		if len(cells) == 0 {
			continue
		}

		if inProcTable {
			// | 0   0 | 1011595 | python3 | 2163 | NA |
			if len(cells) < 4 {
				continue
			}
			idx, ok := ascendAtoi(cells[0])
			if !ok {
				continue
			}
			pid := cells[1]
			if pid == "" || pid == "NA" {
				continue
			}
			raw = append(raw, ascendRawProc{idx: idx, pid: pid, mem: parseNum(cells[3])})
			continue
		}

		if m := ascendDevIdxRe.FindStringSubmatch(cells[0]); m != nil {
			curIdx = int(parseNum(m[1]))
			if devs[curIdx] == nil {
				devs[curIdx] = &ascendDevCtx{}
			}
			continue
		}

		if len(cells) >= 2 && ascendBusIdRe.MatchString(cells[1]) {
			dev := devs[curIdx]
			if dev == nil {
				dev = &ascendDevCtx{}
				devs[curIdx] = dev
			}
			dev.pciAddr = cells[1]
			if m := ascendPairRe.FindStringSubmatch(cells[len(cells)-1]); m != nil {
				dev.memTotal = parseNum(m[2])
			}
		}
	}

	procs := make([]AscendNpuProcessMetrics, 0, len(raw))
	for _, p := range raw {
		m := AscendNpuProcessMetrics{Pid: p.pid, Idx: p.idx, Mem: p.mem}
		if dev, ok := devs[p.idx]; ok {
			m.PciAddr = dev.pciAddr
			m.MemUtil = util(p.mem, dev.memTotal)
		}
		procs = append(procs, m)
	}
	return procs
}

func splitAscendRow(line string) []string {
	cells := make([]string, 0, 8)
	for _, p := range strings.Split(line, "|") {
		if p = strings.TrimSpace(p); p != "" {
			cells = append(cells, p)
		}
	}
	return cells
}

func ascendAtoi(s string) (int, bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, false
	}
	return n, true
}
