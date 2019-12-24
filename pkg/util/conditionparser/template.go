package conditionparser

import (
	"regexp"
	"strings"

	"yunion.io/x/pkg/errors"
)

var (
	tempPattern = regexp.MustCompile(`\$\{[\w\d._-]*\}`)
)

func IsTemplate(template string) bool {
	return tempPattern.MatchString(template)
}

func EvalTemplate(template string, input interface{}) (string, error) {
	var output strings.Builder
	matches := tempPattern.FindAllStringSubmatchIndex(template, -1)
	offset := 0
	for _, match := range matches {
		output.WriteString(template[offset:match[0]])
		o, err := EvalString(template[match[0]+2:match[1]-1], input)
		if err != nil {
			return "", errors.Wrap(err, template[match[0]+2:match[1]])
		}
		output.WriteString(o)
		offset = match[1]
	}
	if offset < len(template) {
		output.WriteString(template[offset:])
	}
	return output.String(), nil
}
