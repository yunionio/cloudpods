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

package storageman

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

const (
	TMPDIR = "/tmp/vmware-root"
)

var (
	MNT_PATTERN = regexp.MustCompile(`Disk flat file mounted under ([^\s]+)`)
)

type VDDKDisk struct {
	Host     string
	Port     int
	User     string
	Passwd   string
	VmRef    string
	DiskPath string

	FUseDir  string
	PartDirs []string
	Proc     *exec.Cmd
	Pid      int
}

func execpath() string {
	return "/opt/vmware-vddk/bin/vix-mntapi-sample"
}

func libdir() string {
	return "/usr/lib/vmware"
}

func logpath(pid int) string {
	return fmt.Sprintf("/tmp/vmware-root/vixDiskLib-%d.log", pid)
}

func (vd *VDDKDisk) ParsePartitions(buf string) error {
	// Disk flat file mounted under /run/vmware/fuse/7673253059900458465
	// Mounted Volume 1, Type 1, isMounted 1, symLink /tmp/vmware-root/7673253059900458465_1, numGuestMountPoints 0 (<null>)
	// print buf
	ms := MNT_PATTERN.FindAllStringSubmatch(buf, -1)
	if len(ms) != 0 {
		vd.FUseDir = ms[0][1]
		diskId := filepath.Base(vd.FUseDir)
		files, err := ioutil.ReadDir(TMPDIR)
		if err != nil {
			return errors.Wrapf(err, "ioutil.ReadDir for %s", TMPDIR)
		}
		for _, f := range files {
			if strings.HasPrefix(f.Name(), diskId) {
				vd.PartDirs = append(vd.PartDirs, filepath.Join(TMPDIR, f.Name()))
			}
		}
	}
	log.Infof("Fuse path: %s partitiaons: %s", vd.FUseDir, vd.PartDirs)
	return nil
}

func (vd *VDDKDisk) Mount() (err error) {
	defer func() {
		if err == nil {
			return
		}
		vd.Proc = nil
		log.Errorf("Exec vix-mntapi-sample error: %s", err)
	}()

	err = vd.ExecProg()
	if err != nil {
		return errors.Wrap(err, "VDDKDisk.ExecProg")
	}
	err = vd.WaitMounted()
	if err != nil {
		return errors.Wrap(err, "VDDKDisk.Mount")
	}
	return nil
}

func (vd *VDDKDisk) Unmount() error {
	if vd.Proc != nil {
		time.Sleep(time.Second)
		_, err := vd.Proc.Stdin.Read([]byte{'y'})
		if err != nil {
			errors.Wrap(err, "send 'y' to VDDKDisk.Proc")
		}
	}
	if len(vd.FUseDir) != 0 {
		for _, p := range append(vd.PartDirs, vd.FUseDir) {
			vd.fUseUmount(p)
		}
	}
	if vd.Pid != 0 {
		logpath := logpath(vd.Pid)
		_, err := os.Stat(logpath)
		if err == nil || os.IsExist(err) {
			os.Remove(logpath)
		}
	}
	return nil
}

func (vd *VDDKDisk) fUseUmount(path string) {
	maxTries, tried := 4, 0

	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		// no such path
		return
	}

	for tried < maxTries {
		tried += 1
		err := exec.Command("umount", path).Run()
		if err != nil {
			time.Sleep(time.Duration(tried) * 15 * time.Second)
			log.Errorf("Fail to umount %s: %s", path, err)
			continue
		}
		_, err = os.Stat(path)
		if err == nil || os.IsExist(err) {
			err = exec.Command("rm", "-rf", path).Run()
			if err != nil {
				time.Sleep(time.Duration(tried) * 15 * time.Second)
				log.Errorf("Fail to umount %s: %s", path, err)
				continue
			}
		}
	}
}

func (vd *VDDKDisk) ExecProg() error {
	thumb, err := vd.getServerCertThumbSha1(fmt.Sprintf("%s:%d", vd.Host, vd.Port))
	if err != nil {
		return errors.Wrapf(err, "Fail contact server %s", vd.Host)
	}
	cmd := exec.Command(execpath(), "-info", "-host", vd.Host, "-port", strconv.Itoa(vd.Port), "-user", vd.User,
		"-password", vd.Passwd, "-mode", "nbd", "-thumb", thumb, "-vm", fmt.Sprintf("moref=%s", vd.VmRef))
	env := os.Environ()
	env = append(env, fmt.Sprintf("LD_LIBRARY_PATH=%s", libdir()))
	cmd.Env = env
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "cmd.Start")
	}
	vd.Proc = cmd
	vd.Pid = cmd.Process.Pid
	return nil
}

// getServerCertBin try to obtain the remote ssl certificate
func (vd *VDDKDisk) getServerCertBin(addr string) ([]byte, error) {
	rawConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Wrapf(err, "net.Dial for addr '%s'", addr)
	}
	defer rawConn.Close()

	// get the wrpped conn
	sslWrappedConn := tls.Client(rawConn, &tls.Config{InsecureSkipVerify: true})
	err = sslWrappedConn.Handshake()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to complete ssl handshake with addr '%s'", addr)
	}
	return sslWrappedConn.ConnectionState().PeerCertificates[0].Raw, nil
}

func (vd *VDDKDisk) getServerCertThumbSha1(addr string) (string, error) {
	certBin, err := vd.getServerCertBin(addr)
	if err != nil {
		return "", err
	}
	sha := sha1.Sum(certBin)
	shaHex := hex.EncodeToString(sha[:])
	length := len(shaHex) / 2 * 2
	tmp := make([][]byte, 0, length)
	for i := 1; i < length; i += 2 {
		tmp = append(tmp, []byte{shaHex[i-1], shaHex[i]})
	}
	return string(bytes.Join(tmp, []byte{':'})), nil
}

func (vd *VDDKDisk) WaitMounted() error {
	var buf bytes.Buffer
	endStr := []byte("Do you want to procede to unmount the volume")
	timeout := 30 * time.Second
	endClock := time.After(timeout)
	isEnd := false

Loop:
	for !vd.Proc.ProcessState.Exited() {
		select {
		case <-endClock:
			break Loop
		default:
			bys := make([]byte, 0, 100)
			_, err := vd.Proc.Stdout.Write(bys)
			if err != nil {
				log.Errorf("Read error: %s", err)
				break Loop
			}
			buf.Write(bys)
			if bytes.Contains(buf.Bytes(), endStr) {
				isEnd = true
				break Loop
			}
		}
	}

	bufStr := buf.String()

	err := vd.ParsePartitions(bufStr)
	if err != nil {
		return errors.Wrap(err, "VDDKDisk.ParsePartitions")
	}
	if vd.Proc.ProcessState.Exited() {
		retCode := vd.Proc.ProcessState.ExitCode()
		// ignore the error
		vd.Proc.Process.Kill()
		vd.Proc = nil
		return errors.Error(fmt.Sprintf("VDDKDisk prog exit error(%d): %s", retCode, bufStr))
	} else if isEnd {
		// timeout
		vd.Proc.Process.Kill()
		time.Sleep(time.Second)
		return errors.Error(fmt.Sprintf("VDDKDisk read timeout, program blocked"))
	}
	return nil
}

type VDDKPartition struct {
	*guestfs.SLocalGuestFS
}

func (vp *VDDKPartition) Mount() bool {
	panic("implement me")
}

func (vp *VDDKPartition) MountPartReadOnly() bool {
	panic("implement me")
}

func (vp *VDDKPartition) Umount() bool {
	panic("implement me")
}

func (vp *VDDKPartition) IsReadonly() bool {
	return guestfs.IsPartitionReadonly(vp)
}

func (vp *VDDKPartition) GetPhysicalPartitionType() string {
	panic("implement me")
}

func newVDDKPartition(mntPath string) *VDDKPartition {
	return &VDDKPartition{guestfs.NewLocalGuestFS(mntPath)}
}

func (vp *VDDKPartition) probe() error {
	// test readonly ??
	return nil
}

type sMountVDDKRootfs struct {
	Passwd   string
	Host     string
	Port     int
	User     string
	VmRef    string
	DiskPath string
	Disk     *VDDKDisk
}

func newMountVDDKRootfs(vmref, diskPath string, host interface{}, port int, user, passwd string) *sMountVDDKRootfs {

	var realHost string

	if dict, ok := host.(*jsonutils.JSONDict); ok {
		passwd, _ = dict.GetString("password")
		realHost, _ = dict.GetString("host")
		user, _ = dict.GetString("account")

		port1, _ := dict.Int("port")
		port = int(port1)
		vcId, _ := dict.GetString("vcenter_id")
		p, err := seclib2.DecryptBase64(vcId, passwd)
		if err == nil {
			passwd = p
		}
	} else {
		realHost = host.(string)
	}
	return &sMountVDDKRootfs{
		Passwd:   passwd,
		Host:     realHost,
		Port:     port,
		User:     user,
		VmRef:    vmref,
		DiskPath: diskPath,
		Disk:     nil,
	}
}

func (mvr *sMountVDDKRootfs) enter() fsdriver.IRootFsDriver {
	drivers := fsdriver.GetRootfsDrivers()
	disk := VDDKDisk{
		Host:     mvr.Host,
		Port:     mvr.Port,
		User:     mvr.User,
		Passwd:   mvr.Passwd,
		VmRef:    mvr.VmRef,
		DiskPath: mvr.DiskPath,
	}
	mvr.Disk = &disk
	if err := disk.Mount(); err != nil {
		for _, mntPath := range disk.PartDirs {
			part := newVDDKPartition(mntPath)
			part.probe()
			for _, drvcls := range drivers {
				fs := drvcls(part)
				if mvr.testRootFs(fs) {
					return fs
				}
			}
		}
	}
	return nil
}

func (mvr *sMountVDDKRootfs) exit() {
	if mvr.Disk != nil {
		mvr.Disk.Unmount()
	}
}

func (mvr *sMountVDDKRootfs) testRootFs(fs fsdriver.IRootFsDriver) bool {
	caseInsensitive := fs.IsFsCaseInsensitive()
	for _, rd := range fs.RootSignatures() {
		if !fs.GetPartition().Exists(rd, caseInsensitive) {
			log.Infof("[%s]test_root_fs: %s not exists", fs.GetName(), rd)
			return false
		}
	}
	ex := fs.RootExcludeSignatures()
	if ex != nil {
		for _, rd := range ex {
			if fs.GetPartition().Exists(rd, caseInsensitive) {
				log.Infof("[%s]test_root_fs: %s exists, test failed", fs.GetName(), rd)
				return false
			}
		}
	}
	return true
}

type MoutVdDDKRootfsParam struct {
	Vmref    string
	DiskPath string
	Host     interface{}
	Port     int
	User     string
	Passwd   string
}

func MountVDDKRootfs(param MoutVdDDKRootfsParam, dealFunc func(fsdriver.IRootFsDriver)) {
	mvr := newMountVDDKRootfs(param.Vmref, param.DiskPath, param.Host, param.Port, param.User, param.Passwd)
	dealFunc(mvr.enter())
	mvr.exit()
}
