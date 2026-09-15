package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/regclient/regclient/internal/conffile"
)

const (
	dockerEnv       = "DOCKER_CONFIG"
	dockerDir       = ".docker"
	dockerConfFile  = "config.json"
	dockerHelperPre = "docker-credential-"
)

// methods to parse user's docker config.json file

// dockerConfig is used to parse the ~/.docker/config.json
type dockerConfig struct {
	AuthConfigs       map[string]dockerAuthConfig  `json:"auths"`
	HTTPHeaders       map[string]string            `json:"HttpHeaders,omitempty"`
	DetachKeys        string                       `json:"detachKeys,omitempty"`
	CredentialsStore  string                       `json:"credsStore,omitempty"`
	CredentialHelpers map[string]string            `json:"credHelpers,omitempty"`
	Proxies           map[string]dockerProxyConfig `json:"proxies,omitempty"`
}

// dockerProxyConfig contains proxy configuration settings
type dockerProxyConfig struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
	FTPProxy   string `json:"ftpProxy,omitempty"`
	AllProxy   string `json:"allProxy,omitempty"`
}

// dockerAuthConfig contains the auths
type dockerAuthConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`

	ServerAddress string `json:"serveraddress,omitempty"`

	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `json:"identitytoken,omitempty"`

	// RegistryToken is a bearer token to be sent to a registry
	RegistryToken string `json:"registrytoken,omitempty"`
}

func DockerLoad() ([]Host, error) {
	cf := conffile.New(conffile.WithDirName(dockerDir, dockerConfFile), conffile.WithEnvDir(dockerEnv, dockerConfFile))
	return dockerParse(cf)
}

// parse from io.Reader to []Host
func dockerParse(cf *conffile.File) ([]Host, error) {
	rdr, err := cf.Open()
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return []Host{}, nil
	} else if err != nil {
		return nil, err
	}
	defer rdr.Close()
	dc := dockerConfig{}
	if err := json.NewDecoder(rdr).Decode(&dc); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	hosts := []Host{}
	for name, auth := range dc.AuthConfigs {
		h, err := dockerAuthToHost(name, dc, auth)
		if err != nil {
			continue
		}
		hosts = append(hosts, h)
	}
	// also include default entries for credential helpers
	for name, helper := range dc.CredentialHelpers {
		h := HostNewName(name)
		h.CredHelper = dockerHelperPre + helper
		if _, ok := dc.AuthConfigs[h.Name]; ok {
			continue // skip fields with auth config
		}
		hosts = append(hosts, *h)
	}
	// add credStore entries
	if dc.CredentialsStore != "" {
		ch := newCredHelper(dockerHelperPre+dc.CredentialsStore, map[string]string{})
		csHosts, err := ch.list()
		if err == nil {
			hosts = append(hosts, csHosts...)
		}
	}
	return hosts, nil
}

func dockerAuthToHost(name string, conf dockerConfig, auth dockerAuthConfig) (Host, error) {
	helper := ""
	if conf.CredentialHelpers != nil && conf.CredentialHelpers[name] != "" {
		helper = dockerHelperPre + conf.CredentialHelpers[name]
	}
	// parse base64 auth into user/pass
	if auth.Auth != "" {
		var err error
		auth.Username, auth.Password, err = decodeAuth(auth.Auth)
		if err != nil {
			return Host{}, err
		}
	}
	if (auth.Username == "" || auth.Password == "") && auth.IdentityToken == "" && helper == "" {
		return Host{}, fmt.Errorf("no credentials found for %s", name)
	}

	h := HostNewName(name)
	h.User = auth.Username
	h.Pass = auth.Password
	h.Token = auth.IdentityToken
	h.CredHelper = helper
	return *h, nil
}

func decodeAuth(authStr string) (string, string, error) {
	if authStr == "" {
		return "", "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(authStr)
	if err != nil {
		return "", "", err
	}
	userPass := strings.SplitN(string(decoded), ":", 2)
	if len(userPass) != 2 {
		return "", "", fmt.Errorf("invalid auth configuration file")
	}
	return userPass[0], strings.Trim(userPass[1], "\x00"), nil
}
