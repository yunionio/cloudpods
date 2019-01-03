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
