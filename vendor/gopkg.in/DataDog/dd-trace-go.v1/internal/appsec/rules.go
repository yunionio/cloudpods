// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:build appsec
// +build appsec

package appsec

import _ "embed"

// Static recommended AppSec rule 1.4.3
// Source: https://github.com/DataDog/appsec-event-rules/blob/1.4.3/build/recommended.json
//
//go:embed rules.json
var staticRecommendedRules string
