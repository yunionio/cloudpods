package cgrouputils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestCgroupSet(t *testing.T) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter pid: ")
	pid, _ := reader.ReadString('\n')
	pid = strings.TrimSpace(pid)
	t.Logf("Start %s cgroup set", pid)
	CgroupSet(pid, 1)
	CgroupCleanAll()
}
