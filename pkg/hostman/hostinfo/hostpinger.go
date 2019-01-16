package hostinfo

type SHostPingTask struct {
	interval int
}

func NewHostPingTask(interval int) *SHostPingTask {
	return &SHostPingTask{interval}
}

func (p *SHostPingTask) Start() {
	//TODO
}
