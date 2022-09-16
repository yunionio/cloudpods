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

package nbd

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/hostman/guestfs/kvmpart"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	PATH_TYPE_UNKNOWN = 0
	LVM_PATH          = 1
	NON_LVM_PATH      = 2
)

type SLVMImageConnectUniqueToolSet struct {
	lvms    map[string]*sync.Mutex
	nonLvms map[string]struct{}
	lock    *sync.Mutex
}

func NewLVMImageConnectUniqueToolSet() *SLVMImageConnectUniqueToolSet {
	return &SLVMImageConnectUniqueToolSet{
		lvms:    make(map[string]*sync.Mutex),
		nonLvms: make(map[string]struct{}),
		lock:    new(sync.Mutex),
	}
}

func (s *SLVMImageConnectUniqueToolSet) CacheNonLvmImagePath(imagePath string) {
	s.lock.Lock()
	s.nonLvms[imagePath] = struct{}{}
	s.lock.Unlock()
}

func (s *SLVMImageConnectUniqueToolSet) GetPathType(imagePath string) int {
	if _, ok := s.nonLvms[imagePath]; ok {
		return NON_LVM_PATH
	}
	if _, ok := s.lvms[imagePath]; ok {
		return LVM_PATH
	}
	return PATH_TYPE_UNKNOWN
}

func (s *SLVMImageConnectUniqueToolSet) Release(imagePath string) {
	if _, ok := s.lvms[imagePath]; ok {
		s.lvms[imagePath].Unlock()
	}
}

func (s *SLVMImageConnectUniqueToolSet) Acquire(imagePath string) {
	s.lock.Lock()
	if _, ok := s.lvms[imagePath]; !ok {
		s.lvms[imagePath] = new(sync.Mutex)
	}
	s.lock.Unlock()

	s.lvms[imagePath].Lock()
}

type SKVMGuestLVMPartition struct {
	partDev      string
	originVgname string
	vgname       string

	// if need to modify the name when putting down
	needChangeName bool
	vgid           string
}

func findVgname(partDev string) string {
	output, err := procutils.NewCommand("pvscan").Output()
	if err != nil {
		log.Errorf("%s", output)
		return ""
	}

	re := regexp.MustCompile(`\s+`)
	for _, line := range strings.Split(string(output), "\n") {
		data := re.Split(strings.TrimSpace(line), -1)
		if len(data) >= 4 && data[1] == partDev {
			return data[3]
		}
	}
	return ""
}

type SVG struct {
	Id   string
	Name string
}

func findVg(partDev string) (SVG, error) {
	command := procutils.NewCommand("vgs", "-v", "--devices", partDev)
	output, err := command.Output()
	if err != nil {
		return SVG{}, errors.Wrapf(err, "unable to exec command %q", command)
	}
	log.Debugf("command: %s\noutptu: %s", command, output)

	outputStr := string(output)
	r := regexp.MustCompile("WARNING: Device mismatch detected for .* which is accessing .* instead of .*.")
	ret := r.FindStringSubmatch(outputStr)
	if len(ret) > 0 {
		return SVG{}, fmt.Errorf("VG conflicts with the VG UUID of the host")
	}
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) <= 1 {
		return SVG{}, fmt.Errorf("unable to find vg, output is %q", output)
	}
	data := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(lines[len(lines)-1]), -1)
	if len(data) < 1 || len(data) < 9 {
		return SVG{}, fmt.Errorf("The output is not as expected: %q", output)
	}
	return SVG{data[8], data[0]}, nil
}

func NewKVMGuestLVMPartition(partDev string, vg SVG) *SKVMGuestLVMPartition {
	return &SKVMGuestLVMPartition{
		partDev:      partDev,
		originVgname: vg.Name,
		vgname:       uuidWithoutLine(),
		vgid:         vg.Id,
	}
}

func uuidWithoutLine() string {
	uuid := stringutils.UUID4()
	return strings.ReplaceAll(uuid, "-", "")
}

func (p *SKVMGuestLVMPartition) SetupDevice() bool {
	if len(p.vgid) == 0 {
		return false
	}
	if p.vgRename(p.vgid, p.vgname) {
		p.needChangeName = true
	} else {
		p.vgname = p.originVgname
		p.needChangeName = false
	}
	if p.vgActivate(true) {
		return true
	}
	return false
}

func (p *SKVMGuestLVMPartition) FindPartitions() []*kvmpart.SKVMGuestDiskPartition {
	parts := []*kvmpart.SKVMGuestDiskPartition{}
	// try /dev/{vgname}/{lvname}
	files, err := os.ReadDir("/dev/" + p.vgname)
	if err == nil {
		for _, f := range files {
			partPath := fmt.Sprintf("/dev/%s/%s", p.vgname, f.Name())
			part := kvmpart.NewKVMGuestDiskPartition(partPath, p.partDev, true)
			parts = append(parts, part)
		}
		return parts
	}
	if !os.IsNotExist(err) {
		log.Errorf("unable to readir /dev/%s: %v", p.vgname, err)
		return nil
	}
	log.Debugf("unable to read dir '/dev/%s': %v", p.vgname, err)
	// try /dev/mapper/{vgname}-{lvname}
	lvs, err := p.lvs()
	if err != nil {
		log.Errorf("unable to list lvs: %v", err)
		return nil
	}
	for _, lvname := range lvs {
		path := fmt.Sprintf("/dev/mapper/%s-%s", p.vgname, lvname)
		_, err := os.Lstat(path)
		if err == nil {
			part := kvmpart.NewKVMGuestDiskPartition(path, p.partDev, true)
			parts = append(parts, part)
			continue
		}
		log.Errorf("unable to ls %s: %v", path, err)
	}
	return parts
}

var gexp *regexp.Regexp = regexp.MustCompile(`\s+`)

func (p *SKVMGuestLVMPartition) lvs() ([]string, error) {
	command := procutils.NewCommand("lvs", p.vgname)
	output, err := command.Output()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to exec %q command", command)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil, errors.Wrapf(err, "command: %q, no output", command)
	}
	tableHeader := gexp.Split(strings.TrimSpace(lines[0]), -1)
	if tableHeader[0] != "LV" {
		return nil, fmt.Errorf("command: %q, unexpected output: %q", command, output)
	}
	lvname := make([]string, 0, len(lines)-1)
	for i := 1; i < len(lines); i++ {
		tableLine := gexp.Split(strings.TrimSpace(lines[i]), -1)
		lvname = append(lvname, tableLine[0])
	}
	return lvname, nil
}

func (p *SKVMGuestLVMPartition) PutdownDevice() bool {
	if !p.vgActivate(false) {
		return false
	}
	if len(p.originVgname) == 0 || !p.needChangeName {
		return true
	}
	if p.vgRename(p.vgname, p.originVgname) {
		return true
	}
	return false
}

func (p *SKVMGuestLVMPartition) vgActivate(activate bool) bool {
	param := "-an"
	if activate {
		param = "-ay"
	}
	output, err := procutils.NewCommand("vgchange", param, p.vgname).Output()
	if err != nil {
		log.Errorf("%s", output)
		return false
	}
	if out, err := procutils.NewCommand("vgchange", "--refresh").Output(); err != nil {
		log.Errorf("vgchange refresh failed: %s, %s", out, err)
	}
	return true
}

func (p *SKVMGuestLVMPartition) vgRename(oldname, newname string) bool {
	command := procutils.NewCommand("vgrename", "--devices", p.partDev, oldname, newname)
	output, err := command.Output()
	if err != nil {
		log.Errorf("unable to exec command: %q, error: %v, output: %q", command, err, output)
		return false
	}
	log.Infof("VG rename succ from %s to %s", oldname, newname)
	return true
}
