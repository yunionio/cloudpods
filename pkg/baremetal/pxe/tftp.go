package pxe

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pin/tftp"

	"yunion.io/x/log"
)

var (
	PxeLinuxCfgPattern = `^pxelinux.cfg/01-(?P<mac>([0-9a-f]{2}-){5}[0-9a-f]{2})$`
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

// ReadHandler is called when client starts file download from server
func (h *TFTPHandler) ReadHandler(filename string, rf io.ReaderFrom) error {
	log.Debugf("TFTP request file: %s", filename)
	regEx := regexp.MustCompile(PxeLinuxCfgPattern)
	matches := regEx.FindStringSubmatch(filename)

	if len(matches) != 0 {
		paramsMap := make(map[string]string)
		// pxelinux config matched
		for i, name := range regEx.SubexpNames() {
			if i > 0 && i <= len(matches) {
				paramsMap[name] = matches[i]
			}
		}
		mac, ok := paramsMap["mac"]
		if !ok {
			return fmt.Errorf("request filename %q not found mac pattern", filename)
		}
		macAddr, err := net.ParseMAC(mac)
		if err != nil {
			return fmt.Errorf("Parse mac string %q error: %v", mac, err)
		}
		return h.sendPxeLinuxCfgResponse(macAddr, rf)
	}
	return h.sendFile(filename, rf)
}

func (h *TFTPHandler) sendPxeLinuxCfgResponse(mac net.HardwareAddr, rf io.ReaderFrom) error {
	log.Debugf("[TFTP] client mac: %s", mac)
	bmInstance := h.BaremetalManager.GetBaremetalByMac(mac)
	if bmInstance == nil {
		err := fmt.Errorf("Not found baremetal instance by mac: %s", mac)
		log.Errorf("Get baremetal error: %v", err)
		return err
	}
	respStr := bmInstance.GetTFTPResponse()
	log.Debugf("[TFTP] get tftp response config: %s", respStr)
	size := len(respStr)
	buffer := bytes.NewBufferString(respStr)

	rf.(tftp.OutgoingTransfer).SetSize(int64(size))

	n, err := rf.ReadFrom(buffer)
	if err != nil {
		return err
	}

	log.Debugf("[TFTP] %d bytes sent", n)
	return nil
}

func (h *TFTPHandler) sendFile(filename string, rf io.ReaderFrom) error {
	filename = h.getFilePath(filename)
	file, err := os.Open(filename)
	if err != nil {
		log.Errorf("TFTP open file %q error: %v", filename, err)
		return err
	}
	n, err := rf.ReadFrom(file)
	if err != nil {
		return err
	}
	log.Debugf("[TFTP] %d bytes sent", n)
	return nil
}

func (h *TFTPHandler) getFilePath(fileName string) string {
	return filepath.Join(h.RootDir, fileName)
}

func (s *Server) serveTFTP(srv *tftp.Server) error {
	addr := fmt.Sprintf("%s:%d", s.Address, s.TFTPPort)
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		return err
	}

	srv.Serve(conn)
	return nil
}
