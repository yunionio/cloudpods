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
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

/*
# gpu        pid  type    sm   mem   enc   dec   command
# Idx          #   C/G     %     %     %     %   name
    0          -     -     -     -     -     -   -
    0          -     -     -     -     -     -   -
*/

type IluvatarGpuProcessMetrics struct {
	Pid     string // Process ID
	MemUtil float64
	Mem     float64
	Enc     float64
	Dec     float64
	SmUtil  float64
	Idx     int
}

func GetIluvatarGpuProcessMetrics(hostinfo IHostInfo) ([]IluvatarGpuProcessMetrics, error) {
	outputFile := "/tmp/ixsmi_pmon.out"
	cmd := fmt.Sprintf("/usr/local/bin/ixsmi pmon -f %s -c 1", outputFile)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, out)
	}
	output, err := fileutils2.FileGetContents(outputFile)
	if err != nil {
		return nil, errors.Wrapf(err, "FileGetContents %s failed", outputFile)
	}
	return parseIluvatarGpuProcessMetrics(output, hostinfo), nil
}

func parseIluvatarGpuProcessMetrics(gpuMetricsStr string, hostinfo IHostInfo) []IluvatarGpuProcessMetrics {
	gpuProcessMetrics := make([]IluvatarGpuProcessMetrics, 0)
	lines := strings.Split(gpuMetricsStr, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "#") {
			continue
		}
		segs := strings.Split(lines[i], ",")
		segLens := len(segs)
		if segLens < 8 {
			continue
		}
		idxStr, pid, smUtilStr, memPrecentStr, encStr, decStr := segs[0], segs[1], segs[3], segs[4], segs[5], segs[6]
		idx, err := strconv.ParseInt(idxStr, 10, 64)
		if err != nil {
			log.Errorf("Parse IluvatarGpuProcessMetrics idxStr %s failed", idxStr)
			continue
		}
		if pid == "-" {
			continue
		}
		smUtil, err := strconv.ParseFloat(smUtilStr, 64)
		if err != nil {
			log.Errorf("Parse SmUtil %s failed", smUtilStr)
		}
		enc, err := strconv.ParseFloat(encStr, 64)
		if err != nil {
			log.Errorf("Parse Enc %s failed", encStr)
		}
		dec, err := strconv.ParseFloat(decStr, 64)
		if err != nil {
			log.Errorf("Parse Dec %s failed", decStr)
		}
		memPrecent, err := strconv.ParseFloat(memPrecentStr, 64)
		if err != nil {
			log.Errorf("Parse MemPrecent %s failed", memPrecentStr)
		}
		memTotal := hostinfo.GetContainerIluvatarGpuMemSizeByIndex(int(idx))
		mem := float64(memTotal) * memPrecent
		procMetrics := IluvatarGpuProcessMetrics{
			Pid:     pid,
			MemUtil: memPrecent,
			Mem:     mem,
			Enc:     enc,
			Dec:     dec,
			SmUtil:  smUtil,
			Idx:     int(idx),
		}
		gpuProcessMetrics = append(gpuProcessMetrics, procMetrics)
	}
	return gpuProcessMetrics
}
