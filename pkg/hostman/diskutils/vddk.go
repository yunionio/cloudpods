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

package diskutils

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/kvmpart"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
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
	Proc     *Command
	Pid      int

	kvmDisk      *SKVMGuestDisk
	readOnly     bool
	deployDriver string
}

func NewVDDKDisk(vddkInfo *apis.VDDKConInfo, diskPath, deployDriver string, readOnly bool) (*VDDKDisk, error) {
	return &VDDKDisk{
		Host:         vddkInfo.Host,
		Port:         int(vddkInfo.Port),
		User:         vddkInfo.User,
		Passwd:       vddkInfo.Passwd,
		VmRef:        vddkInfo.Vmref,
		DiskPath:     diskPath,
		deployDriver: deployDriver,
		readOnly:     readOnly,
	}, nil
}

type Command struct {
	*exec.Cmd
	done      chan error
	stdouterr *bytes.Buffer
	stdin     io.Writer
}

func NewCommand(name string, arg ...string) *Command {
	cmd := Command{
		Cmd:  exec.Command(name, arg...),
		done: make(chan error, 1),
	}
	cmd.stdouterr = bytes.NewBuffer([]byte{})
	cmd.Stdout = cmd.stdouterr
	cmd.Stderr = cmd.stdouterr
	cmd.stdin, _ = cmd.StdinPipe()
	return &cmd
}

func (c *Command) Send(msg []byte) error {
	_, err := c.stdin.Write(msg)
	return err
}

func (c *Command) Start() error {
	if err := c.Cmd.Start(); err != nil {
		return err
	}
	go func() {
		c.done <- c.Cmd.Wait()
	}()
	return nil
}

func (c *Command) Exited() bool {
	return len(c.done) == 1
}

// Wait will block
func (c *Command) Wait() error {
	return <-c.done
}

func (c *Command) Kill() error {
	return c.Process.Kill()
}

func execpath() string {
	return "/opt/vmware-vddk/bin/vix-mntapi-sample"
}

func libdir() string {
	return "/usr/lib/vmware"
}

func logpath(pid int) string {
	return fmt.Sprintf("%s/vixDiskLib-%d.log", TMPDIR, pid)
}

func (vd *VDDKDisk) Cleanup() {
	if vd.kvmDisk != nil {
		vd.kvmDisk.Cleanup()
		vd.kvmDisk = nil
	}
}

func (vd *VDDKDisk) Connect(*apis.GuestDesc) error {
	flatFile, err := vd.ConnectBlockDevice()
	if err != nil {
		return errors.Wrap(err, "ConnectBlockDevice")
	}
	vd.kvmDisk, err = NewKVMGuestDisk(qemuimg.SImageInfo{Path: flatFile}, vd.deployDriver, vd.readOnly)
	if err != nil {
		vd.DisconnectBlockDevice()
		return errors.Wrap(err, "NewKVMGuestDisk")
	}
	err = vd.kvmDisk.Connect(nil)
	if err != nil {
		vd.DisconnectBlockDevice()
		return errors.Wrap(err, "kvmDisk connect")
	}
	return nil
}

func (vd *VDDKDisk) Disconnect() error {
	if vd.kvmDisk != nil {
		if err := vd.kvmDisk.Disconnect(); err != nil {
			log.Errorf("kvm disk disconnect failed %s", err)
		}
		vd.kvmDisk.Cleanup()
		vd.kvmDisk = nil
	}
	return vd.DisconnectBlockDevice()
}

func (vd *VDDKDisk) MountRootfs() (fsdriver.IRootFsDriver, error) {
	if vd.kvmDisk == nil {
		return nil, fmt.Errorf("kvmDisk is nil")
	}
	return vd.kvmDisk.MountRootfs()
}

func (vd *VDDKDisk) UmountRootfs(fd fsdriver.IRootFsDriver) error {
	if vd.kvmDisk == nil {
		return nil
	}
	return vd.kvmDisk.UmountRootfs(fd)
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

func (vd *VDDKDisk) Umount() error {
	if vd.Proc != nil {
		err := vd.Proc.Send([]byte{'y'})
		if err != nil {
			errors.Wrap(err, "send 'y' to VDDKDisk.Proc")
		}
		err = vd.Proc.Wait()
		if err != nil {
			return errors.Wrap(err, "vd.Proc.Wait")
		}
	}
	if len(vd.FUseDir) != 0 {
		for _, p := range append(vd.PartDirs, vd.FUseDir) {
			vd.fuseUmount(p)
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

func (vd *VDDKDisk) fuseUmount(path string) {
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
	cmd := NewCommand(execpath(), "-info", "-host", vd.Host, "-port", strconv.Itoa(vd.Port), "-user", vd.User,
		"-password", vd.Passwd, "-mode", "nbd", "-thumb", thumb, "-vm", fmt.Sprintf("moref=%s", vd.VmRef), vd.DiskPath)
	log.Debugf("command to mount: %s", cmd)
	env := os.Environ()
	env = append(env, fmt.Sprintf("LD_LIBRARY_PATH=%s", libdir()))
	cmd.Env = env
	vd.Proc = cmd
	err = vd.Proc.Start()
	if err != nil {
		return errors.Wrap(err, "vd.Proc.Start")
	}
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
	endStr := []byte("Do you want to procede to unmount the volume")
	timeout := 300 * time.Second
	endClock := time.After(timeout)
	isEnd := false

Loop:
	for !vd.Proc.Exited() {
		select {
		case <-endClock:
			break Loop
		default:
			if bytes.Contains(vd.Proc.stdouterr.Bytes(), endStr) {
				log.Debugf("find the mark")
				isEnd = true
				break Loop
			}
		}
		// Reduce inspection density
		time.Sleep(100 * time.Millisecond)
	}

	backup := vd.Proc.stdouterr.String()
	log.Debugf(backup)
	err := vd.ParsePartitions(backup)
	if err != nil {
		return errors.Wrap(err, "VDDKDisk.ParsePartitions")
	}
	if vd.Proc.Exited() {
		retCode := vd.Proc.ProcessState.ExitCode()
		err := vd.Proc.Kill()
		if err != nil {
			log.Errorf("unable to kill process '%d'", vd.Proc.Process.Pid)
		}
		return errors.Error(fmt.Sprintf("VDDKDisk prog exit error(%d): %s", retCode, backup))
	} else if !isEnd {
		err := vd.Proc.Kill()
		if err != nil {
			log.Errorf("unable to kill process '%d'", vd.Proc.Process.Pid)
		}
		return errors.Error("VDDKDisk read timeout, program blocked")
	}
	return nil
}

// connect vddk disk as fuse block device on local host
// return fuse device path, is null error
func (vd *VDDKDisk) ConnectBlockDevice() (string, error) {
	thumb, err := vd.getServerCertThumbSha1(fmt.Sprintf("%s:%d", vd.Host, vd.Port))
	if err != nil {
		return "", errors.Wrapf(err, "Fail contact server %s", vd.Host)
	}
	cmd := NewCommand(execpath(), "-info", "-connect-disk", "-host", vd.Host, "-port", strconv.Itoa(vd.Port), "-user", vd.User,
		"-password", vd.Passwd, "-mode", "nbd", "-thumb", thumb, "-vm", fmt.Sprintf("moref=%s", vd.VmRef), vd.DiskPath)
	log.Infof("command to mount: %s", cmd)
	env := os.Environ()
	env = append(env, fmt.Sprintf("LD_LIBRARY_PATH=%s", libdir()))
	cmd.Env = env
	vd.Proc = cmd
	err = vd.Proc.Start()
	if err != nil {
		return "", errors.Wrap(err, "vd.Proc.Start")
	}
	vd.Pid = cmd.Process.Pid

	var (
		timeout  = time.After(30 * time.Second)
		matchStr = "Log: Disk flat file mounted under"
		flatFile string
	)
Loop:
	for !vd.Proc.Exited() {
		select {
		case <-timeout:
			break Loop
		default:
			if idx := strings.Index(vd.Proc.stdouterr.String(), matchStr); idx >= 0 {
				output := vd.Proc.stdouterr.String()
				output = output[idx+len(matchStr):]
				if idx := strings.Index(output, "\n"); idx < 0 {
					return "", fmt.Errorf("find disk flat file failed")
				} else {
					flatFile = strings.TrimSpace(output[:idx])
				}
				log.Infof("disk flat file mounted under %s", flatFile)
				break Loop
			}
		}
	}
	if vd.Proc.Exited() {
		log.Errorf("process is exited: %s", vd.Proc.stdouterr.String())
		return "", vd.Proc.Wait()
	}
	vd.FUseDir = flatFile
	return path.Join(flatFile, "flat"), nil
}

func (vd *VDDKDisk) DisconnectBlockDevice() error {
	if vd.Proc != nil {
		_, err := vd.Proc.stdin.Write([]byte("y\n"))
		if err != nil {
			return errors.Wrap(err, "send 'y' to VDDKDisk.Proc")
		}
		return vd.Umount()
	}
	return fmt.Errorf("vddk disk has not connected")
}

func (vd *VDDKDisk) DeployGuestfs(req *apis.DeployParams) (res *apis.DeployGuestFsResponse, err error) {
	return vd.kvmDisk.DeployGuestfs(req)
}

func (d *VDDKDisk) ResizeFs() (*apis.Empty, error) {
	return d.kvmDisk.ResizeFs()
}

func (d *VDDKDisk) FormatFs(req *apis.FormatFsParams) (*apis.Empty, error) {
	return d.kvmDisk.FormatFs(req)
}

func (d *VDDKDisk) SaveToGlance(req *apis.SaveToGlanceParams) (*apis.SaveToGlanceResponse, error) {
	return d.kvmDisk.SaveToGlance(req)
}

func (d *VDDKDisk) ProbeImageInfo(req *apis.ProbeImageInfoPramas) (*apis.ImageInfo, error) {
	return d.kvmDisk.ProbeImageInfo(req)
}

type VDDKPartition struct {
	*kvmpart.SLocalGuestFS
}

func (vp *VDDKPartition) Mount() bool {
	log.Warningf("VDDKPartition.Mount not implement")
	return true
}

func (vp *VDDKPartition) MountPartReadOnly() bool {
	log.Warningf("VDDKPartition.MountPartReadOnly not implement")
	return true
}

func (vp *VDDKPartition) Umount() error {
	log.Warningf("VDDKPartition.Umount not implement")
	return nil
}

func (vp *VDDKPartition) IsReadonly() bool {
	return guestfs.IsPartitionReadonly(vp)
}

func (vp *VDDKPartition) GetPhysicalPartitionType() string {
	log.Warningf("VDDKPartition.GetPhysicalPartitionType not implement")
	return ""
}

func (vp *VDDKPartition) GetPartDev() string {
	return ""
}

func (vp *VDDKPartition) IsMounted() bool {
	return true
}

func (vp *VDDKPartition) Zerofree() {
	return
}
