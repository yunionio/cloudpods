// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	mcmodels "yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	sshutil "yunion.io/x/onecloud/pkg/util/ssh"
)

type SEditionSwitch struct {
	db.SStatusStandaloneResourceBase
	Edition                   string `width:"36" charset:"utf8" nullable:"false" list:"user" get:"user"`
	Host                      string `width:"64" charset:"ascii" nullable:"true" list:"user" get:"user"`
	Reason                    string `length:"medium" charset:"utf8" nullable:"true" list:"user" get:"user"`
	AnsiblePlaybookInstanceId string `width:"36" charset:"ascii" nullable:"true" get:"user"`
}

type SEditionSwitchManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var EditionSwitchManager *SEditionSwitchManager

func init() {
	EditionSwitchManager = &SEditionSwitchManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SEditionSwitch{},
			"editionswitch_tbl",
			"editionswitch",
			"editionswitches",
		),
	}
	EditionSwitchManager.SetVirtualObject(EditionSwitchManager)
}

func (m *SEditionSwitchManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (m *SEditionSwitchManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func requireSystemAdmin(userCred mcclient.TokenCredential) error {
	cred := policy.SPolicyTokenCredential{TokenCredential: userCred}
	if !cred.HasSystemAdminPrivilege() {
		return httperrors.NewForbiddenError("only system admin can switch edition")
	}
	return nil
}

func (m *SEditionSwitchManager) GetPropertySshable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.EditionSwitchSshableOutput, error) {
	if err := requireSystemAdmin(userCred); err != nil {
		return api.EditionSwitchSshableOutput{}, err
	}
	return ProbeControlPlaneSSH(ctx)
}

func (m *SEditionSwitchManager) PerformSwitch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.EditionSwitchInput) (api.EditionSwitchOutput, error) {
	output := api.EditionSwitchOutput{}
	if err := requireSystemAdmin(userCred); err != nil {
		return output, err
	}
	if input.Edition != api.EditionCE && input.Edition != api.EditionCESupport {
		return output, httperrors.NewInputParameterError("edition must be %s or %s", api.EditionCE, api.EditionCESupport)
	}
	probe, err := ProbeControlPlaneSSH(ctx)
	if err != nil {
		return output, err
	}
	if !probe.Sshable {
		return output, httperrors.NewNotAcceptableError("control plane is not sshable: %s", probe.Reason)
	}
	running, err := m.Query().Equals("status", api.EditionSwitchStatusRunning).CountWithError()
	if err != nil {
		return output, errors.Wrap(err, "count running edition switches")
	}
	if running > 0 {
		return output, httperrors.NewConflictError("an edition switch is already running")
	}

	sw := &SEditionSwitch{
		Edition: input.Edition,
		Host:    probe.Host,
	}
	sw.Name = fmt.Sprintf("switch-%s-%d", strings.ReplaceAll(input.Edition, "/", "-"), time.Now().Unix())
	sw.Status = api.EditionSwitchStatusRunning
	if err := m.TableSpec().Insert(ctx, sw); err != nil {
		return output, errors.Wrap(err, "insert edition switch")
	}
	sw.SetModelManager(m, sw)

	task, err := taskman.TaskManager.NewTask(ctx, "SwitchEditionTask", sw, userCred, nil, "", "")
	if err != nil {
		_ = sw.MarkFailed(err.Error())
		return output, errors.Wrap(err, "start SwitchEditionTask")
	}
	task.ScheduleRun(nil)
	output.Id = sw.Id
	return output, nil
}

func (sw *SEditionSwitch) MarkFailed(reason string) error {
	_, err := db.Update(sw, func() error {
		sw.Status = api.EditionSwitchStatusFailed
		sw.Reason = reason
		return nil
	})
	return err
}

func (sw *SEditionSwitch) MarkSucceeded() error {
	_, err := db.Update(sw, func() error {
		sw.Status = api.EditionSwitchStatusSucceeded
		sw.Reason = ""
		return nil
	})
	return err
}

func (sw *SEditionSwitch) GetDetailsAnsibleLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.EditionSwitchAnsibleLogOutput, error) {
	output := api.EditionSwitchAnsibleLogOutput{
		AnsiblePlaybookInstanceId: sw.AnsiblePlaybookInstanceId,
		Status:                    sw.Status,
	}
	if err := requireSystemAdmin(userCred); err != nil {
		return output, err
	}
	logText, err := ReadOcbootSwitchLog(ctx, sw.Host)
	if err != nil {
		return output, err
	}
	// The switch script creates this file only after detect-os. Until then, and
	// when ocboot exits before that task, the runner stdout is the only record
	// of the python3 ocboot.py invocation.
	if len(strings.TrimSpace(logText)) == 0 {
		if runner, runnerErr := readOcbootRunnerOutput(ctx, sw.AnsiblePlaybookInstanceId); runnerErr == nil && len(strings.TrimSpace(runner)) > 0 {
			logText = runner
		}
	}
	output.Output = logText
	return output, nil
}

func readOcbootRunnerOutput(ctx context.Context, instanceId string) (string, error) {
	if len(instanceId) == 0 {
		return "", nil
	}
	session := auth.GetAdminSessionWithInternal(ctx, "")
	instance, err := ansible.AnsiblePlaybookInstance.Get(session, instanceId, nil)
	if err != nil {
		return "", errors.Wrapf(err, "fetch ansible playbook instance %s", instanceId)
	}
	output, _ := instance.GetString("output")
	return output, nil
}

// RemoveOcbootSwitchLog deletes the previous switch log so a new run does not
// show the last attempt before ocboot recreates the file.
func RemoveOcbootSwitchLog(ctx context.Context, host string) error {
	_, err := runOnControlPlane(ctx, host, "rm -f "+api.OcbootSwitchLogPath)
	return err
}

// ReadOcbootSwitchLog returns the switch script log on the control node.
// The file is created only when ocboot reaches the switch script, so a missing
// file is an empty log rather than an error.
func ReadOcbootSwitchLog(ctx context.Context, host string) (string, error) {
	cmd := fmt.Sprintf("if [ -f %s ]; then cat %s; fi", api.OcbootSwitchLogPath, api.OcbootSwitchLogPath)
	return runOnControlPlane(ctx, host, cmd)
}

func runOnControlPlane(ctx context.Context, host, cmd string) (string, error) {
	if len(host) == 0 {
		return "", errors.Error("edition switch has no control plane host")
	}
	session := auth.GetAdminSession(ctx, "")
	keypairs, err := compute.Sshkeypairs.FetchProjectKeypairsBySession(ctx, session)
	if err != nil {
		return "", errors.Wrap(err, "fetch project ssh key")
	}
	var reasons []string
	for _, kp := range keypairs {
		if len(strings.TrimSpace(kp.PrivateKey)) == 0 {
			continue
		}
		client, dialErr := (sshutil.ClientConfig{
			Username:   api.OcbootSSHUser,
			Host:       host,
			Port:       api.OcbootSSHPort,
			PrivateKey: kp.PrivateKey,
		}).NewClient()
		if dialErr != nil {
			reasons = append(reasons, dialErr.Error())
			continue
		}
		out, runErr := client.RawRun(cmd)
		client.Close()
		if runErr != nil {
			reasons = append(reasons, runErr.Error())
			continue
		}
		if len(out) == 0 {
			return "", nil
		}
		return out[0], nil
	}
	if len(reasons) == 0 {
		return "", errors.Error("no project ssh private key")
	}
	return "", errors.Error(strings.Join(reasons, "; "))
}

func ProbeControlPlaneSSH(ctx context.Context) (api.EditionSwitchSshableOutput, error) {
	output := api.EditionSwitchSshableOutput{
		User: api.OcbootSSHUser,
		Port: api.OcbootSSHPort,
	}
	session := auth.GetAdminSession(ctx, "")
	hostInfo, err := compute.Hosts.Get(session, "k8s-master-node-ips", nil)
	if err != nil {
		output.Reason = fmt.Sprintf("list control plane nodes: %v", err)
		output.ShellCmd = ""
		return output, nil
	}
	output.Host, _ = hostInfo.GetString("primary_ip")
	if len(output.Host) == 0 {
		output.Reason = "no control plane node found"
		return output, nil
	}

	keypairs, err := compute.Sshkeypairs.FetchProjectKeypairsBySession(ctx, session)
	if err != nil {
		output.Reason = fmt.Sprintf("fetch project ssh key: %v", err)
		return output, nil
	}
	output.ShellCmd = authorizedKeysScript(keypairs)

	var reasons []string
	for _, kp := range keypairs {
		if len(strings.TrimSpace(kp.PrivateKey)) == 0 {
			continue
		}
		dialCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		client, dialErr := (sshutil.ClientConfig{
			Username:   api.OcbootSSHUser,
			Host:       output.Host,
			Port:       api.OcbootSSHPort,
			PrivateKey: kp.PrivateKey,
		}).ConnectContext(dialCtx)
		cancel()
		if dialErr != nil {
			reasons = append(reasons, dialErr.Error())
			continue
		}
		client.Close()
		output.Sshable = true
		output.Reason = ""
		return output, nil
	}
	if len(reasons) == 0 {
		output.Reason = "no project ssh private key"
	} else {
		output.Reason = strings.Join(reasons, "; ")
	}
	return output, nil
}

func authorizedKeysScript(keypairs []mcmodels.SshKeypair) string {
	var quoted []string
	for _, kp := range keypairs {
		pub := strings.TrimSpace(kp.PublicKey)
		if len(pub) == 0 {
			continue
		}
		quoted = append(quoted, shellSingleQuote(pub))
	}
	if len(quoted) == 0 {
		return ""
	}
	return fmt.Sprintf(`set -e
install -d -m 700 /root/.ssh
touch /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
for k in %s; do
  grep -qF "$k" /root/.ssh/authorized_keys || echo "$k" >> /root/.ssh/authorized_keys
done
`, strings.Join(quoted, " "))
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (sw *SEditionSwitch) SetAnsibleInstance(id string) {
	_, err := db.Update(sw, func() error {
		sw.AnsiblePlaybookInstanceId = id
		return nil
	})
	if err != nil {
		log.Errorf("record ansible playbook instance %s on edition switch %s: %v", id, sw.Id, err)
	}
}
