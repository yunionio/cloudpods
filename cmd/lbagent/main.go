package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/lbagent"
)

func main() {
	opts := &lbagent.Options{}
	commonOpts := &opts.CommonOpts
	{
		cloudcommon.ParseOptions(opts, commonOpts, os.Args, "lbagent.conf")
		cloudcommon.InitAuth(commonOpts, func() {
			log.Infof("auth finished ok")
		})
	}
	if err := opts.ValidateThenInit(); err != nil {
		log.Fatalf("opts validate: %s", err)
	}

	var haproxyHelper *lbagent.HaproxyHelper
	var apiHelper *lbagent.ApiHelper
	var err error
	{
		haproxyHelper, err = lbagent.NewHaproxyHelper(opts)
		if err != nil {
			log.Fatalf("init haproxy helper failed: %s", err)
		}
	}
	{
		apiHelper, err = lbagent.NewApiHelper(opts)
		if err != nil {
			log.Fatalf("init api helper failed: %s", err)
		}
	}

	{
		wg := &sync.WaitGroup{}
		cmdChan := make(chan *lbagent.LbagentCmd) // internal
		ctx, cancelFunc := context.WithCancel(context.Background())
		ctx = context.WithValue(ctx, "wg", wg)
		ctx = context.WithValue(ctx, "cmdChan", cmdChan)
		wg.Add(2)
		go haproxyHelper.Run(ctx)
		go apiHelper.Run(ctx)

		go func() {
			sigChan := make(chan os.Signal)
			signal.Notify(sigChan, syscall.SIGINT)
			signal.Notify(sigChan, syscall.SIGTERM)
			sig := <-sigChan
			log.Infof("signal received: %s", sig)
			cancelFunc()
		}()
		wg.Wait()
	}
}
