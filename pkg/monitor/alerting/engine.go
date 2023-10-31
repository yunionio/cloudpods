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

package alerting

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/benbjohnson/clock"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/registry"
)

// AlertEngine is the background process that
// schedules alert evaluations and makes sure notifications
// are sent.
type AlertEngine struct {
	execQueue     chan *Job
	ticker        *Ticker
	Scheduler     scheduler
	evalHandler   evalHandler
	ruleReader    ruleReader
	resultHandler resultHandler
}

func init() {
	registry.RegisterService(&AlertEngine{})
}

// IsDisabled returns true if the alerting service is disabled for this instance.
func (e *AlertEngine) IsDisabled() bool {
	// TODO: read from config options
	return false
}

// Init initalizes the AlertingService.
func (e *AlertEngine) Init() error {
	e.ticker = NewTicker(time.Now(), time.Second*0, clock.New())
	e.execQueue = make(chan *Job, 1000)
	e.Scheduler = newScheduler()
	e.evalHandler = NewEvalHandler()
	e.ruleReader = newRuleReader()
	e.resultHandler = newResultHandler()
	return nil
}

// Run starts the alerting service background process.
func (e *AlertEngine) Run(ctx context.Context) error {
	alertGroup, ctx := errgroup.WithContext(ctx)
	alertGroup.Go(func() error { return e.alertingTicker(ctx) })
	alertGroup.Go(func() error { return e.runJobDispatcher(ctx) })

	err := alertGroup.Wait()
	return err
}

func (e *AlertEngine) alertingTicker(ctx context.Context) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Scheduler panic: stopping alertingTicker, error: %v", err)
			debug.PrintStack()
		}
	}()

	tickIndex := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case tick := <-e.ticker.C:
			// TEMP SOLUTION update rules ever tenth tick
			if tickIndex%10 == 0 {
				e.Scheduler.Update(e.ruleReader.fetch())
			}

			e.Scheduler.Tick(tick, e.execQueue)
			tickIndex++
		}
	}
}

func (e *AlertEngine) runJobDispatcher(ctx context.Context) error {
	dispatcherGroup, alertCtx := errgroup.WithContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return dispatcherGroup.Wait()
		case job := <-e.execQueue:
			dispatcherGroup.Go(func() error { return e.processJobWithRetry(alertCtx, job) })
		}
	}
}

var (
	unfinishedWorkTimeout = time.Second * 5
)

func (e *AlertEngine) processJobWithRetry(ctx context.Context, job *Job) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Alert panic, error: %v", err)
		}
	}()

	cancelChan := make(chan context.CancelFunc, options.Options.AlertingMaxAttempts*2)
	attemptChan := make(chan int, 1)

	// Initialize with first attemptID=1
	attemptChan <- 1
	job.SetRunning(true)

	for {
		select {
		case <-ctx.Done():
			// In case monitor server is cancel, let a chance to job processing
			// to finish gracefully - by waiting a timeout duration -
			unfinishedWorkTimer := time.NewTimer(unfinishedWorkTimeout)
			select {
			case <-unfinishedWorkTimer.C:
				return e.endJob(ctx.Err(), cancelChan, job)
			case <-attemptChan:
				return e.endJob(nil, cancelChan, job)
			}
		case attemptId, more := <-attemptChan:
			if !more {
				return e.endJob(nil, cancelChan, job)
			}
			go e.processJob(attemptId, attemptChan, cancelChan, job)
		}
	}
}

func (e *AlertEngine) endJob(err error, cancelChan chan context.CancelFunc, job *Job) error {
	job.SetRunning(false)
	close(cancelChan)
	for cancelFn := range cancelChan {
		cancelFn()
	}
	return err
}

func (e *AlertEngine) processJob(attemptID int, attemptChan chan int, cancelChan chan context.CancelFunc, job *Job) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Alert Panic: error: %v", err)
		}
	}()

	alertCtx, cancelFn := context.WithTimeout(context.Background(), time.Duration(options.Options.AlertingEvaluationTimeoutSeconds)*time.Second)
	cancelChan <- cancelFn
	// span := opentracing.StartSpan("alert execution")
	// alertCtx = opentracing.ContextWithSpan(alertCtx, span)

	evalContext := NewEvalContext(alertCtx, auth.AdminCredential(), job.Rule)
	evalContext.Ctx = alertCtx
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("Alert panic, error: %v", err)
				debug.PrintStack()
				// ext.Error.Set(span, true)
				// span.LogFields(
				//	tlog.Error(fmt.Errorf("%v", err)),
				//	tlog.String("message", "failed to execute alert rule. panic was recovered."),
				//)
				//span.Finish()
				close(attemptChan)
			}
		}()

		e.evalHandler.Eval(evalContext)

		/*span.SetTag("alertId", evalContext.Rule.ID)
		span.SetTag("dashboardId", evalContext.Rule.DashboardID)
		span.SetTag("firing", evalContext.Firing)
		span.SetTag("nodatapoints", evalContext.NoDataFound)
		span.SetTag("attemptID", attemptID)*/

		if evalContext.Error != nil {
			/*ext.Error.Set(span, true)
			span.LogFields(
				tlog.Error(evalContext.Error),
				tlog.String("message", "alerting execution attempt failed"),
			)
			*/
			if attemptID < options.Options.AlertingMaxAttempts {
				// span.Finish(
				log.Warningf("Job Execution attempt triggered retry, timeMs: %v, alertId: %d",
					evalContext.GetDurationMs(), attemptID)
				attemptChan <- (attemptID + 1)
				return
			}
			log.Errorf("gt AlertingMaxAttempts, error: %v", evalContext.Error)
			close(attemptChan)
			return
		}

		// create new context with timeout for notifications
		resultHandleCtx, resultHandleCancelFn := context.WithTimeout(context.Background(), time.Duration(options.Options.AlertingNotificationTimeoutSeconds)*time.Second)
		cancelChan <- resultHandleCancelFn

		// override the context used for evaluation with a new context for notifications.
		// This makes it possible for notifiers to execute when datasources
		// don't respond within the timeout limit. We should rewrite this so notifications
		// don't reuse the evalContext and get its own context.
		evalContext.Ctx = resultHandleCtx
		evalContext.Rule.State = evalContext.GetNewState()
		if err := e.resultHandler.handle(evalContext); err != nil {
			if xerrors.Is(err, context.Canceled) {
				log.Warningf("Result handler returned context.Canceled")
			} else if xerrors.Is(err, context.DeadlineExceeded) {
				log.Warningf("Result handler returned context.DeadlineExceeded")
			} else {
				log.Errorf("Failed to handle result: %v", err)
			}
		}

		// span.Finish()
		log.Debugf("Job execution completed, timeMs: %v, alertId: %s, attemptId: %d", evalContext.GetDurationMs(), evalContext.Rule.Id, attemptID)
		close(attemptChan)
	}()
}
