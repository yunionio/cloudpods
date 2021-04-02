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

package ssh

import (
	"context"

	ssh_util "yunion.io/x/onecloud/pkg/util/ssh"
)

type epClientSet struct {
	cc      ssh_util.ClientConfig
	clients []*Client

	mark bool
}

func (epcs *epClientSet) clearMark() {
	epcs.mark = false
}

func (epcs *epClientSet) setMark() {
	epcs.mark = true
}

func (epcs *epClientSet) getMark() bool {
	return epcs.mark
}

func (epcs *epClientSet) stop(ctx context.Context) {
	for _, client := range epcs.clients {
		client.Stop(ctx)
	}
}

type epClients map[string]*epClientSet // key: epKey

type ClientSet struct {
	epClients epClients
}

func NewClientSet() *ClientSet {
	cs := &ClientSet{
		epClients: epClients{},
	}
	return cs
}

func (cs *ClientSet) ClearAllMark() {
	for _, epcs := range cs.epClients {
		epcs.clearMark()
	}
}

func (cs *ClientSet) ResetIfChanged(ctx context.Context, epKey string, cc ssh_util.ClientConfig) bool {
	epcs, ok := cs.epClients[epKey]
	if ok {
		if epcs.cc != cc {
			epcs.stop(ctx)
			delete(cs.epClients, epKey)
			cs.AddIfNotExist(ctx, epKey, cc)
			return true
		}
		epcs.setMark()
	}
	return false
}

func (cs *ClientSet) AddIfNotExist(ctx context.Context, epKey string, cc ssh_util.ClientConfig) bool {
	epcs, ok := cs.epClients[epKey]
	if !ok {
		epcs := &epClientSet{
			cc: cc,
		}
		epcs.setMark()
		cs.epClients[epKey] = epcs
		return true
	}
	epcs.setMark()
	return false
}

func (cs *ClientSet) ResetUnmarked(ctx context.Context) {
	for epKey, epcs := range cs.epClients {
		if !epcs.getMark() {
			epcs.stop(ctx)
			delete(cs.epClients, epKey)
		}
	}
}

func (cs *ClientSet) ForwardKeySet() ForwardKeySet {
	fks := ForwardKeySet{}
	for epKey, epcs := range cs.epClients {
		for _, client := range epcs.clients {
			fks.addByPortMap(epKey, ForwardKeyTypeL, client.localForwards)
			fks.addByPortMap(epKey, ForwardKeyTypeR, client.remoteForwards)
		}
	}
	return fks
}

func (cs *ClientSet) LocalForward(ctx context.Context, epKey string, req LocalForwardReq) {
	client, created := cs.getOrCreateClient(epKey, ForwardKeyTypeL)
	if created {
		go client.Start(ctx)
	}
	client.LocalForward(ctx, req)
}

func (cs *ClientSet) RemoteForward(ctx context.Context, epKey string, req RemoteForwardReq) {
	client, created := cs.getOrCreateClient(epKey, ForwardKeyTypeR)
	if created {
		go client.Start(ctx)
	}
	client.RemoteForward(ctx, req)
}

func (cs *ClientSet) CloseForward(ctx context.Context, fk ForwardKey) {
	client := cs.getClient(fk.EpKey, fk.Type)
	if client == nil {
		return
	}
	switch fk.Type {
	case ForwardKeyTypeL:
		client.LocalForwardClose(ctx, LocalForwardReq{
			LocalAddr: fk.KeyAddr,
			LocalPort: fk.KeyPort,
		})
	case ForwardKeyTypeR:
		client.RemoteForwardClose(ctx, RemoteForwardReq{
			RemoteAddr: fk.KeyAddr,
			RemotePort: fk.KeyPort,
		})
	}
}

/*
func (cs *ClientSet) LocalForwardClose(ctx context.Context, epKey string, req LocalForwardReq) {
	client := cs.getClient(epKey, ForwardKeyTypeL)
	client.LocalForwardClose(ctx, req)
}

func (cs *ClientSet) RemoteForwardClose(ctx context.Context, epKey string, req RemoteForwardReq) {
	client := cs.getClient(epKey, ForwardKeyTypeR)
	client.RemoteForwardClose(ctx, req)
}
*/

func (cs *ClientSet) getOrCreateClient(epKey string, typ string) (*Client, bool) {
	return cs.getClient_(epKey, typ, true)
}

func (cs *ClientSet) getClient(epKey string, typ string) *Client {
	client, _ := cs.getClient_(epKey, typ, false)
	return client
}

func (cs *ClientSet) getClient_(epKey string, typ string, create bool) (*Client, bool) {
	var client *Client

	clients, ok := cs.epClients[epKey]
	if !ok || len(clients.clients) == 0 {
		if !ok || !create {
			return nil, false
		}
		client = NewClient(&clients.cc)
		clients.clients = append(clients.clients, client)
		cs.epClients[epKey] = clients
		return client, true
	} else {
		client = clients.clients[0]
		return client, false
	}
}
