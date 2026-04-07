// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:build appsec
// +build appsec

package appsec

import (
	"sync"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/dyngo"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/remoteconfig"
)

// Enabled returns true when AppSec is up and running. Meaning that the appsec build tag is enabled, the env var
// DD_APPSEC_ENABLED is set to true, and the tracer is started.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return activeAppSec != nil && activeAppSec.started
}

// Start AppSec when enabled is enabled by both using the appsec build tag and
// setting the environment variable DD_APPSEC_ENABLED to true.
func Start(opts ...StartOption) {
	enabled, set, err := isEnabled()
	if err != nil {
		logUnexpectedStartError(err)
		return
	}
	// Check if AppSec is explicitly disabled
	if set && !enabled {
		log.Debug("appsec: disabled by the configuration: set the environment variable DD_APPSEC_ENABLED to true to enable it")
		return
	}
	// From this point we know that AppSec is either enabled or can be enabled through remote config
	cfg, err := newConfig()
	if err != nil {
		logUnexpectedStartError(err)
		return
	}
	for _, opt := range opts {
		opt(cfg)
	}
	appsec := newAppSec(cfg)
	appsec.startRC()

	// If the env var is not set ASM is disabled, but can be enabled through remote config
	if !set {
		log.Debug("appsec: %s is not set. AppSec won't start until activated through remote configuration", enabledEnvVar)
		if err := appsec.enableRemoteActivation(); err != nil {
			// ASM is not enabled and can't be enabled through remote configuration. Nothing more can be done.
			logUnexpectedStartError(err)
			appsec.stopRC()
			return
		}
	} else if err := appsec.start(); err != nil { // AppSec is specifically enabled
		logUnexpectedStartError(err)
		appsec.stopRC()
		return
	}
	setActiveAppSec(appsec)
}

// Implement the AppSec log message C1
func logUnexpectedStartError(err error) {
	log.Error("appsec: could not start because of an unexpected error: %v\nNo security activities will be collected. Please contact support at https://docs.datadoghq.com/help/ for help.", err)
}

// Stop AppSec.
func Stop() {
	setActiveAppSec(nil)
}

var (
	activeAppSec *appsec
	mu           sync.RWMutex
)

func setActiveAppSec(a *appsec) {
	mu.Lock()
	defer mu.Unlock()
	if activeAppSec != nil {
		activeAppSec.stopRC()
		activeAppSec.stop()
	}
	activeAppSec = a
}

type appsec struct {
	cfg           *Config
	unregisterWAF dyngo.UnregisterFunc
	limiter       *TokenTicker
	rc            *remoteconfig.Client
	started       bool
}

func newAppSec(cfg *Config) *appsec {
	var client *remoteconfig.Client
	var err error
	if cfg.rc != nil {
		client, err = remoteconfig.NewClient(*cfg.rc)
	}
	if err != nil {
		log.Error("appsec: Remote config: disabled due to a client creation error: %v", err)
	}
	return &appsec{
		cfg: cfg,
		rc:  client,
	}
}

// Start AppSec by registering its security protections according to the configured the security rules.
func (a *appsec) start() error {
	a.limiter = NewTokenTicker(int64(a.cfg.traceRateLimit), int64(a.cfg.traceRateLimit))
	a.limiter.Start()
	// Register the WAF operation event listener
	unregisterWAF, err := a.registerWAF()
	if err != nil {
		return err
	}
	a.unregisterWAF = unregisterWAF
	a.started = true
	return nil
}

// Stop AppSec by unregistering the security protections.
func (a *appsec) stop() {
	if a.started {
		a.started = false
		a.unregisterWAF()
		a.limiter.Stop()
	}
}
