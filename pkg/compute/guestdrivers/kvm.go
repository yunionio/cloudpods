package guestdrivers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/util/httputils"
)

type SKVMGuestDriver struct {
	SVirtualizedGuestDriver
}

func init() {
	driver := SKVMGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SKVMGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_KVM
}

func (self *SKVMGuestDriver) RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "GuestDetachAllDisksTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SKVMGuestDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	// guest.DeleteAllDisksInDB(ctx, userCred)
	// do nothing
	return nil
}

func findVNCPort(results string) int {
	reg := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):([\d]+)`)
	finds := reg.FindStringSubmatch(results)
	log.Debugf("finds=%s", finds)
	port, _ := strconv.Atoi(finds[2])
	return port
}

func (self *SKVMGuestDriver) GetGuestVncInfo(userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	url := fmt.Sprintf("/servers/%s/monitor", guest.Id)
	body := jsonutils.NewDict()
	var cmd string
	if guest.GetVdi() == "spice" {
		cmd = "info spice"
	} else {
		cmd = "info vnc"
	}
	body.Add(jsonutils.NewString(cmd), "cmd")
	ret, err := host.Request(userCred, "POST", url, nil, body)
	if err != nil {
		err = fmt.Errorf("Fail to request VNC info %s", err)
		log.Errorf(err.Error())
		return nil, err
	}
	results, _ := ret.GetString("results")
	if len(results) == 0 {
		err = fmt.Errorf("Can't get vnc information from host.")
		return nil, err
	}

	/***
	Server:
	address: 0.0.0.0:5901
	auth: none
	Client: none
		 ***/

	// info_vnc = result['results'].split('\n')
	// port = int(info_vnc[1].split(':')[-1].split()[0])

	port := findVNCPort(results)
	retval := jsonutils.NewDict()
	retval.Add(jsonutils.NewString(host.AccessIp), "host")
	retval.Add(jsonutils.NewString(guest.GetVdi()), "protocol")
	retval.Add(jsonutils.NewInt(int64(port)), "port")
	return retval, nil
}

func (self *SKVMGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	body := jsonutils.NewDict()
	params := task.GetParams()
	timeout, err := params.Int("timeout")
	if err != nil {
		timeout = 30
	}
	isForce, err := params.Bool("is_force")
	if isForce {
		timeout = 0
	}
	body.Add(jsonutils.NewInt(timeout), "timeout")

	header := http.Header{}
	header.Set("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")

	url := fmt.Sprintf("%s/servers/%s/stop", host.ManagerUri, guest.Id)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return err
	}
	return nil
}

func (self *SKVMGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	// ToDO
	return fmt.Errorf("Not Implement")
}

func (self *SKVMGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// ToDo
	return fmt.Errorf("Not Implement")
}

func (self *SKVMGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	header := http.Header{}
	header.Set("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")

	config := jsonutils.NewDict()
	desc := self.GetJsonDescAtHost(ctx, guest, host)
	config.Add(desc, "desc")
	params := task.GetParams()
	if params.Length() > 0 {
		config.Add(params, "params")
	}
	url := fmt.Sprintf("%s/servers/%s/start", host.ManagerUri, guest.Id)
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (self *SKVMGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	header := http.Header{}
	header.Set("X-Auth-Token", userCred.GetTokenString())
	header.Set("X-Region-Version", "v2")

	url := fmt.Sprintf("%s/servers/%s/status", host.ManagerUri, guest.Id)
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", url, header, nil, false)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (self *SKVMGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// ToDo
	return fmt.Errorf("Not Implement")
}

func (self *SKVMGuestDriver) RequestGuestCreateInsertIso(ctx context.Context, imageId string, guest *models.SGuest, task taskman.ITask) error {
	// ToDo
	return fmt.Errorf("Not Implement")
}

func (self *SKVMGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// ToDo
	return fmt.Errorf("Not Implement")
}

func (self *SKVMGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	desc := guest.GetDriver().GetJsonDescAtHost(ctx, guest, host)
	body := jsonutils.NewDict()
	body.Add(desc, "desc")
	if fw_only, _ := task.GetParams().Bool("fw_only"); fw_only {
		body.Add(jsonutils.JSONTrue, "fw_only")
	}
	url := fmt.Sprintf("/servers/%s/sync", guest.Id)
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}
