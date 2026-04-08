// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:build !appsec
// +build !appsec

package appsec

import "gopkg.in/DataDog/dd-trace-go.v1/internal/log"

// Enabled returns true when AppSec is up and running. Meaning that the appsec build tag is enabled, the env var
// DD_APPSEC_ENABLED is set to true, and the tracer is started.
func Enabled() bool {
	return false
}

// Start AppSec when enabled by both using the appsec build tag and
// setting the environment variable DD_APPSEC_ENABLED to true.
func Start(...StartOption) {
	if enabled, _, err := isEnabled(); err != nil {
		// Something went wrong while checking the DD_APPSEC_ENABLED configuration
		log.Error("appsec: error while checking if appsec is enabled: %v", err)
	} else if enabled {
		// The user is willing to enable appsec but didn't use the build tag
		log.Info("appsec: enabled by the configuration but has not been activated during the compilation: please add the go build tag `appsec` to your build options to enable it")
	} else {
		// The user is not willing to start appsec, a simple debug log is enough
		log.Debug("appsec: not been not enabled during the compilation: please add the go build tag `appsec` to your build options to enable it")
	}
}

// Stop AppSec.
func Stop() {}

// Static rule stubs when disabled.
const staticRecommendedRules = ""
