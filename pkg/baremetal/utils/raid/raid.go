package raid

import (
	"strings"
)

func GetCommand(bin string, args ...string) string {
	cmd := []string{bin}
	cmd = append(cmd, args...)
	return strings.Join(cmd, " ")
}
