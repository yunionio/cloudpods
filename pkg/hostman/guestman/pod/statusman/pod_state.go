package statusman

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

var (
	statusManager IPodStatusManager
)

func init() {
	statusManager = newPodStatusManager()
}

func GetManager() IPodStatusManager {
	return statusManager
}

type IPodStatusManager interface {
	UpdateStatus(req *PodStatusUpdateRequest) error
	Start()
	Stop()
}

type ContainerStatus struct {
	Status         string
	RestartCount   int
	StartedAt      *time.Time
	LastFinishedAt *time.Time
}

type IPod interface {
	MarkContainerProbeDirty(ctrStatus string, ctrId string, reason string)
}

type PodStatusUpdateRequest struct {
	Id                string
	Pod               IPod
	Status            string
	ContainerStatuses map[string]*ContainerStatus
	Reason            string
	Result            chan error `json:"-"`
}

func (r PodStatusUpdateRequest) ToServerPerformStatusInput() *computeapi.ServerPerformStatusInput {
	powerState := computeapi.VM_POWER_STATES_OFF
	if r.Status == computeapi.VM_RUNNING {
		powerState = computeapi.VM_POWER_STATES_ON
	}
	guestStatus := &computeapi.ServerPerformStatusInput{
		PerformStatusInput: apis.PerformStatusInput{
			Status:      r.Status,
			PowerStates: powerState,
			Reason:      r.Reason,
		},
		Containers: make(map[string]*computeapi.ContainerPerformStatusInput),
	}
	for ctrId, ctrStatus := range r.ContainerStatuses {
		guestStatus.Containers[ctrId] = &computeapi.ContainerPerformStatusInput{
			PerformStatusInput: apis.PerformStatusInput{
				Status: ctrStatus.Status,
				Reason: r.Reason,
			},
			RestartCount:   ctrStatus.RestartCount,
			StartedAt:      ctrStatus.StartedAt,
			LastFinishedAt: ctrStatus.LastFinishedAt,
		}
	}

	return guestStatus
}

func (r PodStatusUpdateRequest) ToHostUploadGuestsStatusInput() *computeapi.HostUploadGuestsStatusInput {
	id := r.Id
	guestStatus := &computeapi.HostUploadGuestStatusInput{
		PerformStatusInput: apis.PerformStatusInput{
			Status: r.Status,
			Reason: r.Reason,
		},
		Containers: make(map[string]*computeapi.ContainerPerformStatusInput),
	}
	for ctrId, ctrStatus := range r.ContainerStatuses {
		guestStatus.Containers[ctrId] = &computeapi.ContainerPerformStatusInput{
			PerformStatusInput: apis.PerformStatusInput{
				Status: ctrStatus.Status,
				Reason: r.Reason,
			},
			RestartCount:   ctrStatus.RestartCount,
			StartedAt:      ctrStatus.StartedAt,
			LastFinishedAt: ctrStatus.LastFinishedAt,
		}
	}

	return &computeapi.HostUploadGuestsStatusInput{
		Guests: map[string]*computeapi.HostUploadGuestStatusInput{
			id: guestStatus,
		},
	}
}

type podStatusManager struct {
	updateChan chan *PodStatusUpdateRequest
	stopChan   chan struct{}
}

func newPodStatusManager() IPodStatusManager {
	return &podStatusManager{
		updateChan: make(chan *PodStatusUpdateRequest),
		stopChan:   make(chan struct{}),
	}
}

func (m *podStatusManager) Start() {
	go m.processLoop()
}

func (m *podStatusManager) Stop() {
	close(m.stopChan)
}

func (m *podStatusManager) UpdateStatus(req *PodStatusUpdateRequest) error {
	result := make(chan error, 1)
	req.Result = result
	m.updateChan <- req
	return <-result
}

func (m *podStatusManager) processLoop() {
	for {
		select {
		case <-m.stopChan:
			return
		case req := <-m.updateChan:
			err := m.handleUpdate(req)
			req.Result <- err
		}
	}
}

func (m *podStatusManager) handleUpdate(req *PodStatusUpdateRequest) error {
	input := req.ToServerPerformStatusInput()
	if _, err := hostutils.UpdateServerContainersStatus(context.Background(), req.Id, input); err != nil {
		return errors.Wrapf(err, "update server containers status")
	}
	for ctrId, ctrStatus := range input.Containers {
		// 同步容器状态可能会出现 probing 状态，所以需要 mark 成 dirty，等待 probe manager 重新探测容器状态
		req.Pod.MarkContainerProbeDirty(ctrStatus.Status, ctrId, input.Reason)
	}
	return nil
}
