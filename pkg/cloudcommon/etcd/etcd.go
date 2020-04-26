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

package etcd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/seclib2"
)

var (
	ErrNoSuchKey = errors.New("No such key")
)

type SEtcdClient struct {
	client          *clientv3.Client
	requestTimeout  time.Duration
	leaseTtlTimeout int

	namespace string

	leaseId            clientv3.LeaseID
	onKeepaliveFailure func()
	leaseLiving        bool

	watchers map[string]*SEtcdWatcher
}

func defaultOnKeepAliveFailed() {
	log.Fatalf("etcd keepalive failed")
}

func NewEtcdClient(opt *SEtcdOptions, onKeepaliveFailure func()) (*SEtcdClient, error) {
	var err error
	var tlsConfig *tls.Config

	if opt.EtcdEnabldSsl {
		if opt.TLSConfig == nil {
			if len(opt.EtcdSslCaCertfile) > 0 {
				tlsConfig, err = seclib2.InitTLSConfigWithCA(
					opt.EtcdSslCertfile, opt.EtcdSslKeyfile, opt.EtcdSslCaCertfile)
			} else {
				tlsConfig, err = seclib2.InitTLSConfig(opt.EtcdSslCertfile, opt.EtcdSslKeyfile)
			}
			if err != nil {
				log.Errorf("init tls config fail %s", err)
				return nil, err
			}
		} else {
			tlsConfig = opt.TLSConfig
		}
	}

	etcdClient := &SEtcdClient{}
	if onKeepaliveFailure == nil {
		onKeepaliveFailure = defaultOnKeepAliveFailed
	}
	etcdClient.onKeepaliveFailure = onKeepaliveFailure

	timeoutSeconds := opt.EtcdTimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 5
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   opt.EtcdEndpoint,
		DialTimeout: time.Duration(timeoutSeconds) * time.Second,
		Username:    opt.EtcdUsername,
		Password:    opt.EtcdPassword,
		TLS:         tlsConfig,

		DialOptions: []grpc.DialOption{
			grpc.WithBlock(),
		},
	})
	if err != nil {
		return nil, err
	}

	etcdClient.client = cli

	timeoutSeconds = opt.EtcdRequestTimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 2
	}
	etcdClient.requestTimeout = time.Duration(timeoutSeconds) * time.Second

	timeoutSeconds = opt.EtcdLeaseExpireSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 5
	}

	etcdClient.leaseTtlTimeout = timeoutSeconds

	etcdClient.watchers = make(map[string]*SEtcdWatcher)

	etcdClient.namespace = opt.EtcdNamspace

	err = etcdClient.startSession()
	if err != nil {
		if e := etcdClient.Close(); e != nil {
			log.Errorf("etcd client close failed %s", e)
		}
		return nil, err
	}
	return etcdClient, nil
}

func (cli *SEtcdClient) Close() error {
	if cli.client != nil {
		err := cli.client.Close()
		if err != nil {
			return err
		}
		cli.client = nil
	}
	return nil
}

func (cli *SEtcdClient) GetClient() *clientv3.Client {
	return cli.client
}

func (cli *SEtcdClient) startSession() error {
	ctx := context.TODO()

	resp, err := cli.client.Grant(ctx, int64(cli.leaseTtlTimeout))
	if err != nil {
		return err
	}
	cli.leaseId = resp.ID

	ch, err := cli.client.KeepAlive(ctx, cli.leaseId)
	if err != nil {
		return err
	}
	cli.leaseLiving = true

	go func() {
		for {
			ka := <-ch
			if ka == nil {
				cli.leaseLiving = false
				log.Errorf("fail to keepalive sessoin")
				if cli.onKeepaliveFailure != nil {
					cli.onKeepaliveFailure()
				}
				break
			} else {
				log.Debugf("etcd session %d keepalive ttl: %d", ka.ID, ka.TTL)
			}
		}
	}()

	return nil
}

func (cli *SEtcdClient) RestartSession() error {
	if cli.leaseLiving {
		return errors.New("session is living, can't restart")
	}
	return cli.startSession()
}

func (cli *SEtcdClient) getKey(key string) string {
	if len(cli.namespace) > 0 {
		return fmt.Sprintf("%s%s", cli.namespace, key)
	} else {
		return key
	}
}

func (cli *SEtcdClient) Put(ctx context.Context, key string, val string) error {
	return cli.put(ctx, key, val, false)
}

func (cli *SEtcdClient) PutSession(ctx context.Context, key string, val string) error {
	return cli.put(ctx, key, val, true)
}

func (cli *SEtcdClient) put(ctx context.Context, key string, val string, session bool) error {
	nctx, cancel := context.WithTimeout(ctx, cli.requestTimeout)
	defer cancel()

	key = cli.getKey(key)
	if session {
		_, err := cli.client.Put(nctx, key, val, clientv3.WithLease(cli.leaseId))
		return err
	} else {
		_, err := cli.client.Put(nctx, key, val)
		return err
	}
}

func (cli *SEtcdClient) Get(ctx context.Context, key string) ([]byte, error) {
	nctx, cancel := context.WithTimeout(ctx, cli.requestTimeout)
	defer cancel()

	key = cli.getKey(key)

	resp, err := cli.client.Get(nctx, key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrNoSuchKey
	}

	return resp.Kvs[0].Value, nil
}

type SEtcdKeyValue struct {
	Key   string
	Value []byte
}

func (cli *SEtcdClient) List(ctx context.Context, prefix string) ([]SEtcdKeyValue, error) {
	nctx, cancel := context.WithTimeout(ctx, cli.requestTimeout)
	defer cancel()

	prefix = cli.getKey(prefix)

	resp, err := cli.client.Get(nctx, prefix, clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		return nil, err
	}
	ret := make([]SEtcdKeyValue, len(resp.Kvs))
	for i := 0; i < len(resp.Kvs); i += 1 {
		ret[i] = SEtcdKeyValue{
			Key:   string(resp.Kvs[i].Key[len(cli.namespace):]),
			Value: resp.Kvs[i].Value,
		}
	}
	return ret, nil
}

type TEtcdCreateEventFunc func(key, value []byte)
type TEtcdModifyEventFunc func(key, oldvalue, value []byte)

type SEtcdWatcher struct {
	watcher clientv3.Watcher
	cancel  context.CancelFunc
}

func (w *SEtcdWatcher) Cancel() {
	w.watcher.Close()
	w.cancel()
}

func (cli *SEtcdClient) Watch(ctx context.Context, prefix string, onCreate TEtcdCreateEventFunc, onModify TEtcdModifyEventFunc) {
	_, ok := cli.watchers[prefix]
	if ok {
		return
	}

	watcher := clientv3.NewWatcher(cli.client)
	nctx, cancel := context.WithCancel(ctx)

	cli.watchers[prefix] = &SEtcdWatcher{
		watcher: watcher,
		cancel:  cancel,
	}

	prefix = cli.getKey(prefix)

	rch := watcher.Watch(nctx, prefix, clientv3.WithPrefix(), clientv3.WithPrevKV())
	go func() {
		for wresp := range rch {
			for _, ev := range wresp.Events {
				if ev.PrevKv == nil {
					onCreate(ev.Kv.Key[len(cli.namespace):], ev.Kv.Value)
				} else {
					onModify(ev.Kv.Key[len(cli.namespace):], ev.PrevKv.Value, ev.Kv.Value)
				}
			}
		}
		log.Infof("stop watching %s", prefix)
	}()
}

func (cli *SEtcdClient) Unwatch(prefix string) {
	watcher, ok := cli.watchers[prefix]
	if ok {
		log.Debugf("unwatch %s", prefix)
		watcher.Cancel()
		delete(cli.watchers, prefix)
	} else {
		log.Debugf("prefix %s not watched!!", prefix)
	}
}

func (cli *SEtcdClient) Delete(ctx context.Context, key string) ([]byte, error) {
	nctx, cancel := context.WithTimeout(ctx, cli.requestTimeout)
	defer cancel()

	key = cli.getKey(key)

	dresp, err := cli.client.Delete(nctx, key, clientv3.WithPrevKV())
	if err != nil {
		return nil, err
	}

	if dresp.Deleted == 1 {
		return dresp.PrevKvs[0].Value, nil
	} else {
		return nil, nil
	}
}
