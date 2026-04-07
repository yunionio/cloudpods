// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package version

import (
	"regexp"
	"strconv"
)

// Tag specifies the current release tag. It needs to be manually
// updated. A test checks that the value of Tag never points to a
// git tag that is older than HEAD.
const Tag = "v1.47.0"

// Dissected version number. Filled during init()
var (
	// Major is the current major version number
	Major int
	// Minor is the current minor version number
	Minor int
	// Patch is the current patch version number
	Patch int
	// RC is the current release candidate version number
	RC int
)

func init() {
	// This regexp matches the version format we use and captures major/minor/patch/rc in different groups
	r := regexp.MustCompile(`v(?P<ma>\d+)\.(?P<mi>\d+)\.(?P<pa>\d+)(-rc\.(?P<rc>\d+))?`)
	names := r.SubexpNames()
	captures := map[string]string{}
	// Associate each capture group match with the capture group's name to easily retrieve major/minor/patch/rc
	for k, v := range r.FindAllStringSubmatch(Tag, -1)[0] {
		captures[names[k]] = v
	}
	Major, _ = strconv.Atoi(captures["ma"])
	Minor, _ = strconv.Atoi(captures["mi"])
	Patch, _ = strconv.Atoi(captures["pa"])
	RC, _ = strconv.Atoi(captures["rc"])
}
