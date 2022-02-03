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

package pxe

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/util/tftp"
)

var (
	PxeLinuxCfgPattern = `^pxelinux.cfg/01-(?P<mac>([0-9a-f]{2}-){5}[0-9a-f]{2})$`
	GrubCfgPattern     = `grub/grub.cfg-01-(?P<mac>([0-9a-f]{2}-){5}[0-9a-f]{2})$`
)

type TFTPHandler struct {
	RootDir          string
	BaremetalManager IBaremetalManager
}

func NewTFTPHandler(rootDir string, baremetalManager IBaremetalManager) (*TFTPHandler, error) {
	if _, err := os.Stat(rootDir); err != nil {
		return nil, fmt.Errorf("TFTP root dir %q  stat error: %v", rootDir, err)
	}
	return &TFTPHandler{
		RootDir:          rootDir,
		BaremetalManager: baremetalManager,
	}, nil
}

// Handle is called when client starts file download from server
func (h *TFTPHandler) Handle(filename string, clientAddr net.Addr) (io.ReadCloser, int64, error) {
	regPxeEx := regexp.MustCompile(PxeLinuxCfgPattern)
	pxeMatches := regPxeEx.FindStringSubmatch(filename)

	regGrubEx := regexp.MustCompile(GrubCfgPattern)
	grubMatches := regGrubEx.FindStringSubmatch(filename)

	if len(pxeMatches) != 0 || len(grubMatches) != 0 {
		paramsMap := make(map[string]string)
		// pxelinux config matched
		for i, name := range regPxeEx.SubexpNames() {
			if i > 0 && i <= len(pxeMatches) {
				paramsMap[name] = pxeMatches[i]
			}
		}
		// grub config matched
		for i, name := range regGrubEx.SubexpNames() {
			if i > 0 && i <= len(grubMatches) {
				paramsMap[name] = grubMatches[i]
			}
		}
		mac, ok := paramsMap["mac"]
		if !ok {
			return nil, 0, fmt.Errorf("request filename %q not found mac pattern", filename)
		}
		macAddr, err := net.ParseMAC(mac)
		if err != nil {
			return nil, 0, fmt.Errorf("Parse mac string %q error: %v", mac, err)
		}
		return h.sendPxeLinuxCfgResponse(macAddr, clientAddr)
	}
	return h.sendFile(filename, clientAddr)
}

func (h *TFTPHandler) sendPxeLinuxCfgResponse(mac net.HardwareAddr, _ net.Addr) (io.ReadCloser, int64, error) {
	log.Debugf("[TFTP] client mac: %s", mac)
	bmInstance := h.BaremetalManager.GetBaremetalByMac(mac)
	if bmInstance == nil {
		err := fmt.Errorf("Not found baremetal instance by mac: %s", mac)
		log.Errorf("Get baremetal error: %v", err)
		return nil, 0, err
	}
	respStr := bmInstance.GetTFTPResponse()
	log.Debugf("[TFTP] get tftp response config: %s", respStr)
	bs := []byte(respStr)
	size := int64(len(bs))
	buffer := bytes.NewBufferString(respStr)

	return ioutil.NopCloser(buffer), size, nil
}

func (h *TFTPHandler) sendFile(filename string, _ net.Addr) (io.ReadCloser, int64, error) {
	filename = h.getFilePath(filename)

	st, err := os.Stat(filename)
	if err != nil {
		log.Errorf("TFTP stat file %q error: %v", filename, err)
		return nil, 0, err
	}
	if !st.Mode().IsRegular() {
		return nil, 0, fmt.Errorf("requested path %q is not a file", filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Errorf("TFTP open file %q error: %v", filename, err)
		return nil, 0, err
	}
	return file, st.Size(), err
}

func (h *TFTPHandler) getFilePath(fileName string) string {
	if path, ok := options.Options.TftpFileMap[fileName]; ok {
		return path
	}
	return filepath.Join(h.RootDir, fileName)
}

func (h *TFTPHandler) transferLog(clientAddr net.Addr, path string, err error) {
	log.Debugf("TFTP transfer log clientAddr: %s, path: %s, error: %v", clientAddr, path, err)
}

func (s *Server) serveTFTP(l net.PacketConn, handler *TFTPHandler) error {
	bindDial := func(network, addr string) (net.Conn, error) {
		localIp := l.LocalAddr().(*net.UDPAddr).IP
		remoteAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		log.Debugf("[TFTP] dial udp localAddr %s, remoteAddr: %s", localIp, remoteAddr.String())
		return net.DialUDP("udp", &net.UDPAddr{
			IP:   localIp,
			Port: 0, // random free port
		}, remoteAddr)
	}

	ts := tftp.Server{
		Handler:     handler.Handle,
		InfoLog:     func(msg string) { log.Debugf("TFTP msg: %s", msg) },
		TransferLog: handler.transferLog,
		Dial:        bindDial,

		MaxBlockSize:  int64(options.Options.TftpBlockSizeInBytes),
		WriteAttempts: options.Options.TftpMaxTimeoutRetries,
	}
	err := ts.Serve(l)
	if err != nil {
		return fmt.Errorf("TFTP server shut down: %v", err)
	}
	return nil
}
