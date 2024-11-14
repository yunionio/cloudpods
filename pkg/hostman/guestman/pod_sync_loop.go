package guestman

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/guestman/pod/pleg"
	"yunion.io/x/onecloud/pkg/hostman/guestman/pod/runtime"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

func (m *SGuestManager) reconcileContainerLoop(cache runtime.Cache) {
	log.Infof("start reconcile container loop")
	for {
		m.Servers.Range(func(id, obj interface{}) bool {
			podObj, ok := obj.(*sPodGuestInstance)
			if !ok {
				return true
			}
			if podObj.isPodDirtyShutdown() {
				log.Infof("pod %s is dirty shutdown, using dirty shutdown manager to start it", podObj.GetName())
				return true
			}
			if err := m.reconcileContainer(podObj, cache); err != nil {
				log.Warningf("reconcile pod %s: %v", podObj.GetId(), err)
			}
			return true
		})
		time.Sleep(10 * time.Second)
	}
}

func (m *SGuestManager) reconcileContainer(obj *sPodGuestInstance, cache runtime.Cache) error {
	ps, err := cache.Get(obj.GetId())
	if err != nil {
		return errors.Wrapf(err, "get pod status")
	}
	getContainerStatus := func(name string) *runtime.Status {
		for i := range ps.ContainerStatuses {
			cs := ps.ContainerStatuses[i]
			if cs.Name == name {
				return cs
			}
		}
		return nil
	}
	ctrs := obj.GetContainers()
	var errs []error
	for i := range ctrs {
		ctr := ctrs[i]
		cs := getContainerStatus(ctr.Name)
		if cs == nil {
			// container is deleted
			continue
		}
		if cs.State == runtime.ContainerStateExited && cs.ExitCode != 0 {
			if err := m.startContainer(obj, ctr, cs); err != nil {
				errs = append(errs, errors.Wrapf(err, "start container %s", ctr.Name))
			}
		}
	}
	return errors.NewAggregate(errs)
}

func (m *SGuestManager) startContainer(obj *sPodGuestInstance, ctr *hostapi.ContainerDesc, cs *runtime.Status) error {
	_, isInternalStopped := obj.IsInternalStopped(cs.ID.ID)
	if isInternalStopped {
		return nil
	}
	finishedAt := ctr.StartedAt
	if !ctr.LastFinishedAt.IsZero() {
		finishedAt = ctr.LastFinishedAt
	}
	attempt := ctr.RestartCount
	step := 5 * time.Second
	internal := time.Duration(int(step) * (attempt * attempt))
	curInternal := time.Now().Sub(finishedAt)
	if curInternal < internal {
		log.Infof("current internal time (%s) < crash_back_off time (%s), skipping restart container(%s/%s)", curInternal, internal, obj.GetId(), ctr.Name)
		return nil
	} else {
		log.Infof("current internal time (%s | %s) > crash_back_off time (%s), restart container(%s/%s)", finishedAt, curInternal, internal, obj.GetId(), ctr.Name)
	}

	reason := fmt.Sprintf("start died container %s when exit code is %d", ctr.Id, cs.ExitCode)
	ctx := context.Background()
	userCred := hostutils.GetComputeSession(ctx).GetToken()
	if obj.ShouldRestartPodOnCrash() {
		// FIXME: 目前不用 workser 来后台异步运行 pod restart task
		// 这里异步运行会导致容器如果在 10s 没启动完成，又会进行新一轮排队
		// 所以改成同步串行执行
		//obj.RestartLocalPodAndContainers(ctx, userCred)
		newLocalPodRestartTask(ctx, userCred, obj).Run()
	} else {
		_, err := obj.StartLocalContainer(ctx, userCred, ctr.Id)
		if err != nil {
			return errors.Wrap(err, reason)
		} else {
			log.Infof("%s: start local container (%s/%s) success", reason, obj.GetId(), ctr.Name)
		}
	}
	return nil
}

func (m *SGuestManager) syncContainerLoop(plegCh chan *pleg.PodLifecycleEvent) {
	log.Infof("start sync container loop")
	for {
		m.syncContainerLoopIteration(plegCh)
	}
}

func (m *SGuestManager) syncContainerLoopIteration(plegCh chan *pleg.PodLifecycleEvent) {
	select {
	case e := <-plegCh:
		podMan := m.getPodByEvent(e)
		if podMan == nil {
			log.Warningf("can not find pod manager by %s", jsonutils.Marshal(e))
			return
		}
		if podMan.(*sPodGuestInstance).isPodDirtyShutdown() {
			log.Infof("pod %s is dirty shutdown, waiting it to started", podMan.GetName())
			return
		}
		podInstance := podMan.(*sPodGuestInstance)
		if e.Type == pleg.ContainerStarted {
			// 防止读取 podMan.GetCRIId 还没有刷新的问题
			podInstance.startPodLock.Lock()
			defer podInstance.startPodLock.Unlock()

			log.Infof("pod container started: %s", jsonutils.Marshal(e))
			ctrId := e.Data.(string)
			if ctrId == podMan.GetCRIId() {
				log.Infof("pod %s(%s) is started", podMan.GetId(), ctrId)
			} else {
				podMan.SyncStatus("pod container started")
			}
		}
		if e.Type == pleg.ContainerRemoved {
			/*isInternalRemoved := podMan.IsInternalRemoved(e)
			if !isInternalRemoved {
				log.Infof("pod container removed: %s, try recreated", jsonutils.Marshal(e))
			} else {
				log.Infof("pod container removed: %s", jsonutils.Marshal(e))
			}*/
			log.Infof("pod container removed: %s", jsonutils.Marshal(e))
		}
		if e.Type == pleg.ContainerDied {
			ctrId := e.Data.(string)
			ctr, isInternalStopped := podMan.IsInternalStopped(ctrId)
			if !isInternalStopped {
				podStatus, err := m.podCache.Get(e.Id)
				if err != nil {
					log.Errorf("get pod %s status error: %v", e.Id, err)
					return
				}
				log.Infof("pod container exited: %s", jsonutils.Marshal(e))
				// start container again
				ctrStatus := podStatus.GetContainerStatus(ctrId)
				var reason string
				if ctrStatus == nil {
					log.Errorf("can't get container %s status", ctrId)
					reason = "container not exist"
				} else {
					if ctrStatus.ExitCode == 0 {
						log.Infof("container %s exited", ctrId)
						reason = fmt.Sprintf("container %s exited", ctrId)
					} else {
						reason = fmt.Sprintf("exit code of died container %s is %d", ctr.Id, ctrStatus.ExitCode)
					}
				}
				log.Infof("sync pod %s container %s status: %s", e.Id, ctrId, reason)
				podMan.SyncStatus(reason)
			} else {
				log.Infof("pod container exited: %s", jsonutils.Marshal(e))
			}
		}
	}
}

func (m *SGuestManager) getPodByEvent(event *pleg.PodLifecycleEvent) PodInstance {
	obj, ok := m.GetServer(event.Id)
	if !ok {
		return nil
	}
	return obj.(PodInstance)
}

func (s *sPodGuestInstance) IsInternalStopped(ctrCriId string) (*ContainerExpectedStatus, bool) {
	ctr, ok := s.expectedStatus.Containers[ctrCriId]
	if !ok {
		return nil, true
	}
	if ctr.Status == computeapi.CONTAINER_STATUS_EXITED {
		return ctr, true
	}
	return ctr, false
}

func (s *sPodGuestInstance) IsInternalRemoved(ctrCriId string) bool {
	_, ok := s.expectedStatus.Containers[ctrCriId]
	if !ok {
		return true
	}
	return false
}

type ContainerExpectedStatus struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type PodExpectedStatus struct {
	lock       sync.RWMutex
	homeDir    string
	Status     string                              `json:"status"`
	Containers map[string]*ContainerExpectedStatus `json:"containers"`
}

func NewPodExpectedStatus(homeDir string, status string) (*PodExpectedStatus, error) {
	ps := &PodExpectedStatus{
		homeDir:    homeDir,
		Status:     status,
		Containers: make(map[string]*ContainerExpectedStatus),
	}
	if fileutils2.Exists(ps.getFilePath()) {
		content, err := fileutils2.FileGetContents(ps.getFilePath())
		if content == "" {
			return ps, nil
		}
		if err != nil {
			return nil, errors.Wrapf(err, "get %s content", ps.getFilePath())
		}
		obj, err := jsonutils.ParseString(content)
		if err != nil {
			return nil, errors.Wrapf(err, "parse %s content: %s", ps.getFilePath(), content)
		}
		if err := obj.Unmarshal(ps); err != nil {
			return nil, errors.Wrapf(err, "unmarshal to expected status %s", ps.getFilePath())
		}
	}
	return ps, nil
}

func (s *PodExpectedStatus) getFilePath() string {
	return filepath.Join(s.homeDir, "expected_status.json")
}

func (s *PodExpectedStatus) updateFile() error {
	content := jsonutils.Marshal(s).PrettyString()
	if err := fileutils2.FilePutContents(s.getFilePath(), content, false); err != nil {
		return errors.Wrapf(err, "put %s content: %s", s.getFilePath(), content)
	}
	return nil
}

func (s *PodExpectedStatus) SetStatus(status string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Status = status
	if err := s.updateFile(); err != nil {
		return errors.Wrapf(err, "update file")
	}
	return nil
}

func (s *PodExpectedStatus) SetContainerStatus(criId string, id string, status string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Containers[criId] = &ContainerExpectedStatus{
		Id:     id,
		Status: status,
	}
	if err := s.updateFile(); err != nil {
		return errors.Wrapf(err, "update file")
	}
	return nil
}

func (s *PodExpectedStatus) RemoveContainer(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.Containers, id)

	if err := s.updateFile(); err != nil {
		return errors.Wrapf(err, "update file")
	}
	return nil
}
