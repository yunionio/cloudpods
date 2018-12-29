package main

import (
	"context"
	"time"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
)

func main() {
	opt := etcd.SEtcdOptions{}
	opt.EtcdEndpoint = []string{"127.0.0.1:2379"}

	cli, err := etcd.NewEtcdClient(&opt)
	if err != nil {
		log.Errorf("etcd init fail %s", err)
		return
	}
	defer cli.Close()

	ctx := context.Background()

	cli.Watch(ctx, "foo",
		func(key, val []byte) {
			log.Infof("new key %s %s", string(key), string(val))
		},
		func(key, oval, nval []byte) {
			log.Infof("modify key %s %s => %s", string(key), string(oval), string(nval))
		},
	)

	for _, k := range []string{"/foo", "/foo.tmp"} {
		val, err := cli.Get(ctx, k)
		if err != nil && err != etcd.ErrNoSuchKey {
			log.Errorf("get %s fail %s", k, err)
			return
		}

		log.Infof("%s val is %s", k, val)
	}

	err = cli.Put(ctx, "/foo", "bar")
	if err != nil {
		log.Errorf("%s", err)
		return
	}

	err = cli.PutSession(ctx, "foo.tmp", "bar.tmp")
	if err != nil {
		log.Errorf("%s", err)
	}

	err = cli.Put(ctx, "foo", "bar2")
	if err != nil {
		log.Errorf("%s", err)
		return
	}

	time.Sleep(time.Second * 10)

	log.Debugf("unwatch foo")
	cli.Unwatch("foo")

	time.Sleep(time.Second * 10)

	err = cli.Put(ctx, "foo", "bar3")
	if err != nil {
		log.Errorf("%s", err)
		return
	}

	resp, err := cli.Get(ctx, "foo")
	if err != nil {
		log.Errorf("%s", err)
		return
	}
	log.Infof("value is %s", string(resp))

	kvs, err := cli.List(ctx, "")
	if err != nil {
		log.Errorf("list error %s", err)
		return
	}

	for _, kv := range kvs {
		log.Infof("%s : %v", string(kv.Key), string(kv.Value))
	}
}
