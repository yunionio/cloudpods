package guestfs

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
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

func (f *SLocalGuestFS) getLocalPath(path string, caseInsensitive bool) string {
	var fullPath = f.mountPath
	pathSegs := strings.Split(path, "/")
	for i := 0; i < len(pathSegs); i++ {
		if len(pathSegs[i]) > 0 {
			var readSeg string
			files, _ := ioutil.ReadDir(fullPath)
			for _, file := range files {
				if file.Name() == pathSegs[i] 
                    || (caseInsensitive && strings.ToLower(file.Name())) == strings.ToLower(pathSegs[i]) {

				}
			}
		}
	}
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
}

var rootfsDrivers map[string]IRootFsDriver

func DetectRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	//TODO
	return nil
}
