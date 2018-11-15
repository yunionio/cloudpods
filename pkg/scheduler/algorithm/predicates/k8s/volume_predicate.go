package k8s

import (
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
)

const (
	YUNION_CSI_STORAGECLASS = "csi-yunion"
)

type LocalVolumePredicate struct {
	pvcs []*v1.PersistentVolumeClaim
}

func (p *LocalVolumePredicate) Clone() IPredicate {
	return &LocalVolumePredicate{}
}

func (p *LocalVolumePredicate) Name() string {
	return "local-volume"
}

func (p *LocalVolumePredicate) PreExecute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) bool {
	if cli == nil {
		log.Errorf("k8s client is nil, not execute %s filter", p.Name())
		return false
	}
	pvcs := make([]*v1.PersistentVolumeClaim, 0)
	if pod.Spec.Volumes != nil && len(pod.Spec.Volumes) > 0 {
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim == nil {
				continue
			}
			pvcName := v.PersistentVolumeClaim.ClaimName
			pvc, err := cli.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(pvcName, metav1.GetOptions{})
			if err != nil {
				log.Warningf("Not found pvc %s for pod %s/%s", pvcName, pod.Namespace, pod.Name)
				return false
			}
			// pvc's StorageClassName must "csi-yunion"
			if pvc.Spec.StorageClassName != nil && *(pvc.Spec.StorageClassName) != YUNION_CSI_STORAGECLASS {
				continue
			}
			pvcs = append(pvcs, pvc)
		}
	}
	p.pvcs = pvcs
	return len(pvcs) != 0
}

func (p *LocalVolumePredicate) getPvcRequestSize(pvc *v1.PersistentVolumeClaim) int64 {
	req := pvc.Spec.Resources.Requests[v1.ResourceStorage]
	return req.Value()
}

func (p *LocalVolumePredicate) Execute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) (bool, error) {
	var reqSize int64
	for _, pvc := range p.pvcs {
		pvName := pvc.Spec.VolumeName
		if pvName != "" {
			pv, _ := cli.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
			if pv != nil {
				// PersistentVolume already exists
				log.V(10).Debugf("PV %s already exists", pv)
				continue
			}
		}
		reqSize += p.getPvcRequestSize(pvc)
	}
	reqSizeMB := reqSize / 1024 / 1024
	return p.canHostStorageCreateVol(host, reqSizeMB)
}

func (p *LocalVolumePredicate) canHostStorageCreateVol(host *candidate.HostDesc, reqSize int64) (bool, error) {
	freeSize := host.GetFreeStorageSizeOfType("local", false)
	log.Debugf("[host %s] PVC request %dMB, free %dMB", host.Name, reqSize, freeSize)
	if freeSize > reqSize {
		return true, nil
	}
	return false, fmt.Errorf("Out of local storage for volume: %d/%d (request/free)MB", reqSize, freeSize)
}
