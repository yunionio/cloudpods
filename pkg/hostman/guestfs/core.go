package guestfs

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/hostman"
)

type SDeployInfo struct {
	publicKey *sshkeys.SSHKeys
	deploys   jsonutils.JSONObject
	password  string
	isInit    bool
}

type SLocalGuestFS struct {
	mountPath string
	readOnly  bool
}

func (f *SLocalGuestFS) isReadonly() bool {
	log.Infof("Test if read-only fs ...")
	var filename = fmt.Sprint("./%f", rand.Float32())
	if err := hostman.FilePutContents(filename, fmt.Sprint("%f", rand.Float32()), false); err == nil {
		f.Remove(filename, false)
		return false
	} else {
		log.Errorf("File system is readonly: %s", err)
		f.readOnly = true
		return true
	}
}

func (f *SLocalGuestFS) getLocalPath(sPath string, caseInsensitive bool) string {
	var fullPath = f.mountPath
	pathSegs := strings.Split(sPath, "/")
	for _, seg := range pathSegs {
		if len(seg) > 0 {
			var realSeg string
			files, _ := ioutil.ReadDir(fullPath)
			for _, file := range files {
				var f = file.Name()
				if f == seg || (caseInsensitive && (strings.ToLower(f)) == strings.ToLower(seg)) ||
					(seg[len(seg)-1] == '*' && strings.HasPrefix(f, seg[:len(seg)-1])) ||
					(caseInsensitive && strings.HasPrefix(strings.ToLower(f),
						strings.ToLower(seg[:len(seg)]))) {
					realSeg = f
					break
				}
			}
			if len(realSeg) > 0 {
				fullPath = path.Join(fullPath, realSeg)
			} else {
				return ""
			}
		}
	}
	return fullPath
}

func (f *SLocalGuestFS) Remove(path string, caseInsensitive bool) {
	path = f.getLocalPath(path, caseInsensitive)
	if len(path) > 0 {
		os.Remove(path)
	}
}

func NewLocalGuestFS(mountPath string) *SLocalGuestFS {
	var ret = new(SLocalGuestFS)
	ret.mountPath = mountPath
	return ret
}

type IRootFsDriver interface {
	GetPartition() *SKVMGuestDiskPartition
	String() string
	TestRootfs() bool
}

var rootfsDrivers map[string]IRootFsDriver

func DetectRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	for k, v := range rootfsDrivers {
		if v.TestRootfs(part) {
			return v
		}
	}
	return nil
}
