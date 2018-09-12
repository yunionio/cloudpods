package command

import (
	"fmt"
	"os"
	"os/exec"

	"yunion.io/x/log"

	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

type Kubectl struct {
	*BaseCommand
	kubeconfig string
}

func NewKubectlCommand(kubeconfig, namespace string) *Kubectl {
	name := o.Options.KubectlPath
	if len(namespace) == 0 {
		namespace = "default"
	}
	cmd := NewBaseCommand(name, "--namespace", namespace)
	return &Kubectl{
		BaseCommand: cmd,
		kubeconfig:  kubeconfig,
	}
}

func (c *Kubectl) GetCommand() *exec.Cmd {
	cmd := c.BaseCommand.GetCommand()
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", c.kubeconfig))
	return cmd
}

func (c Kubectl) GetProtocol() string {
	return PROTOCOL_TTY
}

func (c *Kubectl) Cleanup() error {
	log.Debugf("Remove temp kubeconfig file: %s", c.kubeconfig)
	return os.Remove(c.kubeconfig)
}

type KubectlExec struct {
	*Kubectl
}

func (c *Kubectl) Exec() *KubectlExec {
	// Execute a command in a container
	cmd := &KubectlExec{
		Kubectl: c,
	}
	cmd.AppendArgs("exec")
	return cmd
}

func (c *KubectlExec) Stdin() *KubectlExec {
	// -i: Pass stdin to the container
	c.AppendArgs("-i")
	return c
}

func (c *KubectlExec) TTY() *KubectlExec {
	// -t: Stdin is a TTY
	c.AppendArgs("-t")
	return c
}

func (c *KubectlExec) Container(name string) *KubectlExec {
	if len(name) == 0 {
		return c
	}
	// -c: Container name. If ommitted, the first container in the pod will be chosen
	c.AppendArgs("-c", name)
	return c
}

func (c *KubectlExec) Pod(name string) *KubectlExec {
	// Pod name
	c.AppendArgs(name)
	return c
}

func (c *KubectlExec) Command(cmd string, args ...string) *KubectlExec {
	c.AppendArgs("--", cmd)
	c.AppendArgs(args...)
	return c
}

func NewPodBashCommand(kubeconfig, namespace, pod, container string) ICommand {
	return NewKubectlCommand(kubeconfig, namespace).Exec().
		Stdin().
		TTY().
		Pod(pod).
		Container(container).
		Command("sh")
}

type KubectlLog struct {
	*Kubectl
}

func (c *Kubectl) Logs() *KubectlLog {
	// Print the logs for a container in a pod
	cmd := &KubectlLog{
		Kubectl: c,
	}
	cmd.AppendArgs("logs")
	return cmd
}

func (c *KubectlLog) Follow() *KubectlLog {
	// -f: Specify if the logs should be streamed
	c.AppendArgs("-f")
	return c
}

func (c *KubectlLog) Pod(name string) *KubectlLog {
	// Pod name
	c.AppendArgs(name)
	return c
}

func (c *KubectlLog) Container(name string) *KubectlLog {
	if name == "" {
		return c
	}
	// -c, --container='': Print the logs of this container
	c.AppendArgs("-c", name)
	return c
}

func NewPodLogCommand(kubeconfig, namespace, pod, container string) ICommand {
	return NewKubectlCommand(kubeconfig, namespace).Logs().
		Follow().
		Pod(pod).
		Container(container)
}
