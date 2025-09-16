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
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
)

const (
	QGA_DEFAULT_READ_TIMEOUT_SECOND int = 5
	QGA_EXEC_DEFAULT_WAIT_TIMEOUT   int = 5
)

type QGACallback func([]byte)

type QemuGuestAgent struct {
	id            string
	qgaSocketPath string

	commandQueue  []string
	callbackQueue []QGACallback

	scanner *bufio.Scanner
	rwc     net.Conn

	mutex       *sync.Mutex
	writing     bool
	readTimeout int
}

type TryMutex struct {
	mu sync.Mutex
}

func NewQemuGuestAgent(id, qgaSocketPath string) (*QemuGuestAgent, error) {
	qga := &QemuGuestAgent{
		id:            id,
		qgaSocketPath: qgaSocketPath,
		mutex:         &sync.Mutex{},
		readTimeout:   QGA_DEFAULT_READ_TIMEOUT_SECOND,
	}
	err := qga.connect()
	if err != nil {
		return nil, err
	}
	return qga, nil
}

func (qga *QemuGuestAgent) connect() error {
	qga.mutex.Lock()
	defer qga.mutex.Unlock()

	conn, err := net.Dial("unix", qga.qgaSocketPath)
	if err != nil {
		return errors.Wrap(err, "dial qga socket")
	}

	qga.commandQueue = make([]string, 0)
	qga.callbackQueue = make([]QGACallback, 0)
	qga.rwc = conn
	qga.scanner = bufio.NewScanner(conn)

	go qga.read()
	return nil
}

func (qga *QemuGuestAgent) read() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("QemuGuestAgent read %v %v", r, debug.Stack())
		}
	}()

	scanner := qga.scanner
	for scanner.Scan() {
		res := scanner.Bytes()
		if len(res) == 0 {
			continue
		}
		go qga.callBack(res)
	}
	err := scanner.Err()
	if err != nil {
		log.Debugf("QGA Disconnected %s: %s", qga.id, err)
	}
}

func (qga *QemuGuestAgent) callBack(res []byte) {
	qga.mutex.Lock()
	if len(qga.callbackQueue) == 0 {
		qga.mutex.Unlock()
		return
	}
	cb := qga.callbackQueue[0]
	qga.callbackQueue = qga.callbackQueue[1:]
	qga.mutex.Unlock()
	if cb != nil {
		go cb(res)
	}
}

func (qga *QemuGuestAgent) Close() error {
	qga.mutex.Lock()
	defer qga.mutex.Unlock()

	if qga.rwc == nil {
		return nil
	}
	err := qga.rwc.Close()
	if err != nil {
		return err
	}

	qga.commandQueue = nil
	qga.callbackQueue = nil
	qga.scanner = nil
	qga.rwc = nil
	return nil
}

func (qga *QemuGuestAgent) Query(cmd string, cb QGACallback) int {
	// push
	var cbQueueLength int
	qga.mutex.Lock()
	qga.commandQueue = append(qga.commandQueue, cmd)
	qga.callbackQueue = append(qga.callbackQueue, cb)
	cbQueueLength = len(qga.callbackQueue)
	qga.mutex.Unlock()

	if !qga.writing {
		go qga.query()
	}

	return cbQueueLength
}

func (m *QemuGuestAgent) checkWriting() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.writing {
		return false
	} else {
		m.writing = true
	}
	return true
}

func (m *QemuGuestAgent) query() {
	if !m.checkWriting() {
		return
	}
	for {
		if len(m.commandQueue) == 0 {
			break
		}
		//pop
		m.mutex.Lock()
		cmd := m.commandQueue[0]
		m.commandQueue = m.commandQueue[1:]
		err := m.write(cmd)
		m.mutex.Unlock()
		if err != nil {
			log.Errorf("Write %s to QGA error %s: %s", cmd, m.id, err)
			break
		}
	}
	m.writing = false
}

func (qga *QemuGuestAgent) write(cmd string) error {
	log.Debugf("QGA Write %s: %s", qga.id, cmd)
	length, index := len(cmd), 0
	for index < length {
		i, err := qga.rwc.Write([]byte(cmd))
		if err != nil {
			return err
		}
		index += i
	}
	return nil
}

func (qga *QemuGuestAgent) QgaCommand(cmd *monitor.Command, readTimeout int) ([]byte, error) {
	info, err := qga.GuestInfo()
	if err != nil {
		return nil, err
	}

	if len(info.SupportedCommands) == 0 {
		return nil, errors.Errorf("exec guest-info return empty")
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
	res, err := qga.execCmd(cmd, info.SupportedCommands[i].SuccessResp, readTimeout)
	if err != nil {
		return nil, err
	}
	return *res, nil
}

func (qga *QemuGuestAgent) getQGACallback(expectResp bool, resChan chan string) QGACallback {
	if !expectResp {
		return nil
	}
	return func(res []byte) {
		resChan <- string(res)
	}
}

func (qga *QemuGuestAgent) execCmd(cmd *monitor.Command, expectResp bool, readTimeoutSecond int) (*json.RawMessage, error) {
	if qga.rwc == nil {
		err := qga.connect()
		if err != nil {
			return nil, errors.Wrap(err, "qga connect")
		}
	}

	var resChan = make(chan string)
	var cb = qga.getQGACallback(expectResp, resChan)

	rawCmd := jsonutils.Marshal(cmd).String()
	cbQueueLength := qga.Query(rawCmd, cb)

	if !expectResp {
		return nil, nil
	}

	var res string
	if readTimeoutSecond <= 0 {
		readTimeoutSecond = qga.readTimeout
	}
	select {
	case <-time.After(time.Duration(readTimeoutSecond) * time.Second):
		if cbQueueLength > 30 {
			if err := qga.Close(); err != nil {
				log.Errorf("failed close qga connection %s", err)
			}
		}
		return nil, errors.Errorf("qga read timeout")
	case res = <-resChan:
		break
	}

	var objmap map[string]*json.RawMessage
	log.Debugf("qga response %s", res)
	if err := json.Unmarshal([]byte(res), &objmap); err != nil {
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

func (qga *QemuGuestAgent) QgaFileOpen(path, mode string) (int, error) {
	cmdFileOpen := &monitor.Command{
		Execute: "guest-file-open",
		Args: map[string]interface{}{
			"path": path,
			"mode": mode,
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

type GuestFileWrite struct {
	Count int  `json:"count"`
	Eof   bool `json:"eof"`
}

func (qga *QemuGuestAgent) QgaFileWrite(fileNum int, content string) (int, bool, error) {
	contentEncode := base64.StdEncoding.EncodeToString([]byte(content))
	//write shell to file
	cmdFileWrite := &monitor.Command{
		Execute: "guest-file-write",
		Args: map[string]interface{}{
			"handle":  fileNum,
			"buf-b64": contentEncode,
		},
	}
	rawResFileWrite, err := qga.execCmd(cmdFileWrite, true, -1)
	if err != nil {
		return -1, false, err
	}
	resWrite := new(GuestFileWrite)
	err = json.Unmarshal(*rawResFileWrite, resWrite)
	if err != nil {
		return -1, false, errors.Wrap(err, "unmarshal raw response")
	}

	return resWrite.Count, resWrite.Eof, nil
}

type GuestFileRead struct {
	Count  int    `json:"count"`
	BufB64 string `json:"buf-b64"`
	Eof    bool   `json:"eof"`
}

func (qga *QemuGuestAgent) QgaFileRead(fileNum, readCount int) ([]byte, bool, error) {
	cmdFileRead := &monitor.Command{
		Execute: "guest-file-read",
	}
	args := map[string]interface{}{
		"handle": fileNum,
	}
	// readCount: maximum number of bytes to read (default is 4KB, maximum is 48MB)
	if readCount > 0 {
		args["count"] = readCount
	}
	cmdFileRead.Args = args

	rawResFileRead, err := qga.execCmd(cmdFileRead, true, -1)
	if err != nil {
		return nil, false, err
	}
	resReadInfo := new(GuestFileRead)
	err = json.Unmarshal(*rawResFileRead, resReadInfo)
	if err != nil {
		return nil, false, errors.Wrap(err, "unmarshal raw response")
	}
	content, err := base64.StdEncoding.DecodeString(resReadInfo.BufB64)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed decode base64")
	}

	return content, resReadInfo.Eof, nil
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

func (qga *QemuGuestAgent) QgaAddFileExec(filePath string) error {
	//Adding execution permissions to file
	shellAddAuth := "chmod +x " + filePath
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
	_, err := qga.execCmd(cmdAddAuth, true, -1)
	if err != nil {
		return err
	}
	return nil
}

func (qga *QemuGuestAgent) QgaSetWindowsNetwork(qgaNetMod *monitor.NetworkModify) error {
	ip, subnetMask, err := ParseIPAndSubnet(qgaNetMod.Ipmask)
	if err != nil {
		return err
	}
	networkCmd := fmt.Sprintf(
		"netsh interface ip set address name=\"%s\" source=static addr=%s mask=%s gateway=%s & "+
			"netsh interface ip set address name=\"%s\" dhcp",
		qgaNetMod.Device, ip, subnetMask, qgaNetMod.Gateway, qgaNetMod.Device,
	)

	log.Infof("networkCmd: %s", networkCmd)
	arg := []string{"/C", networkCmd}
	cmdExecNet := &monitor.Command{
		Execute: "guest-exec",
		Args: map[string]interface{}{
			"path":           "C:\\Windows\\System32\\cmd.exe",
			"arg":            arg,
			"env":            []string{},
			"input-data":     "",
			"capture-output": true,
		},
	}
	_, err = qga.execCmd(cmdExecNet, true, -1)
	if err != nil {
		return err
	}
	return nil
}

var NETWORK_RESTRT_SCRIPT = `#!/bin/bash
set -e
DEV=$1

if systemctl is-active --quiet NetworkManager.service; then
	nmcli connection down $DEV && nmcli connection up $DEV
	exit 0
fi

if command -v ifup &> /dev/null; then
	ifdown $DEV && ifup $DEV
	exit 0
fi

if command -v ifconfig &> /dev/null; then
	ifconfig $DEV down && ifconfig $DEV up
	exit 0
fi

if systemctl is-active --quiet network.service; then
	systemctl restart network.service
	exit 0
fi

if command -v ip &> /dev/null; then
	ip link set $DEV down && ip link set $DEV up
	exit 0
fi

echo "No valid method restart network device"
exit 1
`

func (qga *QemuGuestAgent) QgaRestartLinuxNetwork(qgaNetMod *monitor.NetworkModify) error {
	scriptPath := "/tmp/qga_restart_network"
	if err := qga.FilePutContents(scriptPath, NETWORK_RESTRT_SCRIPT, false); err != nil {
		return errors.Wrap(err, "write qga_restart_network script")
	}

	retCode, stdout, stderr, err := qga.CommandWithTimeout("bash", []string{scriptPath, qgaNetMod.Device}, nil, "", true, 10)
	if err != nil {
		return errors.Wrap(err, "CommandWithTimeout")
	}
	if retCode != 0 {
		return errors.Errorf("QgaRestartLinuxNetwork failed: %s %s retcode %d", stdout, stderr, retCode)
	}
	return nil
}

func (qga *QemuGuestAgent) qgaDeployNetworkConfigure(guestNics []*types.SServerNic) error {
	qgaPart := NewQGAPartition(qga)
	fsDriver, err := guestfs.DetectRootFs(qgaPart)
	if err != nil {
		return errors.Wrap(err, "qga DetectRootFs")
	}
	log.Infof("QGA %s DetectRootFs %s", qga.id, fsDriver.String())
	return fsDriver.DeployNetworkingScripts(qgaPart, guestNics)
}

func (qga *QemuGuestAgent) QgaSetNetwork(qgaNetMod *monitor.NetworkModify, guestNics []*types.SServerNic) error {
	//Getting information about the operating system
	resOsInfo, err := qga.QgaGuestGetOsInfo()
	if err != nil {
		return errors.Wrap(err, "get os info")
	}

	//Judgement based on id, currently only windows and other systems are judged
	switch resOsInfo.Id {
	case "mswindows":
		return qga.QgaSetWindowsNetwork(qgaNetMod)
	default:
		// do deploy network configure
		if err := qga.qgaDeployNetworkConfigure(guestNics); err != nil {
			return errors.Wrap(err, "qgaDeployNetworkConfigure")
		}
		return qga.QgaRestartLinuxNetwork(qgaNetMod)
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
