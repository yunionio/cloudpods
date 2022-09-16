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

package nodeid

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/log"
)

func execCmd(cmd *exec.Cmd) (string, error) {
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func stringToLines(s string) (lines []string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()
	return
}

func getLinuxMacAddress() (string, error) {
	path := "/sys/class/net/"
	fs, e := os.ReadDir(path)
	if e != nil {
		return "", e
	}

	macs := []string{}
	for _, f := range fs {
		link, e := os.Readlink(path + f.Name())
		// 过滤掉虚拟网卡
		if e != nil || !strings.Contains(link, "/pci") {
			continue
		}
		u, e := os.ReadFile(path + f.Name() + "/address")
		if e == nil {
			us := strings.TrimSpace(string(u))
			isNew := true
			for _, mac := range macs {
				if us == "00:00:00:00:00:00" || us == mac {
					isNew = false
				}
			}

			if isNew {
				macs = append(macs, us)
			}
		}
	}

	sort.SliceStable(macs, func(i, j int) bool {
		return macs[i] < macs[j]
	})

	if len(macs) < 1 {
		return "", fmt.Errorf("no available mac")
	}

	log.Debugf("macs info %s", macs)
	mac := strings.Replace(macs[0], ":", "", -1)
	if len(mac) != 12 {
		return "", fmt.Errorf("mac length error")
	}

	return mac, nil
}

func getLinuxCpuInfo() (string, error) {
	u, e := os.ReadFile("/proc/cpuinfo")
	if e != nil {
		return "", e
	}

	lines, e := stringToLines(string(u))
	filted := []string{}
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if strings.Contains(l, "cache size") || strings.Contains(l, "processor") {
			v := strings.Split(l, ":")[1]
			vn := strings.Replace(v, " ", "", -1)

			isNew := true
			for _, fl := range filted {
				if fl == vn {
					isNew = false
				}
			}

			if isNew {
				filted = append(filted, vn)
			}
		}
	}

	re := regexp.MustCompile(`([0-9]+)?.*`)
	sort.SliceStable(filted, func(i, j int) bool {
		a := re.FindStringSubmatch(filted[i])[1]
		b := re.FindStringSubmatch(filted[j])[1]
		if a == "" {
			if b == "" {
				return filted[i] < filted[j]
			} else if strings.HasPrefix(b, "0") {
				return false
			} else {
				return true
			}
		} else {
			if b == "" {
				if strings.HasPrefix(a, "0") {
					return true
				} else {
					return false
				}
			} else {
				if a == b {
					return len(filted[i]) < len(filted[j])
				} else {
					ia, _ := strconv.Atoi(a)
					ib, _ := strconv.Atoi(b)
					return ia < ib
				}
			}
		}
	})

	if len(filted) == 0 {
		return "", fmt.Errorf("no available cpu info")
	}
	log.Debugf("cpu info %s", filted)
	h := md5.New()
	h.Write([]byte(strings.Join(filted, "\n") + "\n"))
	cpumd5 := hex.EncodeToString(h.Sum(nil))
	return cpumd5, nil
}

func getLinux() ([]string, error) {
	ret := []string{}
	funcs := make([]interface{}, 0)
	funcs = append(funcs, getLinuxMacAddress)
	funcs = append(funcs, getLinuxCpuInfo)

	for _, f := range funcs {
		fi := reflect.ValueOf(f)
		s := fi.Call(nil)
		if e := s[1].Interface(); e == nil {
			ret = append(ret, s[0].Interface().(string))
		}
	}

	return ret, nil
}

func GetNodeId() ([]byte, error) {
	var f func() ([]string, error)
	if runtime.GOOS == "linux" {
		f = getLinux
	} else {
		return nil, fmt.Errorf("Unsupported OS")
	}

	ret, e := f()
	if e != nil || len(ret) < 2 {
		log.Debugf("service info %s", ret)
		return nil, fmt.Errorf("Fail to generate Node ID")
	}

	sn := md5.Sum([]byte(ret[0] + ret[1]))
	return sn[0:], nil
}
