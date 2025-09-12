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
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/mountutils"
)

const (
	KICKSTART_MONITOR_TIMEOUT         = 30 * time.Minute
	SERIAL_FILE_CHECK_INTERVAL        = 5 * time.Second
	KICKSTART_BASE_DIR                = "/tmp/kickstart"
	KICKSTART_ISO_MOUNT_DIR           = "iso-mount"
	KICKSTART_ISO_BUILD_DIR           = "iso-build"
	KICKSTART_ISO_FILENAME            = "config.iso"
	REDHAT_KICKSTART_ISO_VOLUME_LABEL = "OEMDRV"
	UBUNTU_KICKSTART_ISO_VOLUME_LABEL = "CIDATA"
)

var (
	kickstartInstallingRegex = regexp.MustCompile(`(?i).*KICKSTART_INSTALLING.*`)
	kickstartCompletedRegex  = regexp.MustCompile(`(?i).*KICKSTART_COMPLETED.*`)
	kickstartFailedRegex     = regexp.MustCompile(`(?i).*KICKSTART_FAILED.*`)
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

	// Get the server instance to access its homeDir
	var serialFilePath string
	server, exists := guestManager.GetServer(serverId)
	if exists {
		serialFilePath = path.Join(server.HomeDir(), "kickstart-serial.log")
	} else {
		// Fallback to use /tmp if server not found
		serialFilePath = fmt.Sprintf("/tmp/kickstart-serial-%s.log", serverId)
	}

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

		var status string
		var shouldClose bool = false
		var matched bool = false

		if kickstartInstallingRegex.MatchString(line) {
			status = api.VM_KICKSTART_INSTALLING
			shouldClose = false
			matched = true
		} else if kickstartCompletedRegex.MatchString(line) {
			status = api.VM_KICKSTART_COMPLETED
			shouldClose = true
			matched = true
			// Unmount ISO and restart VM if kickstart install successfully
			if err := m.handleKickstartCompleted(); err != nil {
				log.Errorf("Failed to handle kickstart success for server %s: %v", m.serverId, err)
			}
		} else if kickstartFailedRegex.MatchString(line) {
			// TODO: auto retry or alert and stop
			status = api.VM_KICKSTART_FAILED
			shouldClose = true
			matched = true
		}

		if matched {
			log.Infof("Kickstart status update for server %s: %s (matched: %s)", m.serverId, status, line)

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
			if err := m.updateKickstartStatus(api.VM_KICKSTART_FAILED); err != nil {
				log.Errorf("Failed to update kickstart status to failed on timeout for server %s: %v", m.serverId, err)
			}
			m.Close()
		}
	}()

	return nil
}

// handleKickstartCompleted handles the successful kickstart completion
// It unmounts the ISO image and restarts the VM
func (m *SKickstartSerialMonitor) handleKickstartCompleted() error {
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
		mountPoint := filepath.Join(KICKSTART_BASE_DIR, m.serverId, KICKSTART_ISO_MOUNT_DIR)
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
	tmpDir := filepath.Join(KICKSTART_BASE_DIR, serverId, KICKSTART_ISO_BUILD_DIR)
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
	case "centos", "rhel", "fedora", "openeuler":
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
	isoPath := filepath.Join(KICKSTART_BASE_DIR, serverId, KICKSTART_ISO_FILENAME)
	args := []string{
		"-o", isoPath,
		"-V", volumeLabel,
		"-r",
		"-J",
	}
	args = append(args, filePaths...)

	cmd := exec.Command("mkisofs", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", errors.Wrapf(err, "mkisofs failed: %s", string(output))
	}

	log.Infof("Successfully created kickstart ISO: %s", isoPath)
	return isoPath, nil
}

// ComputeKickstartKernelInitrdPaths returns absolute kernel and initrd paths by combining
// mountPath with OS-specific relative paths and validating that files exist.
// GetKernelInitrdPaths resolves absolute kernel and initrd paths under a mounted ISO.
func GetKernelInitrdPaths(mountPath, osType string) (string, string, error) {
	var kernelRelPath, initrdRelPath string
	switch osType {
	case "centos", "rhel", "fedora", "openeuler":
		kernelRelPath = "images/pxeboot/vmlinuz"
		initrdRelPath = "images/pxeboot/initrd.img"
	case "ubuntu":
		kernelRelPath = "casper/vmlinuz"
		initrdRelPath = "casper/initrd"
	default:
		return "", "", errors.Errorf("unsupported OS type: %s", osType)
	}

	kernelPath := path.Join(mountPath, kernelRelPath)
	initrdPath := path.Join(mountPath, initrdRelPath)

	if !fileutils2.Exists(kernelPath) {
		return "", "", errors.Errorf("kernel file not found: %s", kernelPath)
	}
	if !fileutils2.Exists(initrdPath) {
		return "", "", errors.Errorf("initrd file not found: %s", initrdPath)
	}
	return kernelPath, initrdPath, nil
}

// BuildKickstartAppendArgs builds kernel append args for kickstart/autoinstall
// based on OS type and whether a local config ISO is present.
// isoPath non-empty indicates a locally attached config ISO.
func BuildKickstartAppendArgs(config *api.KickstartConfig, isoPath string) string {
	if config == nil {
		return ""
	}
	baseArgs := []string{}
	var kickstartArgs []string
	switch config.OSType {
	case "centos", "rhel", "fedora", "openeuler":
		if config.ConfigURL != "" {
			kickstartArgs = append(kickstartArgs, fmt.Sprintf("inst.ks=%s", config.ConfigURL))
		} else if isoPath != "" {
			kickstartArgs = append(kickstartArgs, fmt.Sprintf("inst.ks=hd:LABEL=%s:/anaconda-ks.cfg", REDHAT_KICKSTART_ISO_VOLUME_LABEL))
		} else {
			kickstartArgs = append(kickstartArgs, "inst.ks=cdrom:/ks.cfg")
		}
	case "ubuntu":
		if config.ConfigURL != "" {
			kickstartArgs = append(kickstartArgs,
				"autoinstall",
				"ip=dhcp",
				fmt.Sprintf("ds=nocloud-net;s=%s", config.ConfigURL),
			)
		} else if isoPath != "" {
			kickstartArgs = append(kickstartArgs, "autoinstall")
		} else {
			kickstartArgs = append(kickstartArgs, "autoinstall", "ip=dhcp", "ds=nocloud;s=/cdrom/")
		}
	}
	return strings.Join(append(baseArgs, kickstartArgs...), " ")
}
