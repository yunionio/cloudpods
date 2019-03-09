package storageman

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/pkg/util/stringutils"
)

const (
	PATH_NOT_FOUND = 0
	LVM_PATH       = 1
	NON_LVM_PATH   = 2
)

type SLVMImageConnectUniqueTool struct {
	cond     *sync.Cond
	refCount int
}

func NewLVMImageConnectUniqueTool() *SLVMImageConnectUniqueTool {
	return &SLVMImageConnectUniqueTool{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (t *SLVMImageConnectUniqueTool) AddRef() {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()
	t.refCount++
}

func (t *SLVMImageConnectUniqueTool) Wait() {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()
	t.cond.Wait()
}

func (t *SLVMImageConnectUniqueTool) Signal() {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()
	t.cond.Signal()
}

func (t *SLVMImageConnectUniqueTool) CanBeDestoryed() bool {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()
	return t.refCount == 0
}

type SLVMImageConnectUniqueToolSet struct {
	lvms               map[string]*SLVMImageConnectUniqueTool
	nonLVMImagePathSet map[string]struct{}
	lock               *sync.Mutex
}

func NewLVMImageConnectUniqueToolSet() *SLVMImageConnectUniqueToolSet {
	return &SLVMImageConnectUniqueToolSet{
		lvms:               make(map[string]*SLVMImageConnectUniqueTool),
		nonLVMImagePathSet: make(map[string]struct{}),
		lock:               new(sync.Mutex),
	}
}

func (s *SLVMImageConnectUniqueToolSet) Add(imagePath string) {
	s.lock.Lock()
	if _, ok := s.lvms[imagePath]; !ok {
		s.lvms[imagePath] = NewLVMImageConnectUniqueTool()
	}
	s.lock.Unlock()

	s.lvms[imagePath].AddRef()
}

func (s *SLVMImageConnectUniqueToolSet) CacheNonLvmImagePath(imagePath string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.nonLVMImagePathSet[imagePath] = struct{}{}
}

func (s *SLVMImageConnectUniqueToolSet) GetPathType(imagePath string) int {
	if _, ok := s.lvms[imagePath]; ok {
		return LVM_PATH
	} else if _, ok := s.nonLVMImagePathSet[imagePath]; ok {
		return NON_LVM_PATH
	} else {
		return PATH_NOT_FOUND
	}
}

func (s *SLVMImageConnectUniqueToolSet) Wait(imagePath string) {
	if tool, ok := s.lvms[imagePath]; ok {
		tool.Wait()
	}
}

func (s *SLVMImageConnectUniqueToolSet) Signal(imagePath string) {
	if tool, ok := s.lvms[imagePath]; ok {
		tool.Signal()

		if tool.CanBeDestoryed() {
			s.lock.Lock()
			defer s.lock.Unlock()
			delete(s.lvms, imagePath)
		}
	}
}

type SKVMGuestLVMPartition struct {
	partDev      string
	originVgname string
	vgname       string
}

func findVgname(partDev string) string {
	output, err := procutils.NewCommand("pvscan").Run()
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

func NewKVMGuestLVMPartition(partDev, originVgname string) *SKVMGuestLVMPartition {
	return &SKVMGuestLVMPartition{
		partDev:      partDev,
		originVgname: originVgname,
		vgname:       stringutils.UUID4(),
	}
}

func (p *SKVMGuestLVMPartition) SetupDevice() bool {
	if len(p.originVgname) == 0 {
		return false
	}
	if !p.vgRename(p.originVgname, p.vgname) {
		return false
	}
	if findVgname(p.partDev) != p.vgname {
		return false
	}
	if p.vgActivate(true) {
		return true
	}
	return false
}

func (p *SKVMGuestLVMPartition) FindPartitions() []*guestfs.SKVMGuestDiskPartition {
	if !p.isVgActive() {
		return nil
	}

	files, err := ioutil.ReadDir("/dev/" + p.vgname)
	if err != nil {
		log.Errorln(err)
		return nil
	}

	parts := []*guestfs.SKVMGuestDiskPartition{}
	for _, f := range files {
		partPath := fmt.Sprintf("/dev/%s/%s", p.vgname, f.Name())
		part := guestfs.NewKVMGuestDiskPartition(partPath)
		parts = append(parts, part)
	}
	return parts
}

func (p *SKVMGuestLVMPartition) PutdownDevice() bool {
	if !p.isVgActive() {
		return false
	}
	if !p.vgActivate(false) {
		return false
	}
	if p.isVgActive() {
		return false
	}
	if len(p.originVgname) == 0 {
		return false
	}
	if p.vgRename(p.vgname, p.originVgname) {
		return true
	}
	return false
}

func (p *SKVMGuestLVMPartition) isVgActive() bool {
	if fileutils2.Exists("/dev/" + p.vgname) {
		return true
	} else {
		return false
	}
}

func (p *SKVMGuestLVMPartition) vgActivate(activate bool) bool {
	param := "-an"
	if activate {
		param = "-ay"
	}
	output, err := procutils.NewCommand("vgchange", param, p.vgname).Run()
	if err != nil {
		log.Errorf("%s", output)
		return false
	}
	return true
}

func (p *SKVMGuestLVMPartition) vgRename(oldname, newname string) bool {
	output, err := procutils.NewCommand("vgrename", oldname, newname).Run()
	if err != nil {
		log.Errorf("%s", output)
		return false
	}
	log.Infof("VG rename succ from %s to %s", oldname, newname)
	return true
}
