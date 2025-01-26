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

package hostmetrics

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type VastaitechGpuProcessMetrics struct {
	DevId       string
	PciAddr     string // Device Pci Addr
	Pid         string // Process ID
	Enc         float64
	Dec         float64
	Gfx         float64
	GfxMem      float64
	GfxMemUsage float64
}

func GetVastaitechGpuProcessMetrics() ([]VastaitechGpuProcessMetrics, error) {
	outputFile := "/tmp/vasmi_pmon.csv"
	cmd := fmt.Sprintf("/usr/bin/vasmi pmon --outputformat=csv --outputfile=%s --loop 1", outputFile)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, out)
	}
	output, err := fileutils2.FileGetContents(outputFile)
	if err != nil {
		return nil, errors.Wrapf(err, "FileGetContents %s failed", outputFile)
	}

	return parseVastaitechGpuProcessMetrics(output), nil
}

/*
LoopTimes,AIC,DevId,PCIe_Bus_Id,PID,VPID,Command,Container_Name,Enc,Dec,Gfx,Gfx_Mem,Gfx_Mem_Usage,Reserved_Mem
1,0,0,0000:18:00.0,3112054,388,android.hardware.graphics.allocator@2.0-service,va_androidxx,0,0,0,61.68MB,0.827308,0.00B
*/

func parseVastaitechGpuProcessMetrics(gpuMetricsStr string) []VastaitechGpuProcessMetrics {
	gpuProcessMetrics := make([]VastaitechGpuProcessMetrics, 0)
	lines := strings.Split(gpuMetricsStr, "\n")
	for i := 1; i < len(lines); i++ {
		segs := strings.Split(lines[i], ",")
		segLens := len(segs)
		if segLens < 14 {
			continue
		}

		devId, pciAddr, pidStr := segs[2], segs[3], segs[4]
		gfxMemUsageStr, gfxMemStr, gfxStr, decStr, encStr := segs[segLens-2], segs[segLens-3], segs[segLens-4], segs[segLens-5], segs[segLens-6]
		gfxMemUsage, err := strconv.ParseFloat(gfxMemUsageStr, 64)
		if err != nil {
			log.Errorf("failed parse gfxMemUsageStr %s: %s", gfxMemUsageStr, err)
		}
		gfxMem, err := parseSizeToFloat64(gfxMemStr)
		if err != nil {
			log.Errorf("failed parse gfxMemStr %s: %s", gfxMemStr, err)
			continue
		}
		pciAddr = strings.ReplaceAll(pciAddr, ":", "_")
		gfxMem = gfxMem / 1024.0 / 1024.0
		gfx, err := strconv.ParseFloat(gfxStr, 64)
		if err != nil {
			log.Errorf("failed parse gfxStr %s: %s", gfxStr, err)
		}
		dec, err := strconv.ParseFloat(decStr, 64)
		if err != nil {
			log.Errorf("failed parse decStr %s: %s", decStr, err)
		}
		enc, err := strconv.ParseFloat(encStr, 64)
		if err != nil {
			log.Errorf("failed parse encStr %s: %s", encStr, err)
		}
		procMetrics := VastaitechGpuProcessMetrics{
			DevId:       devId,
			PciAddr:     pciAddr,
			Pid:         pidStr,
			Gfx:         gfx,
			GfxMem:      gfxMem,
			GfxMemUsage: gfxMemUsage,
			Enc:         enc,
			Dec:         dec,
		}
		gpuProcessMetrics = append(gpuProcessMetrics, procMetrics)
	}
	return gpuProcessMetrics
}

// See: http://en.wikipedia.org/wiki/Binary_prefix
const (
	// Decimal

	KB = 1000
	MB = 1000 * KB
	GB = 1000 * MB
	TB = 1000 * GB
	PB = 1000 * TB
)

type unitMap map[string]int64

var (
	decimalMap = unitMap{"k": KB, "m": MB, "g": GB, "t": TB, "p": PB}
	sizeRegex  = regexp.MustCompile(`^(\d+(\.\d+)*) ?([kKmMgGtTpP])?[iI]?[bB]?$`)
)

// Parses the human-readable size string into the amount it represents.
func parseSizeToFloat64(sizeStr string) (float64, error) {
	matches := sizeRegex.FindStringSubmatch(sizeStr)
	if len(matches) != 4 {
		return -1, fmt.Errorf("invalid size: '%s'", sizeStr)
	}

	size, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return -1, err
	}

	unitPrefix := strings.ToLower(matches[3])
	if mul, ok := decimalMap[unitPrefix]; ok {
		size *= float64(mul)
	}

	return size, nil
}
