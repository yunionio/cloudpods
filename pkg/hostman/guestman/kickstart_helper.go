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

package guestman

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/mountutils"
)

const (
	KICKSTART_MONITOR_TIMEOUT        = 30 * time.Minute
	SERIAL_DEVICE_CHECK_INTERVAL     = 5 * time.Second
	SERIAL_DEVICE_CHECK_MAX_ATTEMPTS = 10
)

type SKickstartSerialMonitor struct {
	serverId     string
	logFilePath  string
	serialDevice string

	scanner *bufio.Scanner
	file    *os.File

	ctx    context.Context
	cancel context.CancelFunc
}

// NewKickstartSerialMonitor creates a new kickstart serial monitor
func NewKickstartSerialMonitor(serverId, logFilePath string) *SKickstartSerialMonitor {
	ctx, cancel := context.WithTimeout(context.Background(), KICKSTART_MONITOR_TIMEOUT)

	return &SKickstartSerialMonitor{
		serverId:    serverId,
		logFilePath: logFilePath,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// waitForSerialDevice waits for the serial device to become available
func (m *SKickstartSerialMonitor) waitForSerialDevice() (string, error) {
	pattern := regexp.MustCompile(`char device redirected to (/dev/pts/\d+) \(label charserial0\)`)

	for attempts := 0; attempts < SERIAL_DEVICE_CHECK_MAX_ATTEMPTS; attempts++ {
		select {
		case <-m.ctx.Done():
			return "", errors.Errorf("context cancelled while waiting for serial device")
		default:
		}

		if content, err := os.ReadFile(m.logFilePath); err == nil {
			if matches := pattern.FindStringSubmatch(string(content)); len(matches) >= 2 {
				devicePath := matches[1]
				if _, err := os.Stat(devicePath); err == nil {
					return devicePath, nil
				}
			}
		}

		time.Sleep(SERIAL_DEVICE_CHECK_INTERVAL)
	}

	return "", errors.Errorf("serial device not available after %d attempts", SERIAL_DEVICE_CHECK_MAX_ATTEMPTS)
}

// connect establishes connection to the serial device
func (m *SKickstartSerialMonitor) connect() error {
	devicePath, err := m.waitForSerialDevice()
	if err != nil {
		return err
	}

	file, err := os.Open(devicePath)
	if err != nil {
		return errors.Wrapf(err, "open serial device %s", devicePath)
	}

	m.file = file
	m.serialDevice = devicePath
	m.scanner = bufio.NewScanner(file)

	log.Infof("Kickstart monitor connected to serial device %s for server %s", devicePath, m.serverId)
	return nil
}

// read reads messages from the serial device
func (m *SKickstartSerialMonitor) read() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("KickstartSerialMonitor read %v %s", r, debug.Stack())
		}
	}()

	// process messages by line
	scanner := m.scanner
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		log.Debugf("Received serial message for server %s: %s", m.serverId, line)

		if line == "KICKSTART_SUCCESS" || line == "KICKSTART_FAILED" {
			log.Infof("Kickstart status update for server %s: %s", m.serverId, line)

			// TODO: logic should vary based on the status
			var status string
			switch line {
			case "KICKSTART_SUCCESS":
				status = api.KICKSTART_STATUS_COMPLETED
				// Unmount ISO and restart VM if kickstart install successfully
				if err := m.handleKickstartSuccess(); err != nil {
					log.Errorf("Failed to handle kickstart success for server %s: %v", m.serverId, err)
				}
			case "KICKSTART_FAILED":
				// TODO: auto retry or alert and stop
				status = api.KICKSTART_STATUS_FAILED
			}

			if err := m.updateKickstartStatus(status); err != nil {
				log.Errorf("Failed to update kickstart status for server %s: %v", m.serverId, err)
			} else {
				log.Infof("Kickstart status for server %s updated to %s", m.serverId, status)
				m.Close()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Debugf("Kickstart serial monitor disconnected %s: %s", m.serverId, err)
	}
}

// Close closes the serial monitor connection
func (m *SKickstartSerialMonitor) Close() error {
	if m.cancel != nil {
		m.cancel()
	}

	if m.file != nil {
		err := m.file.Close()
		m.file = nil
		m.scanner = nil
		log.Infof("Kickstart monitor closed for server %s", m.serverId)
		return err
	}

	return nil
}

// updateKickstartStatus updates the kickstart status via Region API
func (m *SKickstartSerialMonitor) updateKickstartStatus(status string) error {
	ctx := context.Background()
	session := hostutils.GetComputeSession(ctx)

	input := api.ServerUpdateKickstartStatusInput{
		Status: status,
	}

	log.Infof("Updating kickstart status for server %s to %s", m.serverId, status)

	_, err := modules.Servers.PerformAction(session, m.serverId, "update-kickstart-status", jsonutils.Marshal(input))
	if err != nil {
		return errors.Wrapf(err, "failed to update kickstart status for server %s", m.serverId)
	}

	log.Infof("Successfully updated kickstart status for server %s to %s", m.serverId, status)
	return nil
}

// Start starts the kickstart serial monitor
func (m *SKickstartSerialMonitor) Start() error {
	log.Infof("Starting kickstart monitor for server %s", m.serverId)

	if err := m.connect(); err != nil {
		return errors.Wrap(err, "connect to serial device")
	}

	go m.read()

	// Setup timeout handler
	go func() {
		<-m.ctx.Done()
		if m.ctx.Err() == context.DeadlineExceeded {
			log.Warningf("Kickstart monitor timeout for server %s", m.serverId)
		}
		m.Close()
	}()

	return nil
}

// handleKickstartSuccess handles the successful kickstart completion
// It unmounts the ISO image and restarts the VM
func (m *SKickstartSerialMonitor) handleKickstartSuccess() error {
	// Get the server instance to access CDROM information
	server, exists := guestManager.GetServer(m.serverId)
	if !exists {
		return errors.Errorf("server %s not found", m.serverId)
	}

	kvmGuest, ok := server.(*SKVMGuestInstance)
	if !ok {
		return errors.Errorf("server %s is not a KVM guest", m.serverId)
	}

	// Get image ID from CDROM devices to construct mount point
	var imageId string
	if len(kvmGuest.Desc.Cdroms) > 0 {
		for _, cdrom := range kvmGuest.Desc.Cdroms {
			if cdrom.Path != "" {
				filename := path.Base(cdrom.Path)
				imageId = strings.TrimSuffix(filename, ".iso")
				break
			}
		}
	}

	if imageId == "" {
		log.Warningf("No ISO image ID found for server %s, skip unmounting", m.serverId)
	} else {
		// Unmount the ISO image with lazy=true
		mountPoint := fmt.Sprintf("/tmp/kickstart-iso-%s", imageId)
		log.Infof("Unmounting kickstart ISO at %s for server %s", mountPoint, m.serverId)

		if err := mountutils.Unmount(mountPoint, true); err != nil {
			log.Errorf("Failed to unmount ISO at %s: %v", mountPoint, err)
		} else {
			log.Infof("Successfully unmounted kickstart ISO at %s for server %s", mountPoint, m.serverId)
		}
	}

	log.Infof("Restarting VM %s after successful kickstart", m.serverId)

	if err := m.restartServer(); err != nil {
		return errors.Wrapf(err, "failed to restart server %s", m.serverId)
	}

	return nil
}

// restartServer restarts the server using Region API,
// because the kickstart process requires a fully reboot
// to regenerate qemu parameters
func (m *SKickstartSerialMonitor) restartServer() error {
	ctx := context.Background()
	session := hostutils.GetComputeSession(ctx)

	input := jsonutils.NewDict()
	input.Set("is_force", jsonutils.JSONFalse)

	log.Infof("Restarting server %s via Region API", m.serverId)

	_, err := modules.Servers.PerformAction(session, m.serverId, "restart", input)
	if err != nil {
		return errors.Wrapf(err, "failed to restart server %s via API", m.serverId)
	}

	log.Infof("Successfully restarted server %s", m.serverId)
	return nil
}
