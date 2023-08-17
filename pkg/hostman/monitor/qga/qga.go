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

package qga

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/monitor"
)

const QGA_DEFAULT_READ_TIMEOUT_SECOND int = 5

type QemuGuestAgent struct {
	id            string
	qgaSocketPath string

	scanner     *bufio.Scanner
	rwc         net.Conn
	tm          *TryMutex
	mutex       *sync.Mutex
	readTimeout int
}

type TryMutex struct {
	mu sync.Mutex
}

func (m *TryMutex) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&m.mu)), 0, 1)
}

func (m *TryMutex) Unlock() {
	atomic.StoreInt32((*int32)(unsafe.Pointer(&m.mu)), 0)
}

func NewQemuGuestAgent(id, qgaSocketPath string) (*QemuGuestAgent, error) {
	qga := &QemuGuestAgent{
		id:            id,
		qgaSocketPath: qgaSocketPath,
		tm:            &TryMutex{},
		mutex:         &sync.Mutex{},
		readTimeout:   QGA_DEFAULT_READ_TIMEOUT_SECOND * 1000,
	}
	err := qga.connect()
	if err != nil {
		return nil, err
	}
	return qga, nil
}

func (qga *QemuGuestAgent) SetTimeout(timeout int) {
	qga.readTimeout = timeout
}

func (qga *QemuGuestAgent) ResetTimeout() {
	qga.readTimeout = QGA_DEFAULT_READ_TIMEOUT_SECOND * 1000
}

func (qga *QemuGuestAgent) connect() error {
	conn, err := net.Dial("unix", qga.qgaSocketPath)
	if err != nil {
		return errors.Wrap(err, "dial qga socket")
	}
	qga.rwc = conn
	qga.scanner = bufio.NewScanner(conn)
	return nil
}

func (qga *QemuGuestAgent) Close() error {
	err := qga.rwc.Close()
	if err != nil {
		return err
	}

	qga.scanner = nil
	qga.rwc = nil
	return nil
}

func (qga *QemuGuestAgent) write(cmd []byte) error {
	log.Infof("QGA Write %s: %s", qga.id, string(cmd))
	length, index := len(cmd), 0
	for index < length {
		i, err := qga.rwc.Write(cmd)
		if err != nil {
			return err
		}
		index += i
	}
	return nil
}

// Lock before execute qemu guest agent commands
func (qga *QemuGuestAgent) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&qga.tm.mu)), 0, 1)
}

// Unlock after execute qemu guest agent commands
func (qga *QemuGuestAgent) Unlock() {
	atomic.StoreInt32((*int32)(unsafe.Pointer(&qga.tm.mu)), 0)
}

func (qga *QemuGuestAgent) QgaCommand(cmd *monitor.Command) ([]byte, error) {
	info, err := qga.GuestInfo()
	if err != nil {
		return nil, err
	}
	var i = 0
	for ; i < len(info.SupportedCommands); i++ {
		if info.SupportedCommands[i].Name == cmd.Execute {
			break
		}
	}
	if i > len(info.SupportedCommands) {
		return nil, errors.Errorf("unsupported command %s", cmd.Execute)
	}
	if !info.SupportedCommands[i].Enabled {
		return nil, errors.Errorf("command %s not enabled", cmd.Execute)
	}
	res, err := qga.execCmd(cmd, info.SupportedCommands[i].SuccessResp, -1)
	if err != nil {
		return nil, err
	}
	return *res, nil
}

func (qga *QemuGuestAgent) execCmd(cmd *monitor.Command, expectResp bool, readTimeout int) (*json.RawMessage, error) {
	if qga.TryLock() {
		qga.Unlock()
		return nil, errors.Errorf("qga exec cmd but not locked")
	}

	if qga.rwc == nil {
		err := qga.connect()
		if err != nil {
			return nil, errors.Wrap(err, "qga connect")
		}
	}

	rawCmd, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "marshal qga cmd")
	}

	err = qga.write(rawCmd)
	if err != nil {
		return nil, errors.Wrap(err, "write cmd")
	}

	if !expectResp {
		return nil, nil
	}

	if readTimeout < 0 {
		readTimeout = qga.readTimeout
	}
	err = qga.rwc.SetReadDeadline(time.Now().Add(time.Duration(readTimeout) * time.Millisecond))
	if err != nil {
		return nil, errors.Wrap(err, "set read deadline")
	}

	if !qga.scanner.Scan() {
		defer qga.Close()
		return nil, errors.Wrap(qga.scanner.Err(), "qga scanner")
	}
	var objmap map[string]*json.RawMessage
	b := qga.scanner.Bytes()
	log.Infof("qga response %s", b)
	if err := json.Unmarshal(b, &objmap); err != nil {
		return nil, errors.Wrap(err, "unmarshal qga res")
	}
	if val, ok := objmap["return"]; ok {
		return val, nil
	} else if val, ok := objmap["error"]; ok {
		res := &monitor.Error{}
		if err := json.Unmarshal(*val, res); err != nil {
			return nil, errors.Wrapf(err, "unmarshal qemu error resp: %s", *val)
		}
		return nil, errors.Errorf(res.Error())
	} else {
		return nil, nil
	}
}

func (qga *QemuGuestAgent) GuestPing(timeout int) error {
	cmd := &monitor.Command{
		Execute: "guest-ping",
	}
	_, err := qga.execCmd(cmd, true, timeout)
	return err
}

type GuestCommand struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`

	// whether command returns a response on success (since 1.7)
	SuccessResp bool `json:"success-response"`
}

type GuestInfo struct {
	Version           string         `json:"version"`
	SupportedCommands []GuestCommand `json:"supported_commands"`
}

func (qga *QemuGuestAgent) GuestInfo() (*GuestInfo, error) {
	cmd := &monitor.Command{
		Execute: "guest-info",
	}

	rawRes, err := qga.execCmd(cmd, true, -1)
	if err != nil {
		return nil, err
	}
	if rawRes == nil {
		return nil, errors.Errorf("qga no response")
	}
	res := new(GuestInfo)
	err = json.Unmarshal(*rawRes, res)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal raw response")
	}
	return res, nil
}

func (qga *QemuGuestAgent) GuestInfoTask() ([]byte, error) {
	info, err := qga.GuestInfo()
	if err != nil {
		return nil, err
	}
	cmd := &monitor.Command{
		Execute: "guest-info",
	}
	var i = 0
	for ; i < len(info.SupportedCommands); i++ {
		if info.SupportedCommands[i].Name == cmd.Execute {
			break
		}
	}
	if i > len(info.SupportedCommands) {
		return nil, errors.Errorf("unsupported command %s", cmd.Execute)
	}
	if !info.SupportedCommands[i].Enabled {
		return nil, errors.Errorf("command %s not enabled", cmd.Execute)
	}
	res, err := qga.execCmd(cmd, true, -1)
	if err != nil {
		return nil, err
	}
	return *res, nil
}

type IPAddress struct {
	IPAddress     string `json:"ip-address"`
	IPAddressType string `json:"ip-address-type"`
	Prefix        int    `json:"prefix"`
}

type IfnameDetail struct {
	HardwareAddress string      `json:"hardware-address"`
	IPAddresses     []IPAddress `json:"ip-addresses"`
	Name            string      `json:"name"`
	Statistics      struct {
		RxBytes   int `json:"rx-bytes"`
		RxDropped int `json:"rx-dropped"`
		RxErrs    int `json:"rx-errs"`
		RxPackets int `json:"rx-packets"`
		TxBytes   int `json:"tx-bytes"`
		TxDropped int `json:"tx-dropped"`
		TxErrs    int `json:"tx-errs"`
		TxPackets int `json:"tx-packets"`
	} `json:"statistics"`
}

func (qga *QemuGuestAgent) QgaGetNetwork() ([]byte, error) {
	cmd := &monitor.Command{
		Execute: "guest-network-get-interfaces",
	}
	res, err := qga.execCmd(cmd, true, -1)
	if err != nil {
		return nil, err
	}
	return *res, nil
}

type GuestOsInfo struct {
	Id            string `json:"id"`
	KernelRelease string `json:"kernel-release"`
	KernelVersion string `json:"kernel-version"`
	Machine       string `json:"machine"`
	Name          string `json:"name"`
	PrettyName    string `json:"pretty-name"`
	Version       string `json:"version"`
	VersionId     string `json:"version-id"`
}

func (qga *QemuGuestAgent) QgaGuestGetOsInfo() (*GuestOsInfo, error) {
	//run guest-get-osinfo
	cmdOsInfo := &monitor.Command{
		Execute: "guest-get-osinfo",
	}
	rawResOsInfo, err := qga.execCmd(cmdOsInfo, true, -1)
	resOsInfo := new(GuestOsInfo)
	err = json.Unmarshal(*rawResOsInfo, resOsInfo)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal raw response")
	}
	return resOsInfo, nil
}

func (qga *QemuGuestAgent) QgaFileOpen(path string) (int, error) {
	//file open
	cmdFileOpen := &monitor.Command{
		Execute: "guest-file-open",
		Args: map[string]interface{}{
			"path": path,
			"mode": "w+",
		},
	}
	rawResFileOpen, err := qga.execCmd(cmdFileOpen, true, -1)
	if err != nil {
		return 0, err
	}
	fileNum, err := strconv.ParseInt(string(*rawResFileOpen), 10, 64)
	if err != nil {
		return 0, err
	}
	return int(fileNum), nil
}

func (qga *QemuGuestAgent) QgaFileWrite(fileNum int, content string) error {
	contentEncode := base64.StdEncoding.EncodeToString([]byte(content))
	//write shell to file
	cmdFileWrite := &monitor.Command{
		Execute: "guest-file-write",
		Args: map[string]interface{}{
			"handle":  fileNum,
			"buf-b64": contentEncode,
		},
	}
	_, err := qga.execCmd(cmdFileWrite, true, -1)
	if err != nil {
		return err
	}
	return nil
}

func (qga *QemuGuestAgent) QgaFileClose(fileNum int) error {
	//close file
	cmdFileClose := &monitor.Command{
		Execute: "guest-file-close",
		Args: map[string]interface{}{
			"handle": fileNum,
		},
	}
	_, err := qga.execCmd(cmdFileClose, true, -1)
	if err != nil {
		return err
	}
	return nil
}

func ParseIPAndSubnet(input string) (string, string, error) {
	//Converting IP/MASK format to IP and MASK
	parts := strings.Split(input, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid input format")
	}

	ip := parts[0]
	subnetSizeStr := parts[1]

	subnetSize := 0
	for _, c := range subnetSizeStr {
		if c < '0' || c > '9' {
			return "", "", fmt.Errorf("Invalid subnet size")
		}
		subnetSize = subnetSize*10 + int(c-'0')
	}

	mask := net.CIDRMask(subnetSize, 32)
	subnetMask := net.IP(mask).To4().String()
	return ip, subnetMask, nil
}

func (qga *QemuGuestAgent) QgaSetNetwork(qgaNetMod *monitor.NetworkModify) ([]byte, error) {
	//Getting information about the operating system
	resOsInfo, err := qga.QgaGuestGetOsInfo()
	if err != nil {
		return nil, err
	}

	//Judgement based on id, currently only windows and other systems are judged
	if resOsInfo.Id == "mswindows" {
		ip, subnetMask, err := ParseIPAndSubnet(qgaNetMod.Ipmask)
		if err != nil {
			return nil, err
		}
		networkCmd := fmt.Sprintf("netsh interface ip set address name=\"%s\" source=static addr=%s mask=%s gateway=%s && "+
			"netsh interface ip set address name=\"%s\" dhcp",
			qgaNetMod.Device, ip, subnetMask, qgaNetMod.Gateway, qgaNetMod.Device)

		//Save commands executed by windows to the host computer
		qgaWinTest := "/tmp/qgaWinScripts.txt"
		fileOsInfo, err := os.Create(qgaWinTest)
		if err != nil {
			return nil, err
		}
		// Guaranteed closure of files
		defer fileOsInfo.Close()

		_, err = fileOsInfo.Write([]byte(networkCmd))
		if err != nil {
			return nil, err
		}

		//Execute windows command
		arg := []string{"/c", networkCmd}
		cmdPath := "C:\\Windows\\System32\\cmd.exe"
		cmdExecNet := &monitor.Command{
			Execute: "guest-exec",
			Args: map[string]interface{}{
				"path":           cmdPath,
				"arg":            arg,
				"env":            []string{},
				"input-data":     "",
				"capture-output": true,
			},
		}
		resExec, err := qga.execCmd(cmdExecNet, true, -1)
		if err != nil {
			return nil, err
		}
		return *resExec, nil

	} else {
		//Network Configuration Script Contents
		networkCmd := fmt.Sprintf("#!/bin/bash\nset +e\n"+
			"/sbin/ip -4 address flush dev %s\n/sbin/ip -4 address add %s dev %s\n"+
			"/sbin/ip route del default dev %s\n/sbin/ip route add default via %s dev %s\n",
			qgaNetMod.Device, qgaNetMod.Ipmask, qgaNetMod.Device, qgaNetMod.Device, qgaNetMod.Gateway, qgaNetMod.Device)

		//Write the contents of the script to the virtual machine file
		fileFileOpenPath := "/tmp/deviceRestart.sh"
		returnFileNum, err := qga.QgaFileOpen(fileFileOpenPath)
		if err != nil {
			return nil, err
		}

		//shell script write to file
		err = qga.QgaFileWrite(returnFileNum, networkCmd)
		if err != nil {
			return nil, err
		}

		//file close
		err = qga.QgaFileClose(returnFileNum)
		if err != nil {
			return nil, err
		}

		//Adding execution permissions to a file
		shellAddAuth := "chmod +x " + fileFileOpenPath
		arg := []string{"-c", shellAddAuth}
		cmdAddAuth := &monitor.Command{
			Execute: "guest-exec",
			Args: map[string]interface{}{
				"path":           "/bin/bash",
				"arg":            arg,
				"env":            []string{},
				"input-data":     "",
				"capture-output": true,
			},
		}
		_, err = qga.execCmd(cmdAddAuth, true, -1)
		if err != nil {
			return nil, err
		}

		//Executing shell scripts
		cmdExecShell := &monitor.Command{
			Execute: "guest-exec",
			Args: map[string]interface{}{
				"path":           fileFileOpenPath,
				"arg":            []string{},
				"env":            []string{},
				"input-data":     "",
				"capture-output": true,
			},
		}
		resExec, err := qga.execCmd(cmdExecShell, true, -1)
		if err != nil {
			return nil, err
		}
		return *resExec, nil
	}

}

/*
# @username: the user account whose password to change
# @password: the new password entry string, base64 encoded
# @crypted: true if password is already crypt()d, false if raw
#
# If the @crypted flag is true, it is the caller's responsibility
# to ensure the correct crypt() encryption scheme is used. This
# command does not attempt to interpret or report on the encryption
# scheme. Refer to the documentation of the guest operating system
# in question to determine what is supported.
#
# Not all guest operating systems will support use of the
# @crypted flag, as they may require the clear-text password
#
# The @password parameter must always be base64 encoded before
# transmission, even if already crypt()d, to ensure it is 8-bit
# safe when passed as JSON.
#
# Returns: Nothing on success.
#
# Since: 2.3
*/
func (qga *QemuGuestAgent) GuestSetUserPassword(username, password string, crypted bool) error {
	password64 := base64.StdEncoding.EncodeToString([]byte(password))
	cmd := &monitor.Command{
		Execute: "guest-set-user-password",
		Args: map[string]interface{}{
			"username": username,
			"password": password64,
			"crypted":  crypted,
		},
	}
	_, err := qga.execCmd(cmd, true, -1)
	if err != nil {
		return err
	}
	return nil
}

/*
##
# @GuestExec:
# @pid: pid of child process in guest OS
#
# Since: 2.5
##
{ 'struct': 'GuestExec',
  'data': { 'pid': 'int'} }
*/

type GuestExec struct {
	Pid int
}

/*
##
# @guest-exec:
#
# Execute a command in the guest
#
# @path: path or executable name to execute
# @arg: argument list to pass to executable
# @env: environment variables to pass to executable
# @input-data: data to be passed to process stdin (base64 encoded)
# @capture-output: bool flag to enable capture of
#                  stdout/stderr of running process. defaults to false.
#
# Returns: PID on success.
#
# Since: 2.5
##
{ 'command': 'guest-exec',
  'data':    { 'path': 'str', '*arg': ['str'], '*env': ['str'],
               '*input-data': 'str', '*capture-output': 'bool' },
  'returns': 'GuestExec' }
*/

func (qga *QemuGuestAgent) GuestExecCommand(
	cmdPath string, args, env []string, inputData string, captureOutput bool,
) (*GuestExec, error) {
	qgaCmd := &monitor.Command{
		Execute: "guest-exec",
		Args: map[string]interface{}{
			"path":           cmdPath,
			"arg":            args,
			"env":            env,
			"input-data":     inputData,
			"capture-output": captureOutput,
		},
	}
	rawRes, err := qga.execCmd(qgaCmd, true, -1)
	if err != nil {
		return nil, err
	}
	if rawRes == nil {
		return nil, errors.Errorf("qga no response")
	}
	res := new(GuestExec)
	err = json.Unmarshal(*rawRes, res)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal raw response")
	}
	return res, nil
}

/*
##
# @GuestExecStatus:
#
# @exited: true if process has already terminated.
# @exitcode: process exit code if it was normally terminated.
# @signal: signal number (linux) or unhandled exception code
#       (windows) if the process was abnormally terminated.
# @out-data: base64-encoded stdout of the process
# @err-data: base64-encoded stderr of the process
#       Note: @out-data and @err-data are present only
#       if 'capture-output' was specified for 'guest-exec'
# @out-truncated: true if stdout was not fully captured
#       due to size limitation.
# @err-truncated: true if stderr was not fully captured
#       due to size limitation.
#
# Since: 2.5
##
{ 'struct': 'GuestExecStatus',

	'data': { 'exited': 'bool', '*exitcode': 'int', '*signal': 'int',
	          '*out-data': 'str', '*err-data': 'str',
	          '*out-truncated': 'bool', '*err-truncated': 'bool' }}
*/
type GuestExecStatus struct {
	Exited       bool
	Exitcode     int
	Signal       int
	OutData      string `json:"out-data"`
	ErrData      string `json:"err-data"`
	OutTruncated bool   `json:"out-truncated"`
	ErrTruncated bool   `json:"err-truncated"`
}

/*
##
# @guest-exec-status:
#
# Check status of process associated with PID retrieved via guest-exec.
# Reap the process and associated metadata if it has exited.
#
# @pid: pid returned from guest-exec
#
# Returns: GuestExecStatus on success.
#
# Since: 2.5
##
{ 'command': 'guest-exec-status',
  'data':    { 'pid': 'int' },
  'returns': 'GuestExecStatus' }
*/

func (qga *QemuGuestAgent) GuestExecStatusCommand(pid int) (*GuestExecStatus, error) {
	cmd := &monitor.Command{
		Execute: "guest-exec-status",
		Args: map[string]interface{}{
			"pid": pid,
		},
	}
	rawRes, err := qga.execCmd(cmd, true, -1)
	if err != nil {
		return nil, err
	}
	if rawRes == nil {
		return nil, errors.Errorf("qga no response")
	}
	res := new(GuestExecStatus)
	err = json.Unmarshal(*rawRes, res)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal raw response")
	}
	return res, nil
}
