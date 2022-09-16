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

package guestfish

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/sortedmap"
)

type Guestfish struct {
	*exec.Cmd

	stdoutScanner *bufio.Scanner
	stderrScanner *bufio.Scanner

	pty *os.File

	lock  sync.Mutex
	label string

	stdout  *os.File
	stdoutW *os.File
	stderr  *os.File
	stderrW *os.File

	alive bool
}

const guestFishToken = "><fs>"

func NewGuestfish() (*Guestfish, error) {
	gf := &Guestfish{Cmd: exec.Command("guestfish")}
	stdout, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stderr, stderrW, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	gf.stdout = stdout
	gf.stdoutW = stdoutW

	gf.stderr = stderr
	gf.stderrW = stderrW

	gf.Cmd.Stdout = stdoutW
	gf.Cmd.Stderr = stderrW

	gf.stdoutScanner = bufio.NewScanner(stdout)
	gf.stderrScanner = bufio.NewScanner(stderr)

	pty, err := pty.Start(gf.Cmd)
	if err != nil {
		return nil, errors.Wrap(err, "start Guestfish")
	}

	/* pty handle stdin */
	gf.pty = pty

	/* exec guestfish run command */
	if err = gf.Run(); err != nil {
		return nil, err
	}

	gf.alive = true
	return gf, nil
}

func (fish *Guestfish) IsAlive() bool {
	return fish.alive
}

func (fish *Guestfish) execute(cmd string) ([]string, error) {
	fish.lock.Lock()
	defer fish.lock.Unlock()
	log.Debugf("exec command: %s", cmd)
	_, err := fish.pty.WriteString(cmd + "\n\n")
	if err != nil {
		return nil, errors.Wrapf(err, "exec cmd %s", cmd)
	}
	return fish.fetch()
}

func (fish *Guestfish) fetch() ([]string, error) {
	var (
		stdout = make([]string, 0)
	)

	var tokenMeeted = false
	for fish.stdoutScanner.Scan() {
		line := fish.stdoutScanner.Text()
		log.Debugf("Guestfish stdoutScanner: %s", line)
		if strings.HasPrefix(line, guestFishToken) {
			if !tokenMeeted {
				tokenMeeted = true
				continue
			}
			log.Debugf("fetch success")
			break
		}
		stdout = append(stdout, line)
	}
	if err := fish.stdoutScanner.Err(); err != nil {
		log.Errorf("scan guestfish stdoutScanner error %s", err)
		fish.Quit()
		return nil, err
	}

	fish.stderr.SetReadDeadline(time.Now().Add(time.Second * 1))
	output, err := io.ReadAll(fish.stderr)
	if err != nil && !strings.Contains(err.Error(), "i/o timeout") {
		log.Errorf("scan guestfish stderrScanner error %s", err)
		fish.Quit()
		return nil, err
	}
	var stderrErr error
	if len(output) > 0 {
		stderrErr = errors.Errorf(string(output))
	}
	return stdout, stderrErr
}

/* Fetch error message from stderrScanner, until got ><fs> from stdoutScanner */
func (fish *Guestfish) fetchError() error {
	_, err := fish.fetch()
	return err
}

func (fish *Guestfish) Run() error {
	_, err := fish.execute("run")
	return err
}

func (fish *Guestfish) Quit() error {
	checkError := func(err error) {
		if err != nil {
			log.Errorln(err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("fish quit faild %s \nstack: %s", r, debug.Stack())
		}
	}()

	fish.lock.Lock()
	defer fish.lock.Unlock()
	log.Debugf("exec command: %s", "quit")
	_, err := fish.pty.WriteString("quit\n\n")
	if err != nil {
		return errors.Wrap(err, "exec cmd kill-subprocess and quit")
	}

	fish.alive = false
	if err := fish.Cmd.Wait(); err != nil {
		log.Errorf("failed wait guestfish %s", err)
	}

	checkError(fish.stdout.Close())
	checkError(fish.stderr.Close())
	checkError(fish.stdoutW.Close())
	checkError(fish.stderrW.Close())
	checkError(fish.pty.Close())
	return err
}

func (fish *Guestfish) AddDrive(path, label string, readonly bool) error {
	cmd := fmt.Sprintf("add-drive %s label:%s", path, label)
	if readonly {
		cmd += " readonly:true"
	}
	_, err := fish.execute(cmd)
	if err != nil {
		return err
	}
	fish.label = label
	return nil
}

func (fish *Guestfish) RemoveDrive() error {
	if len(fish.label) == 0 {
		return errors.Errorf("no drive add")
	}
	_, err := fish.execute(fmt.Sprintf("remove-drive %s", fish.label))
	if err != nil {
		return err
	}
	fish.label = ""
	return err
}

func (fish *Guestfish) ListFilesystems() (*sortedmap.SSortedMap, error) {
	output, err := fish.execute("list-filesystems")
	if err != nil {
		return nil, err
	}
	return fish.parseListFilesystemsOutput(output), nil
}

func (fish *Guestfish) parseListFilesystemsOutput(output []string) *sortedmap.SSortedMap {
	/* /dev/sda1: xfs
	   /dev/centos/root: xfs
	   /dev/centos/swap: swap */
	res := sortedmap.SSortedMap{}
	for i := 0; i < len(output); i++ {
		line := output[i]
		log.Debugf("line %s", line)
		segs := strings.Split(line, ":")
		log.Debugf("parse line of list filesystems: %#v", segs)
		if len(segs) != 2 {
			log.Warningf("Guestfish: parse list filesystem got unwanted line: %s", line)
		}
		res = sortedmap.Add(res, strings.TrimSpace(segs[0]), strings.TrimSpace(segs[1]))
	}
	return &res
}

func (fish *Guestfish) ListDevices() ([]string, error) {
	return fish.execute("list-devices")
}

func (fish *Guestfish) Mount(partition string) error {
	_, err := fish.execute(fmt.Sprintf("mount %s /", partition))
	return err
}

func (fish *Guestfish) MountLocal(localmountpoint string, readonly bool) error {
	cmd := fmt.Sprintf("mount-local %s", localmountpoint)
	if readonly {
		cmd += " readonly:true"
	}
	_, err := fish.execute(cmd)
	return err
}

func (fish *Guestfish) Umount(partition string) error {
	_, err := fish.execute("umount")
	return err
}

func (fish *Guestfish) UmountLocal() error {
	_, err := fish.execute("umount-local")
	return err
}

/* This should only be called after "mount_local" returns successfully.
 * The call will not return until the filesystem is unmounted. */
func (fish *Guestfish) MountLocalRun() error {
	_, err := fish.execute("mount-local-run")
	return err
}

/* Clears the LVM cache and performs a volume group scan. */
func (fish *Guestfish) LvmClearFilter() error {
	_, err := fish.execute("lvm-clear-filter")
	return err
}

func (fish *Guestfish) Lvs() ([]string, error) {
	return fish.execute("lvs")
}

func (fish *Guestfish) SfdiskL(dev string) ([]string, error) {
	return fish.execute(fmt.Sprintf("sfdisk-l %s", dev))
}

func (fish *Guestfish) Fsck(dev, fs string) error {
	out, err := fish.execute(fmt.Sprintf("fsck %s %s", fs, dev))
	log.Infof("FSCK ret code: %v", out)
	return err
}

func (fish *Guestfish) Ntfsfix(dev string) error {
	out, err := fish.execute(fmt.Sprintf("ntfsfix %s", dev))
	log.Infof("NTFSFIX ret code: %v", out)
	return err
}

func (fish *Guestfish) Zerofree(dev string) error {
	_, err := fish.execute(fmt.Sprintf("zerofree %s", dev))
	return err
}

func (fish *Guestfish) ZeroFreeSpace(dir string) error {
	_, err := fish.execute(fmt.Sprintf("zero-free-space %s", dir))
	return err
}

func (fish *Guestfish) Blkid(partDev string) ([]string, error) {
	return fish.execute(fmt.Sprintf("blkid %s", partDev))
}

func (fish *Guestfish) Mkswap(partDev, uuid, label string) error {
	cmd := fmt.Sprintf("mkswap %s", partDev)
	if len(uuid) > 0 {
		cmd += fmt.Sprintf(" uuid:%s", uuid)
	}
	if len(label) > 0 {
		cmd += fmt.Sprintf(" label:%s", label)
	}
	_, err := fish.execute(cmd)
	return err
}

func (fish *Guestfish) Mkfs(dev, fs string) error {
	_, err := fish.execute(fmt.Sprintf("mkfs %s %s", fs, dev))
	return err
}

func (fish *Guestfish) PartDisk(dev, diskType string) error {
	_, err := fish.execute(fmt.Sprintf("part-disk %s %s", dev, diskType))
	return err
}
