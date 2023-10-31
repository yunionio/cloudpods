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

package lbagent

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/system_service"
	agentmodels "yunion.io/x/onecloud/pkg/lbagent/models"
	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type HaproxyHelper struct {
	opts *Options

	lbagentId string

	configDirMan *agentutils.ConfigDirManager
}

func NewHaproxyHelper(opts *Options, lbagentId string) (*HaproxyHelper, error) {
	helper := &HaproxyHelper{
		opts:         opts,
		lbagentId:    lbagentId,
		configDirMan: agentutils.NewConfigDirManager(opts.haproxyConfigDir),
	}
	{
		// sysctl
		args := []string{
			"sysctl", "-w",
			"net.ipv4.ip_nonlocal_bind=1",
			"net.ipv4.ip_forward=1",
		}
		if err := helper.runCmd(args); err != nil {
			return nil, fmt.Errorf("sysctl: %s", err)
		}
	}

	system_service.Init()

	return helper, nil
}

func (h *HaproxyHelper) Run(ctx context.Context) {
	defer func() {
		wg := ctx.Value("wg").(*sync.WaitGroup)
		wg.Done()
	}()
	cmdChan := ctx.Value("cmdChan").(chan *LbagentCmd)
	for {
		for {
			select {
			case <-ctx.Done():
				log.Infof("haproxy helper bye")
				return
			case cmd := <-cmdChan:
				h.handleCmd(ctx, cmd)
			}
		}
	}
}

func (h *HaproxyHelper) handleCmd(ctx context.Context, cmd *LbagentCmd) {
	switch cmd.Type {
	case LbagentCmdUseCorpus:
		cmdData := cmd.Data.(*LbagentCmdUseCorpusData)
		defer cmdData.Wg.Done()
		h.handleUseCorpusCmd(ctx, cmd)
	case LbagentCmdStopDaemons:
		h.handleStopDaemonsCmd(ctx)
	default:
		log.Warningf("command type ignored: %v", cmd.Type)
	}
}

func (h *HaproxyHelper) handleStopDaemonsCmd(ctx context.Context) {
	pidFiles := []*agentutils.PidFile{
		h.gobetweenPidFile(),
		h.haproxyPidFile(),
		h.telegrafPidFile(),
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(pidFiles))

	for _, pf := range pidFiles {
		go func(pf *agentutils.PidFile) {
			defer wg.Done()
			proc, confirmed, err := pf.ConfirmOrUnlink()
			if confirmed {
				log.Infof("stopping %s(%d)", pf.Comm, proc.Pid)
				proc.Signal(syscall.SIGTERM)
				for etime := time.Now().Add(5 * time.Second); etime.Before(time.Now()); {
					if err := proc.Signal(syscall.Signal(0)); err == nil {
						return
					}
					time.Sleep(500 * time.Millisecond)
				}
				proc.Kill()
				// TODO check whether proc.Ppid == os.Getpid()
				proc.Wait()
			}
			if err != nil {
				log.Warningln(err.Error())
			}
		}(pf)
	}
	wg.Wait()
}

func (h *HaproxyHelper) handleUseCorpusCmd(ctx context.Context, cmd *LbagentCmd) {
	// haproxy config dir
	dir, err := h.configDirMan.NewDir(func(dir string) error {
		cmdData := cmd.Data.(*LbagentCmdUseCorpusData)
		corpus := cmdData.Corpus
		agentParams := cmdData.AgentParams
		{
			opt := fmt.Sprintf("stats socket %s expose-fd listeners", h.haproxyStatsSocketFile())
			agentParams.SetHaproxyParams("global_stats_socket", opt)
		}
		var genHaproxyConfigsResult *agentmodels.GenHaproxyConfigsResult
		var err error
		{
			// haproxy toplevel global/defaults config
			err = corpus.GenHaproxyToplevelConfig(dir, agentParams)
			if err != nil {
				err = fmt.Errorf("generating haproxy toplevel config failed: %s", err)
				return err
			}
		}
		{
			// haproxy configs
			genHaproxyConfigsResult, err = corpus.GenHaproxyConfigs(dir, agentParams)
			if err != nil {
				err = fmt.Errorf("generating haproxy config failed: %s", err)
				return err
			}
		}
		{
			// gobetween config
			opts := &agentmodels.GenGobetweenConfigOptions{
				LoadbalancersEnabled: genHaproxyConfigsResult.LoadbalancersEnabled,
				AgentParams:          agentParams,
			}
			err := corpus.GenGobetweenConfigs(dir, opts)
			if err != nil {
				err = fmt.Errorf("generating gobetween config failed: %s", err)
				return err
			}
		}
		{
			// keepalived config
			opts := &agentmodels.GenKeepalivedConfigOptions{
				LoadbalancersEnabled: genHaproxyConfigsResult.LoadbalancersEnabled,
				AgentParams:          agentParams,
			}
			log.Infof("GenKeepalivedConfigs %s", dir)
			err := corpus.GenKeepalivedConfigs(dir, opts)
			if err != nil {
				err = fmt.Errorf("generating keepalived config failed: %s", err)
				return err
			}
		}
		if agentParams.AgentModel.Params.Telegraf.InfluxDbOutputUrl != "" {
			agentParams.SetTelegrafParams("haproxy_input_stats_socket", h.haproxyStatsSocketFile())
			// telegraf config
			buf := bytes.NewBufferString("# yunion lb auto-generated telegraf.conf\n")
			tmpl := agentParams.TelegrafConfigTmpl
			err := tmpl.Execute(buf, agentParams.Data)
			if err == nil {
				d := buf.Bytes()
				p := filepath.Join(dir, "telegraf.conf")
				err := ioutil.WriteFile(p, d, agentutils.FileModeFile)
				if err == nil {
					err := h.reloadTelegraf(ctx, agentParams)
					if err != nil {
						log.Errorf("reloading telegraf.conf failed: %s", err)
					}
				} else {
					log.Errorf("writing %s failed: %s", p, err)
				}
			} else {
				log.Errorf("making telegraf.conf failed: %s, tmpl:\n%#v", err, tmpl)
			}
		}
		return nil
	})
	if err != nil {
		log.Errorf("making configs: %s", err)
		return
	}
	if err := h.configDirMan.Prune(h.opts.DataPreserveN); err != nil {
		log.Errorf("prune configs dir failed: %s", err)
		// continue
	}
	if err := h.useConfigs(ctx, dir); err != nil {
		log.Errorf("useConfigs: %s", err)
	}
}

func (h *HaproxyHelper) useConfigs(ctx context.Context, d string) error {
	lnF := func(old, new string) error {
		err := os.RemoveAll(new)
		if err != nil {
			return err
		}
		err = os.Symlink(old, new)
		return err
	}
	haproxyConfD := h.haproxyConfD()
	gobetweenJson := filepath.Join(h.opts.haproxyConfigDir, "gobetween.json")
	keepalivedConf := filepath.Join(h.opts.haproxyConfigDir, "keepalived.conf")
	telegrafConf := filepath.Join(h.opts.haproxyConfigDir, "telegraf.conf")
	dirMap := map[string]string{
		haproxyConfD:   d,
		gobetweenJson:  filepath.Join(d, "gobetween.json"),
		keepalivedConf: filepath.Join(d, "keepalived.conf"),
		telegrafConf:   filepath.Join(d, "telegraf.conf"),
	}
	for new, old := range dirMap {
		err := lnF(old, new)
		if err != nil {
			return err
		}
	}
	{
		var errs []error
		var err error
		{
			// reload haproxy
			err = h.reloadHaproxy(ctx)
			if err != nil {
				errs = append(errs, err)
			}
		}
		{
			// reload gobetween
			err = h.reloadGobetween(ctx)
			if err != nil {
				errs = append(errs, err)
			}
		}
		{
			// reload keepalived
			err = h.reloadKeepalived(ctx)
			if err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) == 0 {
			return nil
		}
		return errors.NewAggregate(errs)
	}
}

func (h *HaproxyHelper) haproxyConfD() string {
	return filepath.Join(h.opts.haproxyConfigDir, "haproxy.conf.d")
}

func (h *HaproxyHelper) haproxyPidFile() *agentutils.PidFile {
	pf := agentutils.NewPidFile(
		filepath.Join(h.opts.haproxyRunDir, "haproxy.pid"),
		"haproxy",
	)
	return pf
}

func (h *HaproxyHelper) haproxyStatsSocketFile() string {
	return filepath.Join(h.opts.haproxyRunDir, "haproxy.sock")
}

func (h *HaproxyHelper) reloadHaproxy(ctx context.Context) error {
	// NOTE we may sometimes need to specify a custom the executable path
	pidFile := h.haproxyPidFile()
	args := []string{
		h.opts.HaproxyBin,
		"-D", // goes daemon
		"-p", pidFile.Path,
		"-C", h.haproxyConfD(),
		"-f", h.haproxyConfD(),
	}
	proc, confirmed, err := pidFile.ConfirmOrUnlink()
	if !confirmed {
		log.Infof("starting haproxy")
		return h.runCmd(args)
	}
	if err != nil {
		log.Warningln(err.Error())
	}

	{
		// try reload
		args_ := make([]string, len(args))
		copy(args_, args)
		args_ = append(args_, "-sf", fmt.Sprintf("%d", proc.Pid))
		{
			statsSocket := h.haproxyStatsSocketFile()
			if fi, err := os.Stat(statsSocket); err == nil && fi.Mode()&os.ModeSocket != 0 {
				args_ = append(args_, "-x", statsSocket)
			} else {
				log.Warningf("stats socket %s not found", statsSocket)
			}
		}
		log.Infof("reloading haproxy")
		err := h.runCmd(args_)
		if err == nil {
			return nil
		}
		log.Errorf("reloading haproxy: %s", err)
	}
	{
		// reload failed
		// kill the old
		log.Errorf("killing old haproxy %d", proc.Pid)
		proc.Signal(syscall.SIGKILL)
		killed := false
	loop:
		for {
			timeout := time.NewTimer(3 * time.Second)
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			defer timeout.Stop()
			select {
			case <-ticker.C:
				if err := proc.Signal(syscall.Signal(0)); err != nil {
					killed = true
					break loop
				}
			case <-timeout.C:
				break loop
			}
		}
		if !killed {
			return fmt.Errorf("failed killing haproxy %d", proc.Pid)
		}
		log.Infof("restarting haproxy")
		return h.runCmd(args)
	}
}

func (h *HaproxyHelper) gobetweenConf() string {
	return filepath.Join(h.opts.haproxyConfigDir, "gobetween.json")
}

func (h *HaproxyHelper) gobetweenPidFile() *agentutils.PidFile {
	pf := agentutils.NewPidFile(
		filepath.Join(h.opts.haproxyRunDir, "gobetween.pid"),
		"gobetween",
	)
	return pf
}

func (h *HaproxyHelper) reloadGobetween(ctx context.Context) error {
	pidFile := h.gobetweenPidFile()
	{
		proc, confirmed, err := pidFile.ConfirmOrUnlink()
		if confirmed {
			log.Infof("stopping gobetween(%d)", proc.Pid)
			proc.Kill()
			proc.Wait()
		}
		if err != nil {
			log.Warningln(err.Error())
		}
	}

	args := []string{
		h.opts.GobetweenBin,
		"--config", h.gobetweenConf(),
		"--format", "json",
	}
	log.Infof("starting gobetween")
	cmd, err := h.startCmd(args)
	if err != nil {
		return err
	}
	err = agentutils.WritePidFile(cmd.Process.Pid, pidFile.Path)
	if err != nil {
		return fmt.Errorf("writing gobetween pid file: %s", err)
	}
	return nil
}

func (h *HaproxyHelper) telegrafConf() string {
	return filepath.Join(h.haproxyConfD(), "telegraf.conf")
}

func (h *HaproxyHelper) telegrafPidFile() *agentutils.PidFile {
	pf := agentutils.NewPidFile(
		filepath.Join(h.opts.haproxyRunDir, "telegraf.pid"),
		"telegraf",
	)
	return pf
}

func (h *HaproxyHelper) reloadTelegraf(ctx context.Context, agentParams *agentmodels.AgentParams) error {
	if h.opts.EnableRemoteExecutor {
		return h.remoteReloadTelegraf(ctx, agentParams)
	} else {
		return h.localReloadTelegraf(ctx)
	}
}

func (h *HaproxyHelper) remoteReloadTelegraf(ctx context.Context, agentParams *agentmodels.AgentParams) error {
	telegraf := system_service.GetService("telegraf")
	conf := map[string]interface{}{}
	conf["hostname"] = h.getHostname()
	conf["tags"] = map[string]string{
		"id":                                  h.lbagentId,
		"lbagent_id":                          h.lbagentId,
		"region":                              h.opts.Region,
		"lbagent_ip":                          h.opts.AccessIp,
		hostconsts.TELEGRAF_TAG_KEY_BRAND:     hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND,
		hostconsts.TELEGRAF_TAG_KEY_RES_TYPE:  hostconsts.TELEGRAF_TAG_ONECLOUD_RES_TYPE_LBAGENT,
		hostconsts.TELEGRAF_TAG_KEY_HOST_TYPE: hostconsts.TELEGRAF_TAG_ONECLOUD_HOST_TYPE_LBAGENT,
	}
	conf["nics"] = h.getNicsTelegrafConf()
	if len(agentParams.AgentModel.Params.Telegraf.InfluxDbOutputUrl) > 0 {
		conf[apis.SERVICE_TYPE_INFLUXDB] = map[string]interface{}{
			"url": []string{
				agentParams.AgentModel.Params.Telegraf.InfluxDbOutputUrl,
			},
			"database": agentParams.AgentModel.Params.Telegraf.InfluxDbOutputName,
		}
	}
	conf["haproxy"] = map[string]interface{}{
		"interval":          agentParams.AgentModel.Params.Telegraf.HaproxyInputInterval,
		"stats_socket_path": h.haproxyStatsSocketFile(),
	}
	oldConf := telegraf.GetConf()
	log.Debugf("old config: %s", oldConf)
	log.Debugf("new config: %s", conf)
	if gotypes.IsNil(oldConf) || !reflect.DeepEqual(oldConf, conf) {
		log.Debugf("telegraf config: %s", conf)
		telegraf.SetConf(conf)
		telegraf.BgReloadConf(conf)
	}
	return nil
}

func (h *HaproxyHelper) getNicsTelegrafConf() []map[string]interface{} {
	var ret = make([]map[string]interface{}, 0)
	phyNics, _ := sysutils.Nics()
	for _, pnic := range phyNics {
		ret = append(ret, map[string]interface{}{
			"name":  pnic.Dev,
			"speed": pnic.Speed,
		})
	}
	return ret
}

func (h *HaproxyHelper) getHostname() string {
	hn, err := os.Hostname()
	if err != nil {
		log.Fatalf("fail to get hostname %s", err)
		return ""
	}
	dotIdx := strings.IndexByte(hn, '.')
	if dotIdx >= 0 {
		hn = hn[:dotIdx]
	}
	hn = strings.ToLower(hn)
	if len(hn) == 0 {
		hn = "host"
	}
	masterIp := h.opts.AccessIp
	return hn + "-" + strings.Replace(masterIp, ".", "-", -1)
}

func (h *HaproxyHelper) localReloadTelegraf(ctx context.Context) error {
	pidFile := h.telegrafPidFile()
	{
		proc, confirmed, err := pidFile.ConfirmOrUnlink()
		if confirmed {
			log.Infof("stopping telegraf(%d)", proc.Pid)
			proc.Kill()
			proc.Wait()
		}
		if err != nil {
			log.Warningln(err.Error())
		}
	}
	log.Infof("starting telegraf")
	args := []string{
		h.opts.TelegrafBin,
		"--config", h.telegrafConf(),
	}
	cmd, err := h.startCmd(args)
	if err != nil {
		return err
	}
	err = agentutils.WritePidFile(cmd.Process.Pid, pidFile.Path)
	if err != nil {
		return fmt.Errorf("writing telegraf pid file: %s", err)
	}
	return nil
}

func (h *HaproxyHelper) keepalivedConf() string {
	return filepath.Join(h.opts.haproxyConfigDir, "keepalived.conf")
}

func (h *HaproxyHelper) keepalivedPidFile() *agentutils.PidFile {
	pf := agentutils.NewPidFile(
		filepath.Join(h.opts.haproxyRunDir, "keepalived.pid"),
		"keepalived",
	)
	return pf
}

func (h *HaproxyHelper) keepalivedVrrpPidFile() *agentutils.PidFile {
	pf := agentutils.NewPidFile(
		filepath.Join(h.opts.haproxyRunDir, "keepalived_vrrp.pid"),
		"keepalived",
	)
	return pf
}

func (h *HaproxyHelper) keepalivedCheckersPidFile() *agentutils.PidFile {
	pf := agentutils.NewPidFile(
		filepath.Join(h.opts.haproxyRunDir, "keepalived_checkers.pid"),
		"keepalived",
	)
	return pf
}

func (h *HaproxyHelper) reloadKeepalived(ctx context.Context) error {
	var (
		pidFile         *agentutils.PidFile
		vrrpPidFile     *agentutils.PidFile
		checkersPidFile *agentutils.PidFile
	)
	pidFile = h.keepalivedPidFile()
	{
		proc, confirmed, err := pidFile.ConfirmOrUnlink()
		if confirmed {
			// send SIGHUP to reload
			err := proc.Signal(syscall.SIGHUP)
			if err != nil {
				return fmt.Errorf("keepalived: send HUP failed: %s", err)
			}
			return nil
		}
		if err != nil {
			log.Warningln(err.Error())
		}
	}
	vrrpPidFile = h.keepalivedVrrpPidFile()
	if _, _, err := vrrpPidFile.ConfirmOrUnlink(); err != nil {
		log.Warningln(err.Error())
	}
	checkersPidFile = h.keepalivedCheckersPidFile()
	if _, _, err := checkersPidFile.ConfirmOrUnlink(); err != nil {
		log.Warningln(err.Error())
	}
	args := []string{
		h.opts.KeepalivedBin,
		"--pid", pidFile.Path,
		"--vrrp_pid", vrrpPidFile.Path,
		"--checkers_pid", checkersPidFile.Path,
		"--use-file", h.keepalivedConf(),
		"-D",
		"-d",
		"-S",
		"0",
	}
	return h.runCmd(args)
}

func (h *HaproxyHelper) runCmd(args []string) error {
	log.Debugf("run command %s", args)

	name := args[0]
	args = args[1:]
	cmd := exec.Command(name, args...)

	output, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			stdout := string(output)
			stderr := string(ee.Stderr)
			return fmt.Errorf("%s: %s\nargs: %s\nstdout: %s\nstderr: %s",
				name, err, strings.Join(args, " "), stdout, stderr)
		}
		return fmt.Errorf("%s: %s", name, err)
	}
	return nil
}

func (h *HaproxyHelper) startCmd(args []string) (*exec.Cmd, error) {
	name := args[0]
	args = args[1:]
	cmd := exec.Command(name, args...)

	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd, nil
}
