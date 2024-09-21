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

package metadata

// NOTE keep imports minimal.  DO NOT IMPORT guestman
import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	identity_api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/proxy"
)

func Start(app *appsrv.Application, s *Service) {
	s.addHandler(app)
	addr := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
	log.Infof("Start metadata service on http://%s", addr)
	app.ListenAndServeWithoutCleanup(addr, "", "")
}

type DescGetter interface {
	Get(ip string) (guestDesc *desc.SGuestDesc)
}

type DescGetterFunc func(ip string) (guestDesc *desc.SGuestDesc)

func (f DescGetterFunc) Get(ip string) (guestDesc *desc.SGuestDesc) {
	return f(ip)
}

type Service struct {
	Address string
	Port    int

	DescGetter DescGetter
}

func (s *Service) getGuestDesc(r *http.Request) (guestDesc *desc.SGuestDesc) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		panic(errors.Wrapf(err, "SplitHostPort %s", r.RemoteAddr))
	}
	guestDesc = s.DescGetter.Get(ip)
	return
}

func (s *Service) monitorPrefix() string {
	return "/monitor"
}

func (s *Service) addHandler(app *appsrv.Application) {
	prefix := ""

	for _, method := range []string{"GET", "HEAD"} {
		app.AddHandler(method, fmt.Sprintf("%s/<version:%s>",
			prefix, `(latest|\d{4}-\d{2}-\d{2})`), s.versionOnly)
	}

	for _, method := range []string{"GET", "HEAD"} {
		app.AddHandler(method, fmt.Sprintf("%s/<version:%s>/user-data",
			prefix, `(latest|\d{4}-\d{2}-\d{2})`), s.userData)
		app.AddHandler(method, fmt.Sprintf("%s/<version:%s>/meta-data",
			prefix, `(latest|\d{4}-\d{2}-\d{2})`), s.metaData)
	}

	app.AddReverseProxyHandler(s.monitorPrefix(), s.monitorReverseEndpoint(), s.requestManipulator)
}

func (s *Service) versionOnly(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	hostutils.Response(ctx, w, strings.Join([]string{"meta-data", "user-data"}, "\n"))
}

func (s *Service) userData(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	guestDesc := s.getGuestDesc(r)
	if guestDesc == nil {
		hostutils.Response(ctx, w, "")
		return
	}

	guestUserData := guestDesc.UserData
	if guestUserData == "" {
		hostutils.Response(ctx, w, "")
		return
	}

	userDataDecoded, err := base64.StdEncoding.DecodeString(guestUserData)
	if err != nil {
		log.Errorf("Error format user_data %s, %s", guestDesc.Uuid, guestUserData)
		hostutils.Response(ctx, w, "")
		return
	}
	hostutils.Response(ctx, w, string(userDataDecoded))
}

func (s *Service) metaData(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	guestDesc := s.getGuestDesc(r)
	if guestDesc == nil {
		hostutils.Response(ctx, w, "")
		return
	}

	req := appsrv.SplitPath(r.URL.Path)[2:]

	if len(req) == 0 {
		resNames := []string{
			"ami-launch-index",
			"block-device-mapping/", "hostname",
			"instance-id", "instance-type",
			"local-hostname",
			"local-ipv4",
			"local-ipv6",
			"mac",
			"public-hostname", "public-ipv4",
			"network_config/",
			"local-sub-ipv4s",
			//"amiid", "ami-manifest-path",
			//"instance-action", "kernel-id",
			//"ipv4-associations", "network/",
			//"placement/", "public-keys/",
			//"reservation-id", "security-groups", "password",
		}
		if guestDesc.Pubkey != "" {
			resNames = append(resNames, "public-keys/")
		}
		if guestDesc.Zone != "" {
			resNames = append(resNames, "placement/")
		}
		if guestDesc.Secgroup != "" {
			resNames = append(resNames, "security-groups/")
		}
		hostutils.Response(ctx, w, strings.Join(resNames, "\n"))
		return
	} else {
		resName := req[0]
		switch resName {
		case "public-keys":
			if guestDesc.Pubkey != "" {
				if len(req) == 1 {
					hostutils.Response(ctx, w, "0=my-public-key")
					return
				} else if len(req) == 2 {
					hostutils.Response(ctx, w, "openssh-key")
					return
				} else if len(req) == 3 {
					pubkey := guestDesc.Pubkey
					hostutils.Response(ctx, w, pubkey)
					return
				}
			}
		case "hostname":
			hostutils.Response(ctx, w, guestDesc.Name)
			return
		case "public-hostname", "local-hostname":
			hostutils.Response(ctx, w, guestDesc.Hostname)
			return
		case "instance-id":
			hostutils.Response(ctx, w, guestDesc.Uuid)
			return
		case "instance-type":
			flavor := guestDesc.Flavor
			if flavor == "" {
				flavor = "customized"
			}
			hostutils.Response(ctx, w, flavor)
			return
		case "mac":
			macs := make([]string, 0)
			guestNics := guestDesc.Nics
			for _, nic := range guestNics {
				macs = append(macs, nic.Mac)
			}
			hostutils.Response(ctx, w, strings.Join(macs, "\n"))
			return
		case "local-ipv4":
			ips := make([]string, 0)
			guestNics := guestDesc.Nics
			for _, nic := range guestNics {
				ips = append(ips, nic.Ip)
			}
			hostutils.Response(ctx, w, strings.Join(ips, "\n"))
			return
		case "local-ipv6":
			ips := make([]string, 0)
			guestNics := guestDesc.Nics
			for _, nic := range guestNics {
				if len(nic.Ip6) > 0 {
					ips = append(ips, nic.Ip6)
				}
			}
			hostutils.Response(ctx, w, strings.Join(ips, "\n"))
			return
		case "local-sub-ipv4s":
			ips := make([]string, 0)
			guestNics := guestDesc.Nics
			for _, nic := range guestNics {
				nas, _ := nic.Networkaddresses.GetArray()
				for _, na := range nas {
					if typ, _ := na.GetString("type"); typ == "sub_ip" {
						ip, _ := na.GetString("ip_addr")
						if ip != "" {
							ips = append(ips, ip)
						}
					}
				}
			}
			hostutils.Response(ctx, w, strings.Join(ips, "\n"))
			return
		case "public-ipv4":
			ips := make([]string, 0)
			guestNics := guestDesc.Nics
			for _, nic := range guestNics {
				ipv4, _ := netutils.NewIPV4Addr(nic.Ip)
				if !netutils.IsPrivate(ipv4) {
					ips = append(ips, nic.Ip)
				}
			}
			hostutils.Response(ctx, w, strings.Join(ips, "\n"))
			return
		case "placement":
			if guestDesc.Zone != "" {
				if len(req) == 1 {
					hostutils.Response(ctx, w, "availability-zone")
					return
				} else if len(req) == 2 && req[1] == "availability-zone" {
					hostutils.Response(ctx, w, guestDesc.Zone)
					return
				}
			}
		case "security-groups":
			if guestDesc.Secgroup != "" {
				hostutils.Response(ctx, w, guestDesc.Secgroup)
				return
			}
		case "ami-launch-index":
			hostutils.Response(ctx, w, "0")
			return
		case "network_config":
			if len(req) == 1 {
				hostutils.Response(ctx, w,
					strings.Join([]string{"name", "content_path"}, "\n"))
				return
			} else if len(req) == 2 {
				if req[1] == "name" {
					hostutils.Response(ctx, w, "network_config")
					return
				} else if req[1] == "content_path" {
					hostutils.Response(ctx, w, "content/0001")
					return
				}
			}
		case "block-device-mapping":
			guestDisks := guestDesc.Disks
			swapDisks := make([]string, 0)
			dataDisk := make([]string, 0)
			for _, d := range guestDisks {
				if d.Fs == "swap" {
					swapDisks = append(swapDisks, strconv.Itoa(int(d.Index)))
				} else {
					dataDisk = append(dataDisk, strconv.Itoa(int(d.Index)))
				}
			}
			if len(req) == 1 {
				devs := []string{"root"}
				if len(swapDisks) > 0 {
					devs = append(devs, "swap")
				}
				if len(dataDisk) > 0 {
					for i := 0; i < len(dataDisk); i++ {
						devs = append(devs, fmt.Sprintf("ephemeral%d", i+1))
					}
				}
				hostutils.Response(ctx, w, strings.Join(devs, "\n"))
				return
			} else if len(req) == 2 {
				devs := []string{}
				if req[1] == "root" {
					devs = append(devs, "/dev/root")
				} else if req[1] == "swap" {
					for i := 0; i < len(swapDisks); i++ {
						idx, _ := strconv.Atoi(swapDisks[i])
						devs = append(devs, fmt.Sprintf("/dev/vd%c1", 'a'+idx))
					}
				} else if strings.HasPrefix(req[1], "ephemeral") || strings.HasPrefix(req[1], "ebs") {
					for i := 0; i < len(dataDisk); i++ {
						idx, _ := strconv.Atoi(dataDisk[i])
						devs = append(devs, fmt.Sprintf("/dev/vd%c1", 'a'+idx))
					}
				}
				hostutils.Response(ctx, w, strings.Join(devs, "\n"))
				return
			}
		}
	}
	hostutils.Response(ctx, w, httperrors.NewNotFoundError("Resource not handled"))
}

func (s *Service) monitorReverseEndpoint() *proxy.SEndpointFactory {
	f := func(ctx context.Context, r *http.Request) (string, error) {
		guestDesc := s.getGuestDesc(r)
		if guestDesc == nil {
			return "", httperrors.NewNotFoundError("vm not found")
		}
		s := auth.GetAdminSession(ctx, options.HostOptions.Region)
		srcURL, err := tsdb.GetDefaultServiceSourceURL(s, identity_api.EndpointInterfaceInternal)
		if err != nil {
			return "", errors.Wrap(err, "monitorReverseEndpoint get tsdb url")
		}
		return srcURL, nil
	}
	return proxy.NewEndpointFactory(f, "monitorService")
}

func (s *Service) requestManipulator(ctx context.Context, r *http.Request) (*http.Request, error) {
	path := r.URL.Path[len(s.monitorPrefix()):]
	log.Debugf("Path: %s => %s", r.URL.Path, path)
	r.URL = &url.URL{
		Path:     path,
		RawQuery: r.URL.RawQuery,
		Fragment: r.URL.Fragment,
	}
	return r, nil
}
