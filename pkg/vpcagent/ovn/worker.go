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

package ovn

import (
	"context"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/apihelper"
	apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/ovsutils"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
	"yunion.io/x/onecloud/pkg/vpcagent/options"
	"yunion.io/x/onecloud/pkg/vpcagent/ovnutil"
	"yunion.io/x/onecloud/pkg/vpcagent/worker"
)

type Worker struct {
	opts *options.Options

	apih *apihelper.APIHelper
}

func NewWorker(opts *options.Options) worker.IWorker {
	modelSets := agentmodels.NewModelSets()
	apiOpts := &apihelper.Options{
		CommonOptions:           opts.CommonOptions,
		SyncIntervalSeconds:     opts.APISyncIntervalSeconds,
		RunDelayMilliseconds:    opts.APIRunDelayMilliseconds,
		ListBatchSize:           opts.APIListBatchSize,
		FetchFromComputeService: opts.FetchDataFromComputeService,
		IncludeDetails:          false,

		IncludeOtherCloudEnv: false,
	}
	apih, err := apihelper.NewAPIHelper(apiOpts, modelSets)
	if err != nil {
		return nil
	}
	w := &Worker{
		opts: opts,
		apih: apih,
	}
	return w
}

func (w *Worker) Start(ctx context.Context, app *appsrv.Application, prefix string) {
	wg := ctx.Value("wg").(*sync.WaitGroup)
	defer func() {
		log.Infoln("ovn: worker bye")
		wg.Done()
	}()

	wg.Add(1)
	go w.apih.Start(ctx, app, httputils.JoinPath(prefix, "api"))

	tickDuration := time.Duration(w.opts.OvnWorkerCheckInterval) * time.Second
	tick := time.NewTimer(tickDuration)
	defer tick.Stop()

	var mss *agentmodels.ModelSets
	for {
		select {
		case imss := <-w.apih.ModelSets():
			log.Infof("ovn: got new data from api helper")
			mss = imss.(*agentmodels.ModelSets)
			if err := w.run(ctx, mss); err != nil {
				log.Errorf("ovn: %v", err)
			}
		case <-tick.C:
			if mss != nil {
				log.Infof("ovn: tick check")
				if err := w.run(ctx, mss); err != nil {
					log.Errorf("ovn: %v", err)
				}
			}
			tick.Reset(tickDuration)
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) run(ctx context.Context, mss *agentmodels.ModelSets) (err error) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if panicErr, ok := panicVal.(runtime.Error); ok {
				err = errors.Wrap(panicErr, string(debug.Stack()))
			} else if panicErr, ok := panicVal.(error); ok {
				err = panicErr
			} else {
				panic(panicVal)
			}
		}
	}()

	dbUrl := w.opts.OvnNorthDatabase
	if db, err := ovsutils.NormalizeDbHost(dbUrl); err != nil {
		return err
	} else {
		dbUrl = db
	}

	ovnnbctl := ovnutil.NewOvnNbCtl(dbUrl)
	ovndb, err := DumpOVNNorthbound(ctx, ovnnbctl)
	if err != nil {
		return err
	}

	ovndb.Mark(ctx)
	for _, vpc := range mss.Vpcs {
		if vpc.Id == apis.DEFAULT_VPC_ID {
			continue
		}
		ovndb.ClaimVpc(ctx, vpc)
		if vpcHasEipgw(vpc) {
			ovndb.ClaimVpcEipgw(ctx, vpc)
		}
		for _, network := range vpc.Networks {
			ovndb.ClaimNetwork(ctx, network, w.opts)
			for _, guestnetwork := range network.Guestnetworks {
				if guestnetwork.Guest == nil {
					continue
				}

				if vpcHasDistgw(vpc) {
					var (
						guest   = guestnetwork.Guest
						network = guestnetwork.Network
						vpc     = network.Vpc
						host    = guest.Host
					)
					if host.OvnVersion == "" {
						// Just in case.  This should never happen
						log.Errorf("host %s(%s) of vpc guestnetwork (%s,%s) has no ovn support",
							host.Id, host.Name, guestnetwork.NetworkId, guestnetwork.IpAddr)
						continue
					}
					if host.OvnMappedIpAddr == "" {
						// trigger ovn mapped ip addr allocation
						// apiVersion := "v2"
						s := auth.GetAdminSession(ctx, w.opts.Region)
						j, err := mcclient_modules.Hosts.Update(s, host.Id, nil)
						if err != nil {
							log.Errorf("host %s(%s) dummy update err: %v", host.Id, host.Name, err)
							continue
						}
						j.Unmarshal(host) // update local copy in place
						if host.OvnMappedIpAddr == "" {
							log.Errorf("host %s(%s) has no mapped addr", host.Id, host.Name)
							continue
						}
					}

					ovndb.ClaimVpcHost(ctx, vpc, host)
				}
				ovndb.ClaimGuestnetwork(ctx, guestnetwork, w.opts)
			}
			for _, groupnetwork := range network.Groupnetworks {
				ovndb.ClaimGroupnetwork(ctx, groupnetwork)
			}
			for _, loadbalancerNetwork := range network.LoadbalancerNetworks {
				ovndb.ClaimLoadbalancerNetwork(ctx, loadbalancerNetwork)
			}
		}
		routes := resolveRoutes(vpc, mss)
		ovndb.ClaimRoutes(ctx, vpc, routes)
	}
	for _, vpc := range mss.Vpcs {
		if vpc.Id == apis.DEFAULT_VPC_ID {
			continue
		}
		ovndb.ClaimVpcGuestDnsRecords(ctx, vpc)
	}
	ovndb.ClaimDnsRecords(ctx, mss.Vpcs, mss.DnsRecords)
	ovndb.Sweep(ctx)
	return nil
}
