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
	"os/exec"
	"path"
	"path/filepath"
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
	KICKSTART_MONITOR_TIMEOUT         = 30 * time.Minute
	SERIAL_FILE_CHECK_INTERVAL        = 5 * time.Second
	KICKSTART_SERIAL_FILE_PREFIX      = "/tmp/kickstart-serial"
	KICKSTART_ISO_DIR_PREFIX          = "/tmp/kickstart-iso"
	REDHAT_KICKSTART_ISO_VOLUME_LABEL = "OEMDRV"
	UBUNTU_KICKSTART_ISO_VOLUME_LABEL = "CIDATA"
)

type SKickstartSerialMonitor struct {
	serverId       string
	serialFilePath string

	ctx    context.Context
	cancel context.CancelFunc
}

// NewKickstartSerialMonitor creates a new kickstart serial monitor
func NewKickstartSerialMonitor(serverId string) *SKickstartSerialMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	serialFilePath := fmt.Sprintf("%s-%s.log", KICKSTART_SERIAL_FILE_PREFIX, serverId)

	return &SKickstartSerialMonitor{
		serverId:       serverId,
		serialFilePath: serialFilePath,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (m *SKickstartSerialMonitor) GetSerialFilePath() string {
	return m.serialFilePath
}

// getKickstartTimeout gets kickstart timeout from kickstart config, defaults to KICKSTART_MONITOR_TIMEOUT
func (m *SKickstartSerialMonitor) getKickstartTimeout() time.Duration {
	server, exists := guestManager.GetServer(m.serverId)
	if !exists {
		return KICKSTART_MONITOR_TIMEOUT
	}

	kvmGuest, ok := server.(*SKVMGuestInstance)
	if !ok {
		return KICKSTART_MONITOR_TIMEOUT
	}

	kickstartConfigStr, exists := kvmGuest.Desc.Metadata[api.VM_METADATA_KICKSTART_CONFIG]
	if !exists || kickstartConfigStr == "" {
		return KICKSTART_MONITOR_TIMEOUT
	}

	configObj, err := jsonutils.ParseString(kickstartConfigStr)
	if err != nil {
		return KICKSTART_MONITOR_TIMEOUT
	}

	var config api.KickstartConfig
	if err := configObj.Unmarshal(&config); err != nil {
		return KICKSTART_MONITOR_TIMEOUT
	}

	if config.TimeoutMinutes <= 0 {
		return KICKSTART_MONITOR_TIMEOUT
	}

	timeout := time.Duration(config.TimeoutMinutes) * time.Minute
	log.Infof("Using kickstart timeout %v for server %s", timeout, m.serverId)
	return timeout
}

func (m *SKickstartSerialMonitor) ensureSerialFile() error {
	if _, err := os.Stat(m.serialFilePath); os.IsNotExist(err) {
		file, err := os.Create(m.serialFilePath)
		if err != nil {
			return errors.Wrapf(err, "create serial file %s", m.serialFilePath)
		}
		file.Close()
		log.Infof("Created kickstart serial file %s for server %s", m.serialFilePath, m.serverId)
	}
	return nil
}

func (m *SKickstartSerialMonitor) monitorSerialFile() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("KickstartSerialMonitor monitor %v %s", r, debug.Stack())
		}
	}()

	var lastSize int64 = 0

	ticker := time.NewTicker(SERIAL_FILE_CHECK_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.scanSerialForStatus(&lastSize); err != nil {
				log.Errorf("Failed to scan serial file for status for server %s: %v", m.serverId, err)
			}
		}
	}
}

// scanSerialForStatus scans the serial file for a kickstart status update.
// It reads new content since the last check, parses it for status keywords,
// and triggers status updates and cleanup when a final status is detected.
func (m *SKickstartSerialMonitor) scanSerialForStatus(lastSize *int64) error {
	fileInfo, err := os.Stat(m.serialFilePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	currentSize := fileInfo.Size()
	if currentSize <= *lastSize {
		return nil
	}

	// Read new content
	file, err := os.Open(m.serialFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Seek(*lastSize, 0)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		log.Debugf("Received serial message for server %s: %s", m.serverId, line)

		if line == "KICKSTART_INSTALLING" || line == "KICKSTART_SUCCESS" || line == "KICKSTART_FAILED" {
			log.Infof("Kickstart status update for server %s: %s", m.serverId, line)

			// TODO: logic should vary based on the status
			var status string
			var shouldClose bool = false
			switch line {
			case "KICKSTART_INSTALLING":
				status = api.KICKSTART_STATUS_INSTALLING
				shouldClose = false
			case "KICKSTART_SUCCESS":
				status = api.KICKSTART_STATUS_COMPLETED
				shouldClose = true
				// Unmount ISO and restart VM if kickstart install successfully
				if err := m.handleKickstartSuccess(); err != nil {
					log.Errorf("Failed to handle kickstart success for server %s: %v", m.serverId, err)
				}
			case "KICKSTART_FAILED":
				// TODO: auto retry or alert and stop
				status = api.KICKSTART_STATUS_FAILED
				shouldClose = true
			}

			if err := m.updateKickstartStatus(status); err != nil {
				log.Errorf("Failed to update kickstart status for server %s: %v", m.serverId, err)
			} else {
				log.Infof("Kickstart status for server %s updated to %s", m.serverId, status)
				if shouldClose {
					m.Close()
					return nil
				}
			}
		}
	}

	*lastSize = currentSize
	return scanner.Err()
}

func (m *SKickstartSerialMonitor) Close() error {
	if m.cancel != nil {
		m.cancel()
	}

	if m.serialFilePath != "" {
		if err := os.Remove(m.serialFilePath); err != nil && !os.IsNotExist(err) {
			log.Warningf("Failed to remove serial file %s: %v", m.serialFilePath, err)
		} else {
			log.Debugf("Removed serial file %s", m.serialFilePath)
		}
	}

	log.Infof("Kickstart monitor closed for server %s", m.serverId)
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
	log.Infof("Starting kickstart monitor for server %s, serial file: %s", m.serverId, m.serialFilePath)

	if err := m.ensureSerialFile(); err != nil {
		return errors.Wrap(err, "ensure serial file")
	}

	go m.monitorSerialFile()

	// Setup timeout handler
	go func() {
		timeout := m.getKickstartTimeout()
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		select {
		case <-m.ctx.Done():
			return
		case <-timer.C:
			log.Warningf("Kickstart monitor timeout (%v) for server %s, setting status to failed", timeout, m.serverId)
			if err := m.updateKickstartStatus(api.KICKSTART_STATUS_FAILED); err != nil {
				log.Errorf("Failed to update kickstart status to failed on timeout for server %s: %v", m.serverId, err)
			}
			m.Close()
		}
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
	var kickstartConfigIsoPath string
	if len(kvmGuest.Desc.Cdroms) > 0 {
		for _, cdrom := range kvmGuest.Desc.Cdroms {
			if cdrom.Path != "" {
				filename := path.Base(cdrom.Path)
				imageId = strings.TrimSuffix(filename, ".iso")

				// Check if this is a kickstart ISO
				if strings.HasPrefix(filename, "kickstart-") {
					kickstartConfigIsoPath = cdrom.Path
				}
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

	// Clean up kickstart ISO file if it exists
	if kickstartConfigIsoPath != "" {
		if err := os.Remove(kickstartConfigIsoPath); err != nil && !os.IsNotExist(err) {
			log.Errorf("Failed to cleanup kickstart ISO %s: %v", kickstartConfigIsoPath, err)
		} else {
			log.Infof("Cleaned up kickstart ISO: %s", kickstartConfigIsoPath)
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

// CreateKickstartConfigISO creates an ISO image containing kickstart configuration files
// For Red Hat systems: creates ks.cfg in a ISO with label 'OEMDRV'
// For Ubuntu systems: creates user-data and meta-data files in a ISO with label 'CIDATA'
// reference: https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/10/html/automatically_installing_rhel/starting-kickstart-installations#starting-a-kickstart-installation-automatically-using-a-local-volume
// https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/10/html/automatically_installing_rhel/starting-kickstart-installations#starting-a-kickstart-installation-automatically-using-a-local-volume
func CreateKickstartConfigISO(config *api.KickstartConfig, serverId string) (string, error) {
	log.Infof("Creating kickstart ISO for server %s with OS type %s", serverId, config.OSType)

	if config == nil {
		return "", errors.Errorf("kickstart config is nil")
	}

	if config.Config == "" {
		return "", errors.Errorf("kickstart config content is empty")
	}

	log.Infof("Kickstart config content length: %d characters", len(config.Config))

	// Create temporary directory for ISO contents
	tmpDir := fmt.Sprintf("%s-%s", KICKSTART_ISO_DIR_PREFIX, serverId)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", errors.Wrapf(err, "failed to create temp directory %s", tmpDir)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Warningf("Failed to cleanup temp directory %s: %v", tmpDir, err)
		}
	}()

	var filePaths []string
	var volumeLabel string

	switch config.OSType {
	case "centos", "rhel", "fedora":
		// Create anaconda-ks.cfg for Red Hat systems
		ksFilePath := filepath.Join(tmpDir, "anaconda-ks.cfg")
		if err := os.WriteFile(ksFilePath, []byte(config.Config), 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write kickstart file %s", ksFilePath)
		}
		filePaths = []string{ksFilePath}
		volumeLabel = REDHAT_KICKSTART_ISO_VOLUME_LABEL

	case "ubuntu":
		// Create user-data file for Ubuntu systems
		userDataPath := filepath.Join(tmpDir, "user-data")
		if err := os.WriteFile(userDataPath, []byte(config.Config), 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write user-data file %s", userDataPath)
		}

		// Create empty meta-data file
		metaDataPath := filepath.Join(tmpDir, "meta-data")
		if err := os.WriteFile(metaDataPath, []byte(""), 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write meta-data file %s", metaDataPath)
		}
		filePaths = []string{userDataPath, metaDataPath}
		volumeLabel = UBUNTU_KICKSTART_ISO_VOLUME_LABEL

	default:
		return "", errors.Errorf("unsupported OS type: %s", config.OSType)
	}

	// Create ISO using mkisofs
	isoPath := fmt.Sprintf("/tmp/kickstart-%s.iso", serverId)
	args := []string{
		"-o", isoPath,
		"-V", volumeLabel,
		"-r",
		"-J",
	}
	args = append(args, filePaths...)

	cmd := exec.Command("/usr/bin/mkisofs", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", errors.Wrapf(err, "mkisofs failed: %s", string(output))
	}

	log.Infof("Successfully created kickstart ISO: %s", isoPath)
	return isoPath, nil
}
