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

package elect

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"google.golang.org/grpc"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type EtcdConfig struct {
	Endpoints []string
	Username  string
	Password  string
	TLS       *tls.Config

	LockTTL    int
	LockPrefix string

	opts *options.DBOptions
}

func NewEtcdConfigFromDBOptions(opts *options.DBOptions) (*EtcdConfig, error) {
	tlsCfg, err := opts.GetEtcdTLSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "etcd tls config")
	}

	config := &EtcdConfig{
		Endpoints:  opts.EtcdEndpoints,
		Username:   opts.EtcdUsername,
		Password:   opts.EtcdPassword,
		LockTTL:    opts.EtcdLockTTL,
		LockPrefix: opts.EtcdLockPrefix,
		TLS:        tlsCfg,
	}
	if config.LockTTL <= 0 {
		config.LockTTL = 5
	}
	return config, nil
}

type Elect struct {
	cli  *clientv3.Client
	path string
	ttl  int

	stopFunc context.CancelFunc

	config *EtcdConfig

	mutex       *sync.Mutex
	latestEv    electEvent
	subscribers []chan electEvent
}

type ticket struct {
	session *concurrency.Session
	mutex   *concurrency.Mutex
}

func (t *ticket) tearup(ctx context.Context) {
	if t.session != nil {
		t.session.Close()
		t.session = nil
	}
}

func NewElect(config *EtcdConfig, key string) (*Elect, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: config.Endpoints,
		Username:  config.Username,
		Password:  config.Password,
		TLS:       config.TLS,

		DialOptions: []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTimeout(3000 * time.Millisecond),
		},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, errors.Wrap(err, "new etcd client")
	}
	elect := &Elect{
		cli:  cli,
		path: config.LockPrefix + "/" + key,
		ttl:  config.LockTTL,

		mutex:    &sync.Mutex{},
		latestEv: electEventInit,
	}
	return elect, nil
}

func (elect *Elect) Stop() {
	elect.stopFunc()
}

func (elect *Elect) Start(ctx context.Context) {
	ctx, elect.stopFunc = context.WithCancel(ctx)

	prev := electEventLost
	for {
		select {
		case <-ctx.Done():
			log.Infof("elect bye")
			return
		default:
			now := electEventWin
			ticket, err := elect.do(ctx)
			if err != nil {
				ticket.tearup(ctx)
				now = electEventLost
				log.Errorf("elect error: %v", err)
			}
			if now != prev {
				log.Infof("notify elect event: %s -> %s", prev, now)
				prev = now
				elect.notify(ctx, now)
			}
			if err == nil {
				select {
				case <-ctx.Done():
					ticket.tearup(ctx)
				case <-ticket.session.Done():
				}
			} else {
				time.Sleep(3 * time.Second)
			}
		}
	}
}

// do joins an election.  the 1st return argument ticket must always be non-nil
func (elect *Elect) do(ctx context.Context) (*ticket, error) {
	r := &ticket{}

	sess, err := concurrency.NewSession(
		elect.cli,
		concurrency.WithTTL(elect.ttl),
	)
	if err != nil {
		return r, err
	}
	r.session = sess

	em := concurrency.NewMutex(sess, elect.path)
	if err := em.Lock(ctx); err != nil {
		return r, err
	}
	r.mutex = em

	return r, err
}

func (elect *Elect) subscribe(ctx context.Context, ch chan electEvent) {
	elect.mutex.Lock()
	defer elect.mutex.Unlock()
	elect.subscribers = append(elect.subscribers, ch)
	if ev := elect.latestEv; ev != electEventInit {
		elect.notifyOne(ctx, ev, ch)
	}
}

func (elect *Elect) notifyOne(ctx context.Context, ev electEvent, ch chan electEvent) {
	sent := false
	select {
	case ch <- ev:
		sent = true
	case <-ctx.Done():
	default:
	}
	if !sent {
		log.Errorf("elect event '%s' missed by %#v", ev, ch)
	}
}

func (elect *Elect) notify(ctx context.Context, ev electEvent) {
	elect.mutex.Lock()
	defer elect.mutex.Unlock()

	elect.latestEv = ev
	for _, ch := range elect.subscribers {
		elect.notifyOne(ctx, ev, ch)
	}
}

func (elect *Elect) SubscribeWithAction(ctx context.Context, onWin, onLost func()) {
	go func() {
		ch := make(chan electEvent, 3)
		var ev electEvent
		elect.subscribe(ctx, ch)
		for {
			select {
			case ev = <-ch:
			case <-ctx.Done():
				return
			}
		drain:
			for {
				select {
				case ev = <-ch:
					continue
				default:
					break drain
				}
			}
			log.Infof("elect event %s", ev)
			switch ev {
			case electEventWin:
				onWin()
			case electEventLost:
				onLost()
			}
		}
	}()
}

type electEvent int

const (
	electEventWin electEvent = iota
	electEventLost
	electEventInit
)

func (ev electEvent) String() string {
	switch ev {
	case electEventWin:
		return "win"
	case electEventLost:
		return "lost"
	case electEventInit:
		return "init"
	default:
		return "unexpected"
	}
}
