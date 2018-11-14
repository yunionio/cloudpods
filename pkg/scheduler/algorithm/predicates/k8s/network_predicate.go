package k8s

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/pkg/util/errors"

	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
)

const (
	// k8s annotations for create pod
	YUNION_CNI_NETWORK_ANNOTATION = "cni.yunion.io/network"
	YUNION_CNI_IPADDR_ANNOTATION  = "cni.yunion.io/ip"
)

type NetworkPredicate struct {
	network string
	ipAddr  string
}

func (p *NetworkPredicate) Clone() IPredicate {
	return &NetworkPredicate{}
}

func (p *NetworkPredicate) Name() string {
	return "network"
}

func (p *NetworkPredicate) PreExecute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) bool {
	net, netCont := pod.Annotations[YUNION_CNI_NETWORK_ANNOTATION]
	ipAddr, ipCont := pod.Annotations[YUNION_CNI_IPADDR_ANNOTATION]
	p.network = net
	p.ipAddr = ipAddr
	return netCont || ipCont
}

func (p *NetworkPredicate) Execute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) (bool, error) {
	hostNets := host.Networks
	if p.network != "" {
		err := p.checkByNetworks(hostNets)
		if err != nil {
			return false, err
		}
	}
	if p.ipAddr != "" {
		err := p.checkNetworksIP(p.ipAddr, hostNets)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (p NetworkPredicate) checkByNetworks(nets []*models.NetworkSchedResult) error {
	if len(nets) == 0 {
		return fmt.Errorf("Network is empty")
	}
	errs := make([]error, 0)
	for _, net := range nets {
		err := p.checkByNetwork(net)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (p NetworkPredicate) checkByNetwork(net *models.NetworkSchedResult) error {
	if net.Ports <= 0 {
		return fmt.Errorf("Network %s no free IPs", net.Name)
	}
	if !(p.network == net.Name || p.network == net.ID) {
		return fmt.Errorf("Network %s:%s or id not match %s", net.Name, net.ID, p.network)
	}
	return nil
}

func (p NetworkPredicate) checkNetworksIP(ip string, nets []*models.NetworkSchedResult) error {
	if len(nets) == 0 {
		return fmt.Errorf("Network is empty")
	}
	errs := make([]error, 0)
	for _, net := range nets {
		err := p.checkNetworkIP(ip, net)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (p NetworkPredicate) checkNetworkIP(ip string, net *models.NetworkSchedResult) error {
	if ok, err := net.ContainsIp(ip); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("Network %s not contains ip %s", net.Name, ip)
	}
	return nil
}
