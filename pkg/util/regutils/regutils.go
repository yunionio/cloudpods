package regutils

import (
	"regexp"
)

func SubGroupMatch(pattern string, line string) map[string]string {
	regEx := regexp.MustCompile(pattern)
	params := make(map[string]string)
	matches := regEx.FindStringSubmatch(line)
	if len(matches) == 0 {
		return params
	}
	for i, name := range regEx.SubexpNames() {
		if i > 0 && i <= len(matches) {
			params[name] = matches[i]
		}
	}
	return params
}
