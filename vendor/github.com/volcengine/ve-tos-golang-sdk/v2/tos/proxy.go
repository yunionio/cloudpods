package tos

import (
	"net/url"
	"strconv"
)

type Proxy struct {
	proxyHost     string
	proxyUserName string
	proxyPassword string
	proxyPort     int
}

func (p *Proxy) getRawUrl() string {
	return p.proxyHost + ":" + strconv.Itoa(p.proxyPort)
}

func (p *Proxy) Url() *url.URL {
	proxyURL, _ := url.Parse(p.getRawUrl())
	if p.proxyUserName != "" && p.proxyPassword != "" {
		proxyURL.User = url.UserPassword(p.proxyUserName, p.proxyPassword)
	} else if p.proxyUserName != "" {
		proxyURL.User = url.User(p.proxyUserName)
	}
	return proxyURL
}

func NewProxy(proxyHost string, proxyPort int) (*Proxy, error) {
	proxyUrl, err := url.Parse(proxyHost)
	if err != nil {
		return nil, ProxyUrlInvalid.withCause(err)
	}
	if proxyUrl.Scheme == "" {
		proxyHost = "http://" + proxyHost
	}
	if proxyUrl.Scheme == "https" {
		return nil, ProxyNotSupportHttps
	}

	if _, err := url.Parse(proxyHost + ":" + strconv.Itoa(proxyPort)); err != nil {
		if err != nil {
			return nil, err
		}
	}
	return &Proxy{
		proxyHost: proxyHost,
		proxyPort: proxyPort,
	}, nil
}

func (p *Proxy) WithAuth(username string, password string) {
	p.proxyUserName = username
	p.proxyPassword = password
}
