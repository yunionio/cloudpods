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
	"strconv"
	"strings"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/fileutils2"

	"yunion.io/x/pkg/errors"
)

type CphAmdGpuProcessMetrics struct {
	Pid   string // Process ID
	DevId string
	Mem   float64 // Memory Utilization
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
				metrics := parseCphAmdGpuGemInfo(content, entrys[i].Name())
				if len(metrics) > 0 {
					res = append(res, metrics...)
				}
			}
		}
	}
	return res, nil
}

func parseCphAmdGpuGemInfo(content string, devId string) []CphAmdGpuProcessMetrics {
	res := make([]CphAmdGpuProcessMetrics, 0)
	lines := strings.Split(content, "\n")
	var i, length = 0, len(lines)
	for i < length {
		line := strings.TrimSpace(lines[i])
		segs := strings.Split(line, " ")
		if segs[0] != "pid" {
			i++
			continue
		}
		if len(segs) < 2 {
			i++
			continue
		}
		pid := segs[1]
		var vramTotal int64 = 0
		j := i + 1
		for j < length {
			line := strings.TrimSpace(lines[j])
			segs := strings.Split(line, " ")
			if segs[0] == "pid" {
				break
			}
			if len(segs) < 4 {
				log.Errorf("unknown output line %s", line)
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
		res = append(res, CphAmdGpuProcessMetrics{
			Pid:   pid,
			DevId: devId,
			Mem:   float64(vramTotal) / 1024.0 / 1024.0,
		})
		i = j
	}

	return res
}
