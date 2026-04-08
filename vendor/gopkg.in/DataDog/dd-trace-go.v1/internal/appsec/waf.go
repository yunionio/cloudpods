// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:build appsec
// +build appsec

package appsec

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/dyngo"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/dyngo/instrumentation/grpcsec"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/dyngo/instrumentation/httpsec"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/waf"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/samplernames"
)

const (
	eventRulesVersionTag = "_dd.appsec.event_rules.version"
	eventRulesErrorsTag  = "_dd.appsec.event_rules.errors"
	eventRulesLoadedTag  = "_dd.appsec.event_rules.loaded"
	eventRulesFailedTag  = "_dd.appsec.event_rules.error_count"
	wafDurationTag       = "_dd.appsec.waf.duration"
	wafDurationExtTag    = "_dd.appsec.waf.duration_ext"
	wafTimeoutTag        = "_dd.appsec.waf.timeouts"
	wafVersionTag        = "_dd.appsec.waf.version"
)

// Register the WAF event listener.
func (a *appsec) registerWAF() (unreg dyngo.UnregisterFunc, err error) {
	// Check the WAF is healthy
	if err := waf.Health(); err != nil {
		return nil, err
	}

	// Instantiate the WAF
	waf, err := waf.NewHandle(a.cfg.rules, a.cfg.obfuscator.KeyRegex, a.cfg.obfuscator.ValueRegex)
	if err != nil {
		return nil, err
	}
	// Close the WAF in case of an error in what's following
	defer func() {
		if err != nil {
			waf.Close()
		}
	}()

	// Check if there are addresses in the rule
	ruleAddresses := waf.Addresses()
	if len(ruleAddresses) == 0 {
		return nil, errors.New("no addresses found in the rule")
	}
	// Check there are supported addresses in the rule
	httpAddresses, grpcAddresses, notSupported := supportedAddresses(ruleAddresses)
	if len(httpAddresses) == 0 && len(grpcAddresses) == 0 {
		return nil, fmt.Errorf("the addresses present in the rule are not supported: %v", notSupported)
	} else if len(notSupported) > 0 {
		log.Debug("appsec: the addresses present in the rule are partially supported: not supported=%v", notSupported)
	}

	// Register the WAF event listener
	var unregisterHTTP, unregisterGRPC dyngo.UnregisterFunc
	if len(httpAddresses) > 0 {
		log.Debug("appsec: registering http waf listening to addresses %v", httpAddresses)
		unregisterHTTP = dyngo.Register(newHTTPWAFEventListener(waf, httpAddresses, a.cfg.wafTimeout, a.limiter))
	}
	if len(grpcAddresses) > 0 {
		log.Debug("appsec: registering grpc waf listening to addresses %v", grpcAddresses)
		unregisterGRPC = dyngo.Register(newGRPCWAFEventListener(waf, grpcAddresses, a.cfg.wafTimeout, a.limiter))
	}

	if err := a.enableRCBlocking(wafHandleWrapper{waf}); err != nil {
		log.Error("appsec: Remote config: cannot enable blocking, rules data won't be updated: %v", err)
	}

	// Return an unregistration function that will also release the WAF instance.
	return func() {
		defer waf.Close()
		if unregisterHTTP != nil {
			unregisterHTTP()
		}
		if unregisterGRPC != nil {
			unregisterGRPC()
		}
	}, nil
}

// newWAFEventListener returns the WAF event listener to register in order to enable it.
func newHTTPWAFEventListener(handle *waf.Handle, addresses []string, timeout time.Duration, limiter Limiter) dyngo.EventListener {
	var monitorRulesOnce sync.Once // per instantiation
	actionHandler := httpsec.NewActionsHandler()

	return httpsec.OnHandlerOperationStart(func(op *httpsec.Operation, args httpsec.HandlerOperationArgs) {
		var body interface{}
		wafCtx := waf.NewContext(handle)
		if wafCtx == nil {
			// The WAF event listener got concurrently released
			return
		}

		values := map[string]interface{}{}
		for _, addr := range addresses {
			if addr == httpClientIPAddr && args.ClientIP.IsValid() {
				values[httpClientIPAddr] = args.ClientIP.String()
			}
		}
		// TODO: suspicious request blocking by moving here all the addresses available when the request begins

		matches, actionIds := runWAF(wafCtx, values, timeout)
		if len(matches) > 0 {
			interrupt := false
			for _, id := range actionIds {
				interrupt = actionHandler.Apply(id, op) || interrupt
			}
			op.AddSecurityEvents(matches)
			log.Debug("appsec: WAF detected an attack before executing the request")
			if interrupt {
				wafCtx.Close()
				return
			}
		}

		op.On(httpsec.OnSDKBodyOperationStart(func(op *httpsec.SDKBodyOperation, args httpsec.SDKBodyOperationArgs) {
			body = args.Body
		}))

		// At the moment, AppSec doesn't block the requests, and so we can use the fact we are in monitoring-only mode
		// to call the WAF only once at the end of the handler operation.
		op.On(httpsec.OnHandlerOperationFinish(func(op *httpsec.Operation, res httpsec.HandlerOperationRes) {
			defer wafCtx.Close()
			// Run the WAF on the rule addresses available in the request args
			values := make(map[string]interface{}, len(addresses))
			for _, addr := range addresses {
				switch addr {
				case serverRequestRawURIAddr:
					values[serverRequestRawURIAddr] = args.RequestURI
				case serverRequestHeadersNoCookiesAddr:
					if headers := args.Headers; headers != nil {
						values[serverRequestHeadersNoCookiesAddr] = headers
					}
				case serverRequestCookiesAddr:
					if cookies := args.Cookies; cookies != nil {
						values[serverRequestCookiesAddr] = cookies
					}
				case serverRequestQueryAddr:
					if query := args.Query; query != nil {
						values[serverRequestQueryAddr] = query
					}
				case serverRequestPathParamsAddr:
					if pathParams := args.PathParams; pathParams != nil {
						values[serverRequestPathParamsAddr] = pathParams
					}
				case serverRequestBodyAddr:
					if body != nil {
						values[serverRequestBodyAddr] = body
					}
				case serverResponseStatusAddr:
					values[serverResponseStatusAddr] = res.Status
				}
			}
			// Run the WAF, ignoring the returned actions - if any - since blocking after the request handler's
			// response is not supported at the moment.
			matches, _ := runWAF(wafCtx, values, timeout)

			// Add WAF metrics.
			rInfo := handle.RulesetInfo()
			overallRuntimeNs, internalRuntimeNs := wafCtx.TotalRuntime()
			addWAFMonitoringTags(op, rInfo.Version, overallRuntimeNs, internalRuntimeNs, wafCtx.TotalTimeouts())

			// Add the following metrics once per instantiation of a WAF handle
			monitorRulesOnce.Do(func() {
				addRulesMonitoringTags(op, rInfo)
				op.AddTag(ext.ManualKeep, samplernames.AppSec)
			})

			// Log the attacks if any
			if len(matches) == 0 {
				return
			}
			log.Debug("appsec: attack detected by the waf")
			if limiter.Allow() {
				op.AddSecurityEvents(matches)
			}
		}))
	})
}

// newGRPCWAFEventListener returns the WAF event listener to register in order
// to enable it.
func newGRPCWAFEventListener(handle *waf.Handle, addresses []string, timeout time.Duration, limiter Limiter) dyngo.EventListener {
	var monitorRulesOnce sync.Once // per instantiation
	actionHandler := grpcsec.NewActionsHandler()

	return grpcsec.OnHandlerOperationStart(func(op *grpcsec.HandlerOperation, handlerArgs grpcsec.HandlerOperationArgs) {
		// Limit the maximum number of security events, as a streaming RPC could
		// receive unlimited number of messages where we could find security events
		const maxWAFEventsPerRequest = 10
		var (
			nbEvents          uint32
			logOnce           sync.Once // per request
			overallRuntimeNs  waf.AtomicU64
			internalRuntimeNs waf.AtomicU64
			nbTimeouts        waf.AtomicU64

			events []json.RawMessage
			mu     sync.Mutex // events mutex
		)

		wafCtx := waf.NewContext(handle)
		if wafCtx == nil {
			// The WAF event listener got concurrently released
			return
		}
		defer wafCtx.Close()

		// The same address is used for gRPC and http when it comes to client ip
		values := map[string]interface{}{}
		for _, addr := range addresses {
			if addr == httpClientIPAddr && handlerArgs.ClientIP.IsValid() {
				values[httpClientIPAddr] = handlerArgs.ClientIP.String()
			}
		}

		matches, actionIds := runWAF(wafCtx, values, timeout)
		if len(matches) > 0 {
			interrupt := false
			for _, id := range actionIds {
				interrupt = actionHandler.Apply(id, op) || interrupt
			}
			op.AddSecurityEvents(matches)
			log.Debug("appsec: WAF detected an attack before executing the request")
			if interrupt {
				return
			}
		}

		op.On(grpcsec.OnReceiveOperationFinish(func(_ grpcsec.ReceiveOperation, res grpcsec.ReceiveOperationRes) {
			if atomic.LoadUint32(&nbEvents) == maxWAFEventsPerRequest {
				logOnce.Do(func() {
					log.Debug("appsec: ignoring the rpc message due to the maximum number of security events per grpc call reached")
				})
				return
			}
			// The current workaround of the WAF context limitations is to
			// simply instantiate and release the WAF context for the operation
			// lifetime so that:
			//   1. We avoid growing the memory usage of the context every time
			//      a grpc.server.request.message value is added to it during
			//      the RPC lifetime.
			//   2. We avoid the limitation of 1 event per attack type.
			// TODO(Julio-Guerra): a future libddwaf API should solve this out.
			wafCtx := waf.NewContext(handle)
			if wafCtx == nil {
				// The WAF event listener got concurrently released
				return
			}
			defer wafCtx.Close()
			// Run the WAF on the rule addresses available in the args
			// Note that we don't check if the address is present in the rules
			// as we only support one at the moment, so this callback cannot be
			// set when the address is not present.
			values := map[string]interface{}{grpcServerRequestMessage: res.Message}
			if md := handlerArgs.Metadata; len(md) > 0 {
				values[grpcServerRequestMetadata] = md
			}
			// Run the WAF, ignoring the returned actions - if any - since blocking after the request handler's
			// response is not supported at the moment.
			event, _ := runWAF(wafCtx, values, timeout)

			// WAF run durations are WAF context bound. As of now we need to keep track of those externally since
			// we use a new WAF context for each callback. When we are able to re-use the same WAF context across
			// callbacks, we can get rid of these variables and simply use the WAF bindings in OnHandlerOperationFinish.
			overall, internal := wafCtx.TotalRuntime()
			overallRuntimeNs.Add(overall)
			internalRuntimeNs.Add(internal)
			nbTimeouts.Add(wafCtx.TotalTimeouts())

			if len(event) == 0 {
				return
			}
			log.Debug("appsec: attack detected by the grpc waf")
			atomic.AddUint32(&nbEvents, 1)
			mu.Lock()
			events = append(events, event)
			mu.Unlock()
		}))

		op.On(grpcsec.OnHandlerOperationFinish(func(op *grpcsec.HandlerOperation, _ grpcsec.HandlerOperationRes) {
			rInfo := handle.RulesetInfo()
			addWAFMonitoringTags(op, rInfo.Version, overallRuntimeNs.Load(), internalRuntimeNs.Load(), nbTimeouts.Load())

			// Log the following metrics once per instantiation of a WAF handle
			monitorRulesOnce.Do(func() {
				addRulesMonitoringTags(op, rInfo)
				op.AddTag(ext.ManualKeep, samplernames.AppSec)
			})

			// Log the events if any
			if len(events) > 0 && limiter.Allow() {
				op.AddSecurityEvents(events...)
			}
		}))
	})
}

func runWAF(wafCtx *waf.Context, values map[string]interface{}, timeout time.Duration) ([]byte, []string) {
	matches, actions, err := wafCtx.Run(values, timeout)
	if err != nil {
		if err == waf.ErrTimeout {
			log.Debug("appsec: waf timeout value of %s reached", timeout)
		} else {
			log.Error("appsec: unexpected waf error: %v", err)
			return nil, nil
		}
	}
	return matches, actions
}

// HTTP rule addresses currently supported by the WAF
const (
	serverRequestRawURIAddr           = "server.request.uri.raw"
	serverRequestHeadersNoCookiesAddr = "server.request.headers.no_cookies"
	serverRequestCookiesAddr          = "server.request.cookies"
	serverRequestQueryAddr            = "server.request.query"
	serverRequestPathParamsAddr       = "server.request.path_params"
	serverRequestBodyAddr             = "server.request.body"
	serverResponseStatusAddr          = "server.response.status"
	httpClientIPAddr                  = "http.client_ip"
)

// List of HTTP rule addresses currently supported by the WAF
var httpAddresses = []string{
	serverRequestRawURIAddr,
	serverRequestHeadersNoCookiesAddr,
	serverRequestCookiesAddr,
	serverRequestQueryAddr,
	serverRequestPathParamsAddr,
	serverRequestBodyAddr,
	serverResponseStatusAddr,
	httpClientIPAddr,
}

// gRPC rule addresses currently supported by the WAF
const (
	grpcServerRequestMessage  = "grpc.server.request.message"
	grpcServerRequestMetadata = "grpc.server.request.metadata"
)

// List of gRPC rule addresses currently supported by the WAF
var grpcAddresses = []string{
	grpcServerRequestMessage,
	grpcServerRequestMetadata,
	httpClientIPAddr,
}

func init() {
	// sort the address lists to avoid mistakes and use sort.SearchStrings()
	sort.Strings(httpAddresses)
	sort.Strings(grpcAddresses)
}

// supportedAddresses returns the list of addresses we actually support from the
// given rule addresses.
func supportedAddresses(ruleAddresses []string) (supportedHTTP, supportedGRPC, notSupported []string) {
	// Filter the supported addresses only
	for _, addr := range ruleAddresses {
		supported := false
		if i := sort.SearchStrings(httpAddresses, addr); i < len(httpAddresses) && httpAddresses[i] == addr {
			supportedHTTP = append(supportedHTTP, addr)
			supported = true
		}
		if i := sort.SearchStrings(grpcAddresses, addr); i < len(grpcAddresses) && grpcAddresses[i] == addr {
			supportedGRPC = append(supportedGRPC, addr)
			supported = true
		}
		if !supported {
			notSupported = append(notSupported, addr)
		}
	}
	return
}

type tagsHolder interface {
	AddTag(string, interface{})
}

// Add the tags related to security rules monitoring
func addRulesMonitoringTags(th tagsHolder, rInfo waf.RulesetInfo) {
	if len(rInfo.Errors) == 0 {
		rInfo.Errors = nil
	}
	rulesetErrors, err := json.Marshal(rInfo.Errors)
	if err != nil {
		log.Error("appsec: could not marshal ruleset info errors to json")
	}
	th.AddTag(eventRulesErrorsTag, string(rulesetErrors)) // avoid the tracer's call to fmt.Sprintf on the value
	th.AddTag(eventRulesLoadedTag, float64(rInfo.Loaded))
	th.AddTag(eventRulesFailedTag, float64(rInfo.Failed))
	th.AddTag(wafVersionTag, waf.Version())
}

// Add the tags related to the monitoring of the WAF
func addWAFMonitoringTags(th tagsHolder, rulesVersion string, overallRuntimeNs, internalRuntimeNs, timeouts uint64) {
	// Rules version is set for every request to help the backend associate WAF duration metrics with rule version
	th.AddTag(eventRulesVersionTag, rulesVersion)
	th.AddTag(wafTimeoutTag, float64(timeouts))
	th.AddTag(wafDurationTag, float64(internalRuntimeNs)/1e3)   // ns to us
	th.AddTag(wafDurationExtTag, float64(overallRuntimeNs)/1e3) // ns to us
}
