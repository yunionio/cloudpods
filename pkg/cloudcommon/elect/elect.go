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

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
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
	subscribers []chan ElectEvent
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
			grpc.WithTimeout(500 * time.Millisecond),
		},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, errors.Wrap(err, "new etcd client")
	}
	elect := &Elect{
		cli:   cli,
		path:  config.LockPrefix + "/" + key,
		ttl:   config.LockTTL,
		mutex: &sync.Mutex{},
	}
	return elect, nil
}

func (elect *Elect) Stop() {
	elect.stopFunc()
}

func (elect *Elect) Start(ctx context.Context) {
	ctx, elect.stopFunc = context.WithCancel(ctx)

	prev := ElectEventLost
	for {
		select {
		case <-ctx.Done():
			log.Infof("elect bye")
			return
		default:
			now := ElectEventWin
			ticket, err := elect.do(ctx)
			if err != nil {
				ticket.tearup(ctx)
				now = ElectEventLost
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

func (elect *Elect) Subscribe(ch chan ElectEvent) {
	elect.mutex.Lock()
	defer elect.mutex.Unlock()
	elect.subscribers = append(elect.subscribers, ch)
}

func (elect *Elect) notify(ctx context.Context, ev ElectEvent) {
	elect.mutex.Lock()
	defer elect.mutex.Unlock()

	for _, ch := range elect.subscribers {
		select {
		case ch <- ev:
		case <-ctx.Done():
			return
		default:
		}
	}
}

type ElectEvent int

const (
	ElectEventWin ElectEvent = iota
	ElectEventLost
)

func (ev ElectEvent) String() string {
	switch ev {
	case ElectEventWin:
		return "win"
	case ElectEventLost:
		return "lost"
	default:
		return "unexpected"
	}
}
