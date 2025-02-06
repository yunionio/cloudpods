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
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type CphAmdGpuProcessMetrics struct {
	Pid     string // Process ID
	DevId   string
	Mem     float64 // Memory Utilization
	MemUtil float64
}

/*
pid  2088269 command allocator@2.0-s:
        0x00000001:         4096 byte  GTT CPU_ACCESS_REQUIRED
        0x00000002:      2097152 byte  GTT CPU_ACCESS_REQUIRED
        0x00000003:      2097152 byte VRAM VRAM_CLEARED
        0x00000004:      2097152 byte VRAM NO_CPU_ACCESS VRAM_CLEARED
        0x00000006:      2097152 byte  GTT CPU_ACCESS_REQUIRED VRAM_CLEARED
        0x00000007:      2097152 byte  GTT CPU_ACCESS_REQUIRED VRAM_CLEARED
*/

func GetCphAmdGpuProcessMetrics() ([]CphAmdGpuProcessMetrics, error) {
	debugDriDir := "/sys/kernel/debug/dri"
	entrys, err := os.ReadDir(debugDriDir)
	if err != nil {
		return nil, errors.Wrap(err, "os.ReadDir")
	}

	res := make([]CphAmdGpuProcessMetrics, 0)
	for i := range entrys {
		if entrys[i].IsDir() {
			fpath := path.Join(debugDriDir, entrys[i].Name(), "amdgpu_gem_info")
			if fileutils2.Exists(fpath) {
				content, err := fileutils2.FileGetContents(fpath)
				if err != nil {
					log.Errorf("failed FileGetContents %s: %s", fpath, err)
					continue
				}
				vramInfoPath := path.Join(debugDriDir, entrys[i].Name(), "amdgpu_vram_mm")
				memTotalSize, err := getVramTotalSizeMb(vramInfoPath)
				if err != nil {
					log.Errorf("failed getVramTotalSizeMb %s", err)
				}
				metrics := parseCphAmdGpuGemInfo(content, entrys[i].Name(), memTotalSize)
				if len(metrics) > 0 {
					res = append(res, metrics...)
				}
			}
		}
	}
	return res, nil
}

var pagesRe = regexp.MustCompile(`man size:(\d+) pages`)

// man size:8384512 pages, ram usage:3745MB, vis usage:241MB
func getVramTotalSizeMb(vramInfoPath string) (int, error) {
	if !fileutils2.Exists(vramInfoPath) {
		return 0, nil
	}
	out, err := procutils.NewCommand("tail", "-n", "1", vramInfoPath).Output()
	if err != nil {
		return 0, errors.Wrapf(err, "tail -n 1 %s", vramInfoPath)
	}
	str := strings.TrimSpace(string(out))
	matches := pagesRe.FindStringSubmatch(str)
	if len(matches) > 1 {
		pages, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, errors.Wrapf(err, " failed parse pages count %s", matches[0])
		}
		return pages * 4 * 1024 / 1024 / 1024, nil
	}
	return 0, errors.Errorf("failed parse pages count: %s", str)
}

func parseCphAmdGpuGemInfo(content string, devId string, memTotalSizeMB int) []CphAmdGpuProcessMetrics {
	res := make([]CphAmdGpuProcessMetrics, 0)
	lines := strings.Split(content, "\n")
	var i, length = 0, len(lines)
	for i < length {
		line := strings.TrimSpace(lines[i])
		segs := strings.Fields(line)
		if len(segs) < 2 {
			i++
			continue
		}
		if segs[0] != "pid" {
			i++
			continue
		}
		pid := segs[1]
		var vramTotal int64 = 0
		j := i + 1
		for j < length {
			line := strings.TrimSpace(lines[j])
			if len(line) == 0 {
				break
			}
			segs := strings.Fields(line)
			if len(segs) < 4 {
				log.Errorf("unknown output line %s", line)
				break
			}
			if segs[0] == "pid" {
				break
			}
			memUsedStr, memType := segs[1], segs[3]
			if memType == "VRAM" {
				memUsed, err := strconv.ParseInt(memUsedStr, 10, 64)
				if err != nil {
					log.Errorf("failed parse memused %s %s: %s", line, memUsedStr, err)
					break
				}
				vramTotal += memUsed
			}
			j++
		}
		memSize := float64(vramTotal) / 1024.0 / 1024.0
		res = append(res, CphAmdGpuProcessMetrics{
			Pid:     pid,
			DevId:   devId,
			Mem:     memSize,
			MemUtil: memSize / float64(memTotalSizeMB) * 100.0,
		})
		i = j
	}

	return res
}
