package ssh

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/log"
)

type ClientConfig struct {
	Username   string
	Password   string
	Host       string
	Port       int
	PrivateKey string
}

func parsePrivateKey(keyBuff string) (ssh.Signer, error) {
	return ssh.ParsePrivateKey([]byte(keyBuff))
}

func (conf ClientConfig) ToSshConfig() (*ssh.ClientConfig, error) {
	cliConfig := &ssh.ClientConfig{
		User:            conf.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	auths := make([]ssh.AuthMethod, 0)
	if conf.Password != "" {
		auths = append(auths, ssh.Password(conf.Password))
	}
	if conf.PrivateKey != "" {
		signer, err := parsePrivateKey(conf.PrivateKey)
		if err != nil {
			return nil, err
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	cliConfig.Auth = auths
	return cliConfig, nil
}

func (conf ClientConfig) Connect() (*ssh.Client, error) {
	cliConfig, err := conf.ToSshConfig()
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	client, err := ssh.Dial("tcp", addr, cliConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type Client struct {
	config ClientConfig
	client *ssh.Client
}

func (conf ClientConfig) NewClient() (*Client, error) {
	cli, err := conf.Connect()
	if err != nil {
		return nil, err
	}
	return &Client{
		config: conf,
		client: cli,
	}, nil
}

func NewClient(
	host string,
	port int,
	username string,
	password string,
	privateKey string,
) (*Client, error) {
	config := &ClientConfig{
		Host:       host,
		Port:       port,
		Username:   username,
		Password:   password,
		PrivateKey: privateKey,
	}
	return config.NewClient()
}

func (s *Client) GetConfig() ClientConfig {
	return s.config
}

func (s *Client) Run(cmds ...string) ([]string, error) {
	ret := []string{}
	for _, cmd := range cmds {
		session, err := s.client.NewSession()
		if err != nil {
			return nil, err
		}
		defer session.Close()
		log.Debugf("Run command: %q", cmd)
		var stdOut bytes.Buffer
		var stdErr bytes.Buffer
		session.Stdout = &stdOut
		session.Stderr = &stdErr
		err = session.Run(cmd)
		//out, err := session.CombinedOutput(cmd)
		if err != nil {
			log.Errorf("Command: %q, Error output: %s", cmd, stdErr.String())
			return nil, fmt.Errorf("%q error: %v, Stderr: %s", cmd, err, stdErr.Bytes())
		}
		ret = append(ret, ParseOutput(stdOut.Bytes())...)
	}

	return ret, nil
}

func ParseOutput(output []byte) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func (s *Client) Close() {
	s.client.Close()
}
