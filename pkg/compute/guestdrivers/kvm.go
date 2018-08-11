package guestdrivers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
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

func (self *SKVMGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	// TODO
	return nil
}

func (self *SKVMGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// TODO

	return nil
}

func (self *SKVMGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// TODO
	return nil
}

func (self *SKVMGuestDriver) RequestStartOnHost(guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	data := jsonutils.NewDict()
	// TODO
	return data, nil
}

func (self *SKVMGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// TODO
	return nil
}

func (self *SKVMGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	data := jsonutils.NewDict()
	// TODO
	return data, nil
}

func (self *SKVMGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// TODO
	return nil
}
