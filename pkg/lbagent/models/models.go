package models

import (
	"yunion.io/x/onecloud/pkg/mcclient/models"
)

type IModel interface {
}

type Loadbalancer struct {
	*models.Loadbalancer

	listeners     LoadbalancerListeners
	backendGroups LoadbalancerBackendGroups
}

type LoadbalancerListener struct {
	*models.LoadbalancerListener

	loadbalancer *Loadbalancer
	certificate  *LoadbalancerCertificate
	rules        LoadbalancerListenerRules
}

type LoadbalancerListenerRule struct {
	*models.LoadbalancerListenerRule

	listener *LoadbalancerListener
}

type LoadbalancerBackendGroup struct {
	*models.LoadbalancerBackendGroup

	backends     LoadbalancerBackends
	loadbalancer *Loadbalancer
}

type LoadbalancerBackend struct {
	*models.LoadbalancerBackend
}

type LoadbalancerAcl struct {
	*models.LoadbalancerAcl
}

type LoadbalancerCertificate struct {
	*models.LoadbalancerCertificate
}
