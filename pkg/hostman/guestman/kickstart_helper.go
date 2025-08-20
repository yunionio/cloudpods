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
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
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

