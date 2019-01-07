package regutils2

import "regexp"

/**
 * Parses val with the given regular expression and returns the
 * group values defined in the expression.
 */
func GetParams(compRegEx *regexp.Regexp, val string) map[string]string {
	match := compRegEx.FindStringSubmatch(val)

	paramsMap := make(map[string]string, 0)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return paramsMap
}

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
