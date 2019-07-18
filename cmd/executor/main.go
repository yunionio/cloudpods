package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"yunion.io/x/log"

	exec "yunion.io/x/onecloud/pkg/executor/executor"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"
)

func init() {
	var socketPath string
	flag.StringVar(&socketPath, "socket-path", "", "execute service listen socket path")
	flag.Parse()
	if len(socketPath) == 0 {
		panic("socket path not provide")
	}
	exec.Init(socketPath)

	signalutils.RegisterSignal(func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}, syscall.SIGUSR1)
	signalutils.StartTrap()
}

func main() {
	fmt.Print("# ")
	reader := bufio.NewReader(os.Stdin)
	var cmdRunning bool
	var pr *io.PipeReader
	var pw *io.PipeWriter
	for {
		content, _ := reader.ReadString('\n')
		if cmdRunning {
			io.WriteString(pw, content)
			continue
		}
		input := strings.TrimSpace(strings.Trim(content, "\n"))
		if utils.IsInStringArray(input, []string{"exit", "quit"}) {
			return
		} else if input == "" {
			fmt.Print("# ")
			continue
		}
		// inputCmd := utils.ArgsStringToArray(input)
		pr, pw = io.Pipe()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c := exec.CommandContext(ctx, "sh", "-c", input)
		c.Stdin = pr
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Start(); err != nil {
			fmt.Printf("cmd %s exec failed: %s\n", input, err)
		}
		cmdRunning = true
		go func() {
			if err := c.Wait(); err != nil {
				fmt.Printf("cmd %s exec failed: %s\n", input, err)
			}
			cancel()
			pr.Close()
			cmdRunning = false
			fmt.Print("# ")
		}()
	}
}
