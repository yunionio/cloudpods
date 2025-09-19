package fsdriver

import (
	"fmt"
	"strings"
)

func genSshdConfig(lines []string, loginAccount, loginPassword string, sshPort int) []string {
	var permitRootLogin, passwordAuthentication, port string
	if loginAccount == "root" {
		permitRootLogin = "PermitRootLogin yes"
	}
	if len(loginPassword) > 0 {
		passwordAuthentication = "PasswordAuthentication yes"
	}
	if sshPort > 0 && sshPort != 22 {
		port = fmt.Sprintf("Port %d", sshPort)
	}
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "PermitRootLogin") {
			if len(permitRootLogin) > 0 {
				line = permitRootLogin
				permitRootLogin = ""
			}
		} else if strings.HasPrefix(line, "PasswordAuthentication") {
			if len(passwordAuthentication) > 0 {
				line = passwordAuthentication
				passwordAuthentication = ""
			}
		} else if strings.HasPrefix(line, "Port") {
			if len(port) > 0 {
				line = port
				port = ""
			}
		}
		lines[i] = line
	}
	if len(permitRootLogin) > 0 {
		lines = append(lines, permitRootLogin)
	}
	if len(passwordAuthentication) > 0 {
		lines = append(lines, passwordAuthentication)
	}
	if len(port) > 0 {
		lines = append(lines, port)
	}
	return lines
}
