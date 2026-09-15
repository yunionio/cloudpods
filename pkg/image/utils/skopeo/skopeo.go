package skopeo

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type CopyParams struct {
	SrcTLSVerify bool
	SrcUsername  string
	SrcPassword  string
	SrcPath      string
	TargetPath   string
}

type ISkopeo interface {
	Copy(params *CopyParams) error
}

type skopeo struct{}

func NewSkopeo() ISkopeo {
	return &skopeo{}
}

func (s *skopeo) GetCommand(args ...string) string {
	cmd := []string{"/usr/bin/skopeo"}
	cmd = append(cmd, args...)
	return strings.Join(cmd, " ")
}

func (s *skopeo) getCopyCommand(params *CopyParams) string {
	args := []string{"copy"}
	if params.SrcTLSVerify {
		args = append(args, "--src-tls-verify=true")
	} else {
		args = append(args, "--src-tls-verify=false")
	}
	if params.SrcUsername != "" {
		args = append(args, fmt.Sprintf("--src-username %s", params.SrcUsername))
	}
	if params.SrcPassword != "" {
		args = append(args, fmt.Sprintf("--src-password %s", params.SrcPassword))
	}
	args = append(args, fmt.Sprintf("docker://%s", params.SrcPath))
	args = append(args, fmt.Sprintf("docker-archive:%s:%s", params.TargetPath, params.SrcPath))
	return s.GetCommand(args...)
}

func (s *skopeo) Copy(params *CopyParams) error {
	if params.SrcPath == "" {
		return errors.Error("src path is empty")
	}
	if params.TargetPath == "" {
		return errors.Error("target path is empty")
	}

	cmd := s.getCopyCommand(params)
	log.Debugf("execute cmd: %s", cmd)

	return s.executeCmd(cmd)
}

func (s *skopeo) executeCmd(cmd string) error {
	parts := strings.Split(cmd, " ")
	if len(parts) <= 2 {
		return errors.Errorf("invalid command: %q", cmd)
	}
	return s.execute(parts[0], parts[1:]...)
}

func (s *skopeo) execute(command string, args ...string) error {
	execDoneChan := make(chan int8)
	defer close(execDoneChan)

	log.Debugf("execute command: %s %v", command, args)

	timeInit := time.Now()
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debugf("execute cmd: %s \n output: %s", cmd.String(), output)
		return errors.Wrapf(err, "command output: %s", output)
	}

	elapsedTime := time.Since(timeInit)
	log.Infof("%s command duration: %s", command, elapsedTime.String())
	return nil
}
