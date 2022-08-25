package qga

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"net"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/monitor"
)

type QemuGuestAgent struct {
	id string

	scanner *bufio.Scanner
	rwc     net.Conn
	c       chan struct{}
	mutex   *sync.Mutex
}

func NewQemuGuestAgent(id, qgaSocketPath string) (*QemuGuestAgent, error) {
	conn, err := net.Dial("unix", qgaSocketPath)
	if err != nil {
		return nil, errors.Wrap(err, "dial qga socket")
	}
	return &QemuGuestAgent{
		id:      id,
		rwc:     conn,
		scanner: bufio.NewScanner(conn),
		c:       make(chan struct{}, 1),
		mutex:   &sync.Mutex{},
	}, nil
}

func (qga *QemuGuestAgent) Close() error {
	return qga.rwc.Close()
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
func (qga *QemuGuestAgent) Lock() {
	qga.c <- struct{}{}
}

// Lock before execute qemu guest agent commands
func (qga *QemuGuestAgent) TryLock(timeout time.Duration) bool {
	select {
	case qga.c <- struct{}{}:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Unlock after execute qemu guest agent commands
func (qga *QemuGuestAgent) Unlock() {
	<-qga.c
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
	res, err := qga.execCmd(cmd, info.SupportedCommands[i].SuccessResp)
	if err != nil {
		return nil, err
	}
	return *res, nil
}

func (qga *QemuGuestAgent) execCmd(cmd *monitor.Command, expectResp bool) (*json.RawMessage, error) {
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

	if !qga.scanner.Scan() {
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

func (qga *QemuGuestAgent) GuestPing() error {
	cmd := &monitor.Command{
		Execute: "guest-ping",
	}
	_, err := qga.execCmd(cmd, true)
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

	rawRes, err := qga.execCmd(cmd, true)
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
	_, err := qga.execCmd(cmd, true)
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
	rawRes, err := qga.execCmd(qgaCmd, true)
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
	rawRes, err := qga.execCmd(cmd, true)
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
